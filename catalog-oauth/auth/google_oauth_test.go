package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"testing"
)

func TestReadSecret(t *testing.T) {
	for _, testData := range []struct {
		expectedErr bool
		secret      *v1.Secret
	}{
		{
			expectedErr: true,
			secret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "NoData",
					Namespace: "secretNamespace",
				},
				Data: nil,
			},
		}, {
			expectedErr: true,
			secret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "NoKey",
					Namespace: "secretNamespace",
				},
				Data: map[string][]byte{
					secretNameKey:      []byte("otherSecretName"),
					secretNamespaceKey: []byte("otherSecretNamespace"),
					scopesKey:          []byte("[]"),
				},
			},
		}, {
			expectedErr: true,
			secret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "NoSecretName",
					Namespace: "secretNamespace",
				},
				Data: map[string][]byte{
					keyKey:             []byte("alsdjfkafkasjakfksajfas"),
					secretNamespaceKey: []byte("otherSecretNamespace"),
					scopesKey:          []byte("[]"),
				},
			},
		}, {
			expectedErr: true,
			secret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "NoSecretNamespace",
					Namespace: "secretNamespace",
				},
				Data: map[string][]byte{
					keyKey:        []byte("alsdjfkafkasjakfksajfas"),
					secretNameKey: []byte("otherSecretName"),
					scopesKey:     []byte("[]"),
				},
			},
		}, {
			expectedErr: false,
			secret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "NoScopes",
					Namespace: "secretNamespace",
				},
				Data: map[string][]byte{
					keyKey:             []byte("alsdjfkafkasjakfksajfas"),
					secretNameKey:      []byte("otherSecretName"),
					secretNamespaceKey: []byte("otherSecretNamespace"),
				},
			},
		}, {
			expectedErr: false,
			secret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "Valid",
					Namespace: "secretNamespace",
				},
				Data: map[string][]byte{
					keyKey:             []byte("alsdjfkafkasjakfksajfas"),
					secretNameKey:      []byte("otherSecretName"),
					secretNamespaceKey: []byte("otherSecretNamespace"),
					scopesKey:          []byte("[\"cloud.google.com\"]"),
				},
			},
		},
	} {
		secret, err := readSecret(testData.secret)
		if !testData.expectedErr {
			if err != nil {
				t.Errorf("unexpected err for test case %s: %v", testData.secret.Name, err)
			} else {
				if !bytes.Equal(secret.privateKey, testData.secret.Data[keyKey]) {
					t.Errorf("testcase %s %s did not match: expected %v but got %v", testData.secret.Name, keyKey, testData.secret.Data[keyKey], secret.privateKey)
				} else if marshalledScopes, err := json.Marshal(secret.scopes); err != nil && !bytes.Equal(marshalledScopes, testData.secret.Data[scopesKey]) {
					t.Errorf("testcase %s %s did not match: expected %s but got %s", testData.secret.Name, scopesKey, string(testData.secret.Data[scopesKey]), string(marshalledScopes))
				} else if secretName := string(testData.secret.Data[secretNameKey]); secretName != secret.secretName {
					t.Errorf("testcase %s %s did not match: expected %s but got %s", testData.secret.Name, secretNameKey, secretName, secret.secretName)
				} else if secretNamespace := string(testData.secret.Data[secretNamespaceKey]); secretNamespace != secret.secretNamespace {
					t.Errorf("testcase %s %s did not match: expected %s but got %s", testData.secret.Name, secretNamespaceKey, secretNamespace, secret.secretNamespace)
				}
			}
		} else if err == nil {
			t.Errorf("Expected err but none was found for test case %s", testData.secret.Name)
		}
	}
}

func TestToken(t *testing.T) {
	privateKey, err := ioutil.ReadFile("test/test0.json")
	if err != nil {
		t.Fatalf("error reading test0 json file: %v", err)
	}
	ctx := context.Background()
	token, err := Token(ctx, privateKey, "https://www.googleapis.com/auth/cloud-platform.read-only")
	if err != nil || token == nil || token.AccessToken == "" {
		t.Fatalf("error getting access token or it's empty: %v", err)
	}
}

func TestWriteSecret(t *testing.T) {
	name, namespace := "test_name", "test_namespace"
	core := fake.NewSimpleClientset().Core()

	// these were real tokens at some point
	token := "ya29.ElqTBACeQOIIo3S1redKwTpmq2keYG4SCBBjfFmsuoIYbKa5kiiKy6EdE_bIfz3rupqu-2ZftawnYOEuDdfONGv0-9NQpoNyLXLtj2cK05owHl22wxHUP11Aup0"
	if err := writeAndCheck(core, token, namespace, name); err != nil {
		t.Fatal(err)
	}
	//test update
	token = "ya29.ElqUBLRjGNVDZce93yvjBqUE6Cnj6vMERzfL-DrfAvH_KBujZCaciD2mRVurLCMGxUiaFaHYjsM2Oj34tNlfVBXKnhmGUKLJY816feu0-RF7uBtT3hoHmTlUdrc"
	if err := writeAndCheck(core, token, namespace, name); err != nil {
		t.Fatal(err)
	}
}

func TestWriteTokenSecret(t *testing.T) {
	const (
		opaqueSecretName      = "opaqueSecretName"
		opaqueSecretNamespace = "opaqueSecretNamespace"
		secretName            = "secret"
		jsonFilename          = "test/test0.json"
	)
	core := fake.NewSimpleClientset().Core()
	ctx := context.Background()

	secret, err := secretFromFile(jsonFilename, secretName, opaqueSecretNamespace, opaqueSecretName)
	WriteTokenSecret(ctx, core, secret)

	_, err = core.Secrets(opaqueSecretNamespace).Get(opaqueSecretName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("error getting opaque token secret: %v", err)
	}
}

func secretFromFile(privateKeyFilename, secretName, opaqueSecretNamespace, opaqueSecretName string) (*v1.Secret, error) {
	const secretNamespace = "google-oauth"
	privateKey, err := ioutil.ReadFile(privateKeyFilename)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %v", privateKeyFilename, err)
	}
	data := make(map[string][]byte)
	data[keyKey] = privateKey
	data[scopesKey] = []byte("[\"https://www.googleapis.com/auth/cloud-platform.read-only\"]")
	data[secretNameKey] = []byte(opaqueSecretName)
	data[secretNamespaceKey] = []byte(opaqueSecretNamespace)

	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: secretNamespace,
		},
		Data: data,
	}, nil
}

func writeAndCheck(core corev1.CoreV1Interface, token, namespace, name string) error {
	err := writeSecret(core, token, namespace, name)
	if err != nil {
		return fmt.Errorf("error writing secret: %v", err)
	}

	secret, err := core.Secrets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting secret: %v", err)
	}
	if string(secret.Data["token"]) != token {
		return fmt.Errorf("token in secret is %q but expected %q", string(secret.Data["token"]), token)
	}
	return nil
}
