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

[[ ( -n "${TRAVIS_BUILD_DIR}" ) && ( -n "${TRAVIS_OS_NAME}" ) ]] \
  || { echo "Run under Travis only"; exit 1; }


{
  echo "Building Service Catalog Installer"
  cd "${TRAVIS_BUILD_DIR}/installer" \
    && make
} \
  || { echo "Build failed."; exit 1; }

{
  echo "Building Broker CLI"
  cd "${TRAVIS_BUILD_DIR}/broker-cli" \
    && make
} \
  || { echo "Build failed."; exit 1; }

if [[ -n "${TRAVIS_TAG}" ]]; then
  # Package the binary to a release file

  case "${TRAVIS_OS_NAME}" in
    linux)
      flavor="linux-amd64"
    ;;
    osx)
      flavor="darwin-amd64"
    ;;
    *) echo Unknown TRAVIS_OS_NAME=${TRAVIS_OS_NAME}
       exit 1
       ;;
  esac

  BIN="${TRAVIS_BUILD_DIR}/installer/output/bin"
  for pkg in cfssl cfssljson; do
    url="https://pkg.cfssl.org/R1.2/${pkg}_${flavor}"
    curl -o "${BIN}/${pkg}" "${url}" \
      || { echo "Cannot download ${url}"; exit 1; }
    curl -o "${BIN}/${pkg}.LICENSE" "https://raw.githubusercontent.com/cloudflare/cfssl/master/LICENSE" \
      || { echo "Cannot download cfssl LICENSE"; exit 1; }
    chmod +x "${BIN}/${pkg}"
  done

  tar --create --gzip \
    --file="${TRAVIS_BUILD_DIR}/installer/output/service-catalog-installer-${TRAVIS_TAG}-${TRAVIS_OS_NAME}.tgz" \
    --directory="${BIN}" \
    --verbose \
    sc cfssl cfssljson \
    cfssl.LICENSE cfssljson.LICENSE
fi
