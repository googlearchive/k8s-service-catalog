// Package auth contains functions to read secrets that contain Google OAuth
// information, generate oauth2.Tokens using that information, and subsequently
// writes new secrets which contain opaque bearer tokens in the way that the
// Kubernetes service catalog
// (http://github.com/kubernetes-incubator/service-catalog) understands.
package auth

import (
	"context"
	"encoding/json"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	keyKey             = "key"
	secretNameKey      = "secretName"
	secretNamespaceKey = "secretNamespace"
	scopesKey          = "scopes"
	tokenKey           = "token"
)

// GoogleOAuthSecret has the information that we expect to parse from the Secret
// which we will use to generate an access token.
type GoogleOAuthSecret struct {
	// private key information (as json)
	privateKey []byte
	// scopes to request authorization for
	scopes []string
	// name of secret which will contain token
	secretName string
	// namespace of secret which will contain token
	secretNamespace string
}

// Token returns an oauth2.Token given the privateKey and scopes.
func Token(ctx context.Context, privateKey []byte, scopes ...string) (*oauth2.Token, error) {
	jwt, err := google.JWTConfigFromJSON(privateKey, scopes...)
	if err != nil {
		return nil, fmt.Errorf("error creating JWT Config: %v: ", err)
	}

	token, err := jwt.TokenSource(ctx).Token()
	if err != nil {
		return nil, fmt.Errorf("error getting token: %v", err)
	}
	return token, nil
}

// WriteTokenSecret will, given a Kubernetes core object and a
// *v1.Secret (oauthSecret) which contains the service account information,
// write a new secret with the opaque token that can be used with the Kubernetes
// service catalog.
// oauthSecret should contain the fields:
// - "key": which has the Google service account JWT information
// - "secretName": which is the desired name of the secret we will create here
// - "secretNamespace: the desired namespace of the secret this function creates
// - "scopes": the scopes of the access token
func WriteTokenSecret(ctx context.Context, core corev1.CoreV1Interface, oauthSecret *v1.Secret) error {
	secret, err := readSecret(oauthSecret)
	if err != nil {
		return fmt.Errorf("error reading secret %s: %v", oauthSecret.Name, err)
	}
	token, err := Token(ctx, secret.privateKey, secret.scopes...)
	if err != nil {
		return fmt.Errorf("error getting token for secret %s: %v", oauthSecret.Name, err)
	}
	if err := writeSecret(core, token.AccessToken, secret.secretNamespace, secret.secretName); err != nil {
		return fmt.Errorf("error writing secret %s in namespace %s: %v", secret.secretName, secret.secretNamespace, err)
	}
	return nil
}

// readSecret parses the given secret and converts it to googleOAuthSecret.
func readSecret(secret *v1.Secret) (*GoogleOAuthSecret, error) {
	if secret.Data == nil {
		return nil, fmt.Errorf("secret %s has no secret.Data field", secret.Name)
	}
	privateKey, ok := secret.Data[keyKey]
	if !ok {
		return nil, fmt.Errorf("secret %s has no %s in Data", secret.Name, keyKey)
	}
	var scopes []string
	scopesBytes, ok := secret.Data[scopesKey]
	if ok {
		if err := json.Unmarshal(scopesBytes, &scopes); err != nil {
			return nil, fmt.Errorf("secret %s has %s in Data but it could not be unmarshalled", secret.Name, scopesKey)
		}
	}

	secretName, ok := secret.Data[secretNameKey]
	if !ok {
		return nil, fmt.Errorf("secret %s has no %s in Data", secret.Name, secretNameKey)
	}

	secretNamespace, ok := secret.Data[secretNamespaceKey]
	if !ok {
		return nil, fmt.Errorf("secret %s has no %s in Data", secret.Name, secretNamespaceKey)
	}

	return &GoogleOAuthSecret{
		privateKey:      privateKey,
		scopes:          scopes,
		secretName:      string(secretName),
		secretNamespace: string(secretNamespace),
	}, nil
}

// writeSecret writes/updates a secret with given name and namespace to have the
// token as an opaque bearer token in a way the service catalog understands.
func writeSecret(core corev1.CoreV1Interface, token, namespace, name string) error {
	secretSpace := core.Secrets(namespace)
	secret, err := secretSpace.Get(name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		// If secret is not made then create it.
		data := make(map[string][]byte)
		data[tokenKey] = []byte(token)
		secret = &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Data: data,
		}
		if secret, err = secretSpace.Create(secret); err != nil {
			return fmt.Errorf("error creating secret: %v", err)
		}
	} else if err != nil {
		return fmt.Errorf("error getting secret: %v", err)
	} else {
		// Secret already exists so update it.
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}
		secret.Data[tokenKey] = []byte(token)
		if secret, err = secretSpace.Update(secret); err != nil {
			return fmt.Errorf("error updating secret: %v", err)
		}
	}
	return nil
}
