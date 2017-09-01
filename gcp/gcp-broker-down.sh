#!/bin/bash
##################################################################
#
# Author: Sean Sullivan (seans)
# Date:   08/07/2017
# Description:
#   Cleans up the Google Cloud Platform broker artifacts in
# the service catalog.
#
##################################################################

SVC_ROOT=$(dirname "${BASH_SOURCE}")/../..
SVC_DIR=${SVC_ROOT}/config/gcp

echo "Uninstalling GCP Broker..."
echo

# Delete the broker resource.
echo "Removing GCP Broker Resource"
kubectl delete -f ${SVC_DIR}/gcp-broker.yaml

echo "Removing Google OAuth Secret Tranformer"
kubectl delete -f ${SVC_DIR}/google-oauth-deployment.yaml

# Delete service account secret
echo "Deleting GCP Broker Service Account Secret"
kubectl delete -f ${SVC_DIR}/service-account-secret.yaml

echo "Deleting Namespace for GCP Broker Service Account Secret"
kubectl delete -f ${SVC_DIR}/google-oauth-namespace.yaml

echo "Removing Service Catalog controller manager permissions"
kubectl delete -f ${SVC_DIR}/rbac.yaml

# Clean up generated YAML files.
echo
echo "Deleting generated YAML files"
rm -f ${SVC_DIR}/service-account-secret.yaml

echo "Deleting GCP Broker Namespace for Instances and Bindings"
kubectl delete -f ${SVC_DIR}/gcp-instance-namespace.yaml

echo
echo "Uninstalling GCP Broker...COMPLETE"
