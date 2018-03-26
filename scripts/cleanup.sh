#!/bin/bash

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

