#!/usr/bin/env bash
########################################################################
#
# Copyright 2017 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
########################################################################

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

########################################################################
# Global Variables
########################################################################

# Service catalog namespace
SC_NAMESPACE="service-catalog"

# Repository
REPO=github.com/GoogleCloudPlatform/k8s-service-catalog
REPO_DIR=${GOPATH}/src/${REPO}

########################################################################
# Functions
########################################################################

# Procedure that waits until all pods in a namespace are "Running"
# Parameter: $1 - pod namespace to check
wait_for_ready() {

  NAMESPACE=$1
  if [ -z $NAMESPACE ];then
    echo "ERROR: No namespace specified"
    return 1
  fi

  # Ensure all pods in the namespace entered a Running state
  SUCCESS=0
  PODS_FOUND=0
  POD_RETRY_COUNT=0
  RETRY=18
  RETRY_DELAY=10
  while [ "$POD_RETRY_COUNT" -lt "$RETRY" ]; do
    POD_STATUS=`kubectl get pods --no-headers --namespace $NAMESPACE`
    if [ -z "$POD_STATUS" ];then
      echo "INFO: No pods found for this release, retrying after sleep"
      POD_RETRY_COUNT=$((POD_RETRY_COUNT+1))
      sleep $RETRY_DELAY
      continue
    else
      PODS_FOUND=1
    fi

    if ! echo "$POD_STATUS" | grep -v Running;then
      echo "INFO: All pods entered the Running state"

      CONTAINER_RETRY_COUNT=0
      while [ "$CONTAINER_RETRY_COUNT" -lt "$RETRY" ]; do
        UNREADY_CONTAINERS=`kubectl get pods --namespace $NAMESPACE \
          -o jsonpath="{.items[*].status.containerStatuses[?(@.ready!=true)].name}"`
        if [ -n "$UNREADY_CONTAINERS" ];then
          echo "INFO: Some containers are not yet ready; retrying after sleep"
          CONTAINER_RETRY_COUNT=$((CONTAINER_RETRY_COUNT+1))
          sleep $RETRY_DELAY
          continue
        else
          echo "INFO: All containers are ready"
          return 0
        fi
      done
    fi
  done

  if [ "$PODS_FOUND" -eq 0 ];then
    echo "WARN: No pods launched by this chart's default settings"
    return 0
  else
    echo "ERROR: Some containers failed to reach the ready state"
    return 1
  fi
}

########################################################################
# Main
########################################################################
#
# REQUIREMENTS:
#
# *) Must have a kubernetes cluster already running, and kubectl
# pointing to it. (E.g. kubectl config view)
#
# *) Gcloud must be configured to point to a project that already has
# needed permissions (E.g. gcloud config list)

echo "GOPATH: $GOPATH"

# Ensure Cloudflare SSL tools are installed (needed for certificate generation)
go get -u github.com/cloudflare/cfssl/cmd/...

# Install the service catalog installer binary
go get ${REPO}/installer/cmd/sc

# Install the service catalog into the kubernetes cluster
${GOPATH}/bin/sc install

# Wait for the service catalog deployments to be ready
wait_for_ready ${SC_NAMESPACE}

# TODO TEST: kubectl api-versions -> servicecatalog.k8s.io

# Connect to the GCP broker; list the services
${GOPATH}/bin/sc add-gcp-broker

########################################################################
# RUN TESTS HERE: Create instances, bindings, and check secret info
########################################################################
#
# DOESN'T EXIST YET.
#
# This test will provision instances for all services, and bind them.
# After binding, check the correct data was installed in the named secret.
# Then, unbind bindings; check the secret is removed.
# Finally, de-provision instance; check serviceinstance is removed.
#
# go run $GOPATH/src/github.com/GoogleCloudPlatform/k8s-service-catalog/installer/test/e2e_test.go
#
# TODO(seans): Replace kubectl calls with go tests.
# TODO(seans): Replace "sleep" calls with proper wait functions.
#

# TODO TEST: gcp-broker -> check status == fetched catalog
sleep 10
kubectl get clusterservicebrokers gcp-broker -n service-catalog -o yaml
# TODO TEST: serviceclasses -> check services exposed == pubsub, storage
kubectl get clusterserviceclasses -o=custom-columns=NAME:.metadata.name,EXTERNAL\NAME:.spec.externalName

sleep 10
kubectl create -f ${REPO_DIR}/installer/hack/gcp/gcp-instance-namespace.yaml
kubectl create -f ${REPO_DIR}/installer/hack/gcp/gcp-pubsub-instance.yaml
kubectl get serviceinstances gcp-pubsub-instance -n gcp-apps -o yaml
# TODO TEST: check instance status == provisioned successfully

sleep 60
kubectl create -f ${REPO_DIR}/installer/hack/gcp/gcp-pubsub-binding.yaml
kubectl get servicebindings gcp-pubsub-binding -n gcp-apps -o yaml
# TODO TEST: check binding == success
sleep 90
kubectl get secrets gcp-pubsub-credentials -n gcp-apps -o yaml
# TODO TEST: check secret for pubsub binding has four data keys:
#   project
#   serviceAccount
#   topic
#   subscription
# Base64 decode these four fields and validate returned data

# Remove instances, bindings, application namespace
#
sleep 60
kubectl delete -f ${REPO_DIR}/installer/hack/gcp/gcp-pubsub-binding.yaml

sleep 10
kubectl delete -f ${REPO_DIR}/installer/hack/gcp/gcp-pubsub-instance.yaml
kubectl delete -f ${REPO_DIR}/installer/hack/gcp/gcp-instance-namespace.yaml
# TODO TEST: Check the pubsub instance no longer exists in GCP

# Remove the connection to the GCP broker
sleep 60
${GOPATH}/bin/sc remove-gcp-broker

# Uninstall the service catalog deployments
sleep 10
${GOPATH}/bin/sc uninstall
