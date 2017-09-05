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
# Helper script to bring down service catalog cleanly.
#
##################################################################

SVC_ROOT=$(dirname "${BASH_SOURCE}")
SVC_DIR=${SVC_ROOT}

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
