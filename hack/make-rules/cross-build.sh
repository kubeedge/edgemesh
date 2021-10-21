#!/usr/bin/env bash

# Copyright 2021 The KubeEdge Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


# update the code from https://github.com/kubeedge/sedna/blob/fa6df45c30b1ff934dba07fa0c59f3f64a943d2e/hack/make-rules/cross-build.sh
set -o errexit
set -o nounset
set -o pipefail

EDGEMESH_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
source "${EDGEMESH_ROOT}/hack/lib/init.sh"

edgemesh::buildx::build-multi-platform-images