# Overview

This is a program/container that will create and manage secrets which contain
opaque tokens to use within the Kubernetes service catalog. It creates these
opaque tokens by reading from secrets that contain service account information.

The program will watch a namespace, "google-oauth" by default, and read secrets
from there as they are added and also periodically (currently every 10min) to
refresh the tokens.

The secrets are expected to have four fields:
* "key" which has the service account json private key
* "scopes" the scopes which the OAuth credentials will request
* "secretName" the name of the secret which will contain the token
* "secretNamespace" the namespace of the secret which will contain the token

For example, you can have a secret that looks like

```
apiVersion: v1
kind: Secret
metadata:
  name: oauth
  namespace: google-oauth
type: Opaque
data:
  key: bm90YXJlYWxrZXk=
  scopes: WyJodHRwczovL3d3dy5nb29nbGVhcGlzLmNvbS9hdXRoL2Nsb3VkLXBsYXRmb3JtIl0=
  secretName: b2F1dGg=
  secretNamespace: ZGVmYXVsdA==
```

and that will create a secret with name "oauth" in namespace "default". Note
that the secretName and secretNamespace are base64 encoded as is required for
Kubernetes secret values.

# Build and Deploy

You can build with

```
IMAGE=<IMAGE_TAG> make
```

push to gcr with

```
gcloud docker -a
docker push <tag>
```

and deploy with

```
kubectl run goauth --image=<tag>
```
