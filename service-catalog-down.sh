#!/bin/bash
##################################################################
#
# Author: Sean Sullivan (seans)
# Date:   08/02/2017
# Description: Helper script to bring down service catalog
#   cleanly.
#
##################################################################

SVC_ROOT=$(dirname "${BASH_SOURCE}")/..
SVC_DIR=${SVC_ROOT}/config

NAMESPACE=service-catalog
DELETE_VOLUME=false

echo "Uninstalling Service Catalog..."
echo

# Delete apiserver and controller-manager deployments, as well as
# the etcd stateful set.
kubectl delete -f ${SVC_DIR}/apiserver-deployment.yaml
kubectl delete -f ${SVC_DIR}/controller-manager-deployment.yaml
kubectl delete -f ${SVC_DIR}/tls-cert-secret.yaml
kubectl delete -f ${SVC_DIR}/etcd-svc.yaml
kubectl delete -f ${SVC_DIR}/etcd.yaml
# TODO: delete persistent volume if flag set

# Delete service and api registration
kubectl delete -f ${SVC_DIR}/api-registration.yaml
kubectl delete -f ${SVC_DIR}/service.yaml

# Delete roles and bindings
kubectl delete -f ${SVC_DIR}/rbac.yaml
kubectl delete -f ${SVC_DIR}/service-accounts.yaml

# Delete the namespace
kubectl delete -f ${SVC_DIR}/namespace.yaml

# Delete generated YAML files
echo
echo "Deleting generated YAML files"
rm -f ${SVC_DIR}/tls-cert-secret.yaml
rm -f ${SVC_DIR}/api-registration.yaml

echo
echo "Uninstalling Service Catalog...COMPLETE"
