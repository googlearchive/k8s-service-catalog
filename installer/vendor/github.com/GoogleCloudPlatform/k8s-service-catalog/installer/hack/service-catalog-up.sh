#!/bin/bash
##################################################################
# Copyright 2017 Google Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Helper script to bring up service catalog using the YAML files
# in this directory. The flow is basically:
#   *) Create namespace
#   *) Create service account, roles, and bindings
#   *) Create service and register
#   *) Create the etcd stateful set and service
#   *) Deploy apiserver and controller manager.
# After this script is run, the user should be able to issue
# the command: kubectl api-versions
# and see "servicecatalog.k8s.io/v1alpha1" listed.
#
##################################################################

SVC_ROOT=$(dirname "${BASH_SOURCE}")
SVC_DIR=${SVC_ROOT}

# Create the TLS key files for secure communication between
# main APIServer and service catalog.
TEMP_DIR=$(mktemp -d)
${SVC_DIR}/make-keys.sh $TEMP_DIR

CA_PUBLIC_KEY=$(base64 --wrap 0 $TEMP_DIR/ca.pem)
SVC_PUBLIC_KEY=$(base64 --wrap 0 $TEMP_DIR/apiserver.pem)
SVC_PRIVATE_KEY=$(base64 --wrap 0 $TEMP_DIR/apiserver-key.pem)

# Update the yaml files that use the TLS key data.
sed -e 's|CA_PUBLIC_KEY|'"${CA_PUBLIC_KEY}"'|g' \
    ${SVC_DIR}/api-registration.yaml.tmpl > ${SVC_DIR}/api-registration.yaml
sed -e 's|SVC_PUBLIC_KEY|'"${SVC_PUBLIC_KEY}"'|g' \
    -e 's|SVC_PRIVATE_KEY|'"${SVC_PRIVATE_KEY}"'|g' \
    ${SVC_DIR}/tls-cert-secret.yaml.tmpl > ${SVC_DIR}/tls-cert-secret.yaml

echo $TEMP_DIR
# Create namespace for service catalog resources
kubectl create -f ${SVC_DIR}/namespace.yaml

# Create service accounts, roles, and bindings
kubectl create -f ${SVC_DIR}/service-accounts.yaml
kubectl create -f ${SVC_DIR}/rbac.yaml

# Create service and register
kubectl create -f ${SVC_DIR}/service.yaml
kubectl create -f ${SVC_DIR}/api-registration.yaml

# Create the etcd stateful set and service
kubectl create -f ${SVC_DIR}/etcd.yaml
kubectl create -f ${SVC_DIR}/etcd-svc.yaml

# Deploy apiserver and controller manager
kubectl create -f ${SVC_DIR}/tls-cert-secret.yaml
kubectl create -f ${SVC_DIR}/apiserver-deployment.yaml
kubectl create -f ${SVC_DIR}/controller-manager-deployment.yaml
