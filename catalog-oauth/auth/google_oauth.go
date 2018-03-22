/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Package auth contains functions to read secrets that contain Google OAuth
information, generate oauth2.Tokens using that information, and subsequently
writes new secrets which contain opaque bearer tokens in the way that the
Kubernetes service catalog
(http://github.com/kubernetes-incubator/service-catalog) understands.
*/

package auth

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
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
		return nil, fmt.Errorf("error creating JWT Config: %v", err)
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
	secret := readSecret(oauthSecret)
	if secret == nil {
		return nil // secret was not recognized as Service Catalog authentication extension
	}
	token, err := Token(ctx, secret.privateKey, secret.scopes...)
	if err != nil {
		return fmt.Errorf("error creating an OAuth access token for secret %s/%s: %v", oauthSecret.Namespace, oauthSecret.Name, err)
	}
	return writeSecret(core, token.AccessToken, secret.secretNamespace, secret.secretName)
}

// readSecret parses the given secret and converts it to googleOAuthSecret.
func readSecret(secret *v1.Secret) *GoogleOAuthSecret {
	if secret.Data == nil {
		glog.Infof("Secret '%s/%s' is not compatible with the Service Catalog authentication extension contract (missing 'data' field); skipping...", secret.Namespace, secret.Name)
		return nil
	}
	privateKey, ok := secret.Data[keyKey]
	if !ok {
		glog.Infof("Secret '%s/%s' is not compatible with the Service Catalog authentication extension contract (missing '%s' field); skipping...", secret.Namespace, secret.Name, keyKey)
		return nil
	}
	secretName, ok := secret.Data[secretNameKey]
	if !ok {
		glog.Infof("Secret '%s/%s' is not compatible with the Service Catalog authentication extension contract (missing '%s' field); skipping...", secret.Namespace, secret.Name, secretNameKey)
		return nil
	}
	secretNamespace, ok := secret.Data[secretNamespaceKey]
	if !ok {
		glog.Infof("Secret '%s/%s' is not compatible with the Service Catalog authentication extension contract (missing '%s' field); skipping...", secret.Namespace, secret.Name, secretNamespaceKey)
		return nil
	}

	var scopes []string
	scopesBytes, ok := secret.Data[scopesKey]
	if ok {
		if err := json.Unmarshal(scopesBytes, &scopes); err != nil {
			glog.Errorf("Secret %s/%s appears compatible with Service Catalog authentication extension contract, but the value in '%s' field could not be unmarshalled:\n%s\n%v",
				secret.Namespace, secret.Name, scopesKey, string(scopesBytes), err)
			return nil
		}
	}

	return &GoogleOAuthSecret{
		privateKey:      privateKey,
		scopes:          scopes,
		secretName:      string(secretName),
		secretNamespace: string(secretNamespace),
	}
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
			return fmt.Errorf("failed to create a new OAuth credentials secret %s/%s: %v", namespace, name, err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to read a pre-existing OAuth credentials secret %s/%s: %v", namespace, name, err)
	} else {
		// Secret already exists so update it.
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}
		secret.Data[tokenKey] = []byte(token)
		if secret, err = secretSpace.Update(secret); err != nil {
			return fmt.Errorf("failed to save an OAuth access token into a secret %s/%s: %v", namespace, name, err)
		}
	}

	glog.Infof("Successfully wrote an OAuth access token into secret %s/%s", namespace, name)
	return nil
}
