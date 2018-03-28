#!/bin/bash

# Copyright 2018 Google LLC
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

# Deletes all existing service instances and bindings
# across all namespaces.

delete_resource () {
  printf "Deleting $1 in $2\n"
  kubectl delete $1 -n $2 > /dev/null
  while kubectl get $1 -n $2 -o name > /dev/null 2>&1; do
    printf "."
  done
  printf "\nDone deleting $1 in $2\n"
}

echo Cleanup!
for namespace in $(kubectl get namespace -o name)
do

  for resource in servicebinding serviceinstance; do
    for r in $(kubectl get $resource -n $namespace -o name); do
      printf "$r in $namespace\n"
    done
  done

done

REPLY='n'
read -n 10000 -t 1
read -p "You are about to delete the above resources, are you sure? [y/n] "
if [[ ! $REPLY =~ ^[Yy]$ ]]
then
  printf "Aborting cleanup ...\n"
  exit 0;
fi

echo
for namespace in $(kubectl get namespace -o name)
do

  for resource in servicebinding serviceinstance; do
    for r in $(kubectl get $resource -n $namespace -o name); do
      delete_resource $r $namespace
    done
  done

done

