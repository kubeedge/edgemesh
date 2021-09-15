#!/usr/bin/env bash

###
#Copyright 2019 The KubeEdge Authors.
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

# copy from https://github.com/kubeedge/kubeedge/blob/081d4f245725d44f23d9a2919db99a01c56a83e9/hack/verify-vendor.sh

set -o errexit
set -o nounset
set -o pipefail

# The root of the build/dist directory
EDGEMESH_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"

function edgemesh::git::check_status() {
	# check if there's any uncommitted changes on go.mod, go.sum or vendor/
	echo $( git status --short 2>/dev/null | grep -E "go.mod|go.sum|vendor/" |wc -l)
}

${EDGEMESH_ROOT}/hack/update-vendor.sh
 
ret=$(edgemesh::git::check_status)
if [ ${ret} -eq 0 ]; then
	echo "SUCCESS: Vendor Verified."
else
	git status
	echo  "FAILED: Vendor Verify failed. Please run the command to check your directories: git status"
	exit 1
fi
