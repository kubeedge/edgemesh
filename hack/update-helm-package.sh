#!/usr/bin/env bash

###
#Copyright 2021 The KubeEdge Authors.
#
#Licensed under the Apache License, Version 2.0 (the "License");
#you may not use this file except in compliance with the License.
#You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
#Unless required by applicable law or agreed to in writing, software
#distributed under the License is distributed on an "AS IS" BASIS,
#WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#See the License for the specific language governing permissions and
#limitations under the License.
###

set -o errexit
set -o nounset
set -o pipefail

KUBE_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
EDGEMESH_HELM_DIR=${KUBE_ROOT}/build/helm/edgemesh
GATEWAY_HELM_DIR=${KUBE_ROOT}/build/helm/edgemesh-gateway
_tmpdir=/tmp/edgemesh

function package:helm:files {
    mkdir -p ${_tmpdir}
    cd $EDGEMESH_HELM_DIR && helm package . -d ${_tmpdir} > /dev/null && mv ${_tmpdir}/*.tgz $EDGEMESH_HELM_DIR/../edgemesh.tgz
    cd $GATEWAY_HELM_DIR && helm package . -d ${_tmpdir}  > /dev/null && mv ${_tmpdir}/*.tgz $GATEWAY_HELM_DIR/../edgemesh-gateway.tgz
    echo "helm files updated...";
}

function cleanup {
  #echo "Removing workspace: ${_tmpdir}"
  rm -rf "${_tmpdir}"
}


package:helm:files
cleanup