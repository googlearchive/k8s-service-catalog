#!/bin/bash
##################################################################
#
# Script to generate public/private key pairs for a certificate
# authority and the service catalog APIServer. These certificates
# will be used for secure communication between the main,
# aggregated APIServer and the extension service catalog
# APIServer.
#
# Parameters:
#    TUT_DIR=temporary directory for generation
#
# NOTE: Currently this script assumes the following binaries
# are in the path:
#   cfssl
#   cfssljson
# These binaries should have been downloaded into ${GOPATH}/bin
# with:
#   go get -u github.com/cloudflare/cfssl/cmd/...
#
##################################################################

# Parameter: temporary directory for key generation.
if [ "$#" -ne 1 ]; then
    echo "Missing parameter: temporary directory"
    exit 1
fi
export TUT_DIR=$1

# These are used later to locate key files.
export TUT_BARE_CA=ca
export TUT_BARE_API=apiserver

function TUT_confirmFile {
  if [ ! -f "$1" ]; then
    echo Problem creating $1
    exit 1
  fi
}

function TUT_makeKeys {
  local svcCatServiceName=service-catalog-api
  local svcCatNamespace=service-catalog

  # TODO: Document these host choices.
  local h1=${svcCatServiceName}.${svcCatNamespace}
  local h2=${svcCatServiceName}.${svcCatNamespace}.svc
  local hosts=\"$h1\",\"$h2\"

  # Generate a self-signed root certificate request bundle,
  # and have cfssljson split the bundle into the files
  # ca.csr, ca.pem, and ca-key.pem.

  cat <<EOF | \
      cfssl genkey --initca - | \
      cfssljson -bare $TUT_BARE_CA
  {
    "hosts": [ ${hosts} ],
    "key": {
        "algo": "rsa",
        "size": 2048
    },
    "names": [
        {
            "C": "US",
            "L": "san jose",
            "O": "kube",
            "OU": "WWW",
            "ST": "California"
        }
    ]
  }
EOF

  # Configure the cert generation process.
  local configFile=$(mktemp --tmpdir=$TUT_DIR)
  cat > $configFile <<EOF
  {
    "signing": { 
      "default": {
        "expiry": "43800h",
        "usages": [ "signing", "key encipherment", "server" ]
      }
    }
  }
EOF

  # Now using this self-signed certificate authority info,
  # generates a locally issued cert and key:
  cat <<EOF | \
      cfssl gencert \
          -ca=${TUT_BARE_CA}.pem \
          -ca-key=${TUT_BARE_CA}-key.pem \
          -config=$configFile - | \
      cfssljson -bare ${TUT_BARE_API}
  {
    "CN": "${svcCatServiceName}",
    "hosts": [ ${hosts} ],
    "key": {
      "algo": "rsa",
      "size": 2048
    }
  }
EOF

  echo "PKI/TLS files for secure communication between APIServer and service catalog:"
  echo "host1=$h1"
  echo "host2=$h2"
  TUT_confirmFile ${TUT_BARE_CA}.pem
  TUT_confirmFile ${TUT_BARE_API}.pem
  TUT_confirmFile ${TUT_BARE_API}-key.pem
}

cd $TUT_DIR
TUT_makeKeys

