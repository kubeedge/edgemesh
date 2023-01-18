#!/usr/bin/env bash

###
#Copyright 2022 The KubeEdge Authors.
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

EDGEMESH_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
source "${EDGEMESH_ROOT}/hack/update-helm-package.sh"

CURRENT_TAG=$1
HANDLE_FILES=(
  "${EDGEMESH_ROOT}/build/agent/resources/05-daemonset.yaml"
  "${EDGEMESH_ROOT}/build/gateway/resources/05-deployment.yaml"
  "${EDGEMESH_ROOT}/build/helm/edgemesh/values.yaml"
  "${EDGEMESH_ROOT}/build/helm/edgemesh-gateway/values.yaml"
)

function update:image:tag {
  for((i=0;i<=${#HANDLE_FILES[@]}-1;i++))
  do
       updateTag ${HANDLE_FILES[i]}
  done
}

updateTag() {
  filename=$1
  echo $filename
  sed -i "s/\bimage: \(.\+\/.\+\):\(.\+\)/image: \1:${CURRENT_TAG}/g" $filename
}

update:image:tag
package:helm:files