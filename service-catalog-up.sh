#!/bin/bash
##################################################################
#
# Author: Sean Sullivan (seans)
# Date:   08/02/2017
# Description: Helper script to bring up service catalog using
# the YAML files in this directory. The flow is basically:
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

SVC_ROOT=$(dirname "${BASH_SOURCE}")/..
SVC_DIR=${SVC_ROOT}/config

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
