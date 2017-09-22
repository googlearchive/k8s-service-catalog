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
# Prerequisites:
#   Assumes k8s cluster running (GCE or GKE).
#   Assumes gcloud config pointing to project running k8s.
#   Assumes service catalog is already running in k8s cluster.
# Description:
#   Connects service catalog to GCP broker by creating the
# broker resource in service catalog.
#   *) Enabling GCP API's
#   *) Create GCP service account
#   *) Create GCP service account key
#   *) Store the created key in a secret
#   *) Create broker resource using secret
#   *) Create namespace for future instance provisions
#      and bindings.
#   *) Grant permission to controller manager in the
#      namespace mentioned above.
# At the end of this script, you should be able to run:
#   kubectl get brokers <broker-name> -n <namespace> -o yaml
# ..and have a status=Ready
#
##################################################################

# Verify that the environment variable PROJECT_ID is set.
if [ -z "$PROJECT_ID" ]; then
  "PROJECT_ID environment variable not set...exiting"
  exit 1
fi

SVC_ROOT=$(dirname "${BASH_SOURCE}")
SVC_DIR=${SVC_ROOT}

SVC_ACCT=service-catalog-gcp
FULL_SVC_ACCT=${SVC_ACCT}@${PROJECT_ID}.iam.gserviceaccount.com

echo "Service Dir: $SVC_DIR"

echo "GCloud Project: $PROJECT_ID"

# Enable the GCP API's we're going to use for the broker.
# TODO: ENABLE DM API
# TODO: GRANT ROLES/OWNER on DM Service Account
GCLOUD_ENABLED_APIS=$(gcloud service-management list --enabled)
# BROKER_API_HOST=staging-servicebroker.sandbox.googleapis.com
BROKER_API_HOST=servicebroker.googleapis.com
if echo "$GCLOUD_ENABLED_APIS" | grep -q $BROKER_API_HOST; then
  echo "GCloud Project API already enabled: $BROKER_API_HOST"
else
  echo "Enabling: $BROKER_API_HOST"
  gcloud service-management enable $BROKER_API_HOST
fi
# REGISTRY_API_HOST=staging-serviceregistry.sandbox.googleapis.com
REGISTRY_API_HOST=serviceregistry.googleapis.com
if echo "$GCLOUD_ENABLED_APIS" | grep -q $REGISTRY_API_HOST; then
  echo "GCloud Project API already enabled: $REGISTRY_API_HOST"
else
  echo "Enabling: $REGISTRY_API_HOST"
  gcloud service-management enable $REGISTRY_API_HOST
fi

# Create the service account if it doesn't already exists.
SVC_ACCT_EXISTS=$(gcloud beta iam service-accounts list)
if echo "$SVC_ACCT_EXISTS" | grep -q $FULL_SVC_ACCT; then
  echo "Service Account Exists: $FULL_SVC_ACCT"
else
  # Create the GCP service account used to communicate with broker.
  echo "Creating Service Account: $FULL_SVC_ACCT"
  gcloud beta iam service-accounts create $SVC_ACCT \
    --display-name "Service Catalog Account"

  # Add necessary editor role for this service account.
  # TODO: SHOULD ONLY NEED roles/serviceBroker.operator
  echo "Adding Editor Role to Service Account"
  gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member serviceAccount:${FULL_SVC_ACCT} \
    --role roles/editor
    # --role roles/serviceBroker.operator
fi

# Generate key for service account and store in secret.
# TODO: DOWN NEEDS TO CLEAN UP JSON KEY
# TODO: SPECIFY KEY NAME + RANDOM -> more descriptive.
echo "Generating Key for GCP Service Account"
# Create a key for this service account.
tmpFile=$(mktemp)
gcloud beta iam service-accounts keys create \
      --iam-account $FULL_SVC_ACCT $tmpFile
SVC_ACCOUNT_KEY=$(base64 --wrap 0 $tmpFile)

echo "Creating Namespace for GCP Service Account Key"
kubectl create -f ${SVC_DIR}/google-oauth-namespace.yaml

echo "Creating Secret from GCP Service Account Key"
sed -e 's|SVC_ACCOUNT_KEY|'"${SVC_ACCOUNT_KEY}"'|g' \
    ${SVC_DIR}/service-account-secret.yaml.tmpl > ${SVC_DIR}/service-account-secret.yaml
kubectl create -f ${SVC_DIR}/service-account-secret.yaml

echo "Creating Google OAuth Secret Tranformer"
kubectl create -f ${SVC_DIR}/google-oauth-deployment.yaml

echo "Creating GCP Broker Namespace for Instances and Bindings"
kubectl create -f ${SVC_DIR}/gcp-instance-namespace.yaml

echo "Granting permission to Service Catalog controller manager"
kubectl create -f ${SVC_DIR}/rbac.yaml

echo "Connecting to GCP Broker"
kubectl create -f ${SVC_DIR}/gcp-broker.yaml

