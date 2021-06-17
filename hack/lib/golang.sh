#!/usr/bin/env bash

# Copyright 2014 The Kubernetes Authors.
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

# -----------------------------------------------------------------------------
# CHANGELOG
# KubeEdge Authors:
# To Get Detail Version Info for KubeEdge Project

# copy from https://github.com/kubeedge/kubeedge/blob/081d4f245725d44f23d9a2919db99a01c56a83e9/hack/lib/golang.sh

set -o errexit
set -o nounset
set -o pipefail

YES="y"
NO="n"

edgemesh::version::get_version_info() {

  GIT_COMMIT=$(git rev-parse "HEAD^{commit}" 2>/dev/null)

  if git_status=$(git status --porcelain 2>/dev/null) && [[ -z ${git_status} ]]; then
    GIT_TREE_STATE="clean"
  else
    GIT_TREE_STATE="dirty"
  fi

  GIT_VERSION=$(git describe --tags --abbrev=14 "${GIT_COMMIT}^{commit}" 2>/dev/null)

  # This translates the "git describe" to an actual semver.org
  # compatible semantic version that looks something like this:
  #   v1.1.0-alpha.0.6+84c76d1142ea4d
  #
  # TODO: We continue calling this "git version" because so many
  # downstream consumers are expecting it there.
  #
  # These regexes are painful enough in sed...
  # We don't want to do them in pure shell, so disable SC2001
  # shellcheck disable=SC2001
  DASHES_IN_VERSION=$(echo "${GIT_VERSION}" | sed "s/[^-]//g")
  if [[ "${DASHES_IN_VERSION}" == "---" ]] ; then
    # shellcheck disable=SC2001
    # We have distance to subversion (v1.1.0-subversion-1-gCommitHash)
    GIT_VERSION=$(echo "${GIT_VERSION}" | sed "s/-\([0-9]\{1,\}\)-g\([0-9a-f]\{14\}\)$/.\1\+\2/")
  elif [[ "${DASHES_IN_VERSION}" == "--" ]] ; then
      # shellcheck disable=SC2001
      # We have distance to base tag (v1.1.0-1-gCommitHash)
      GIT_VERSION=$(echo "${GIT_VERSION}" | sed "s/-g\([0-9a-f]\{14\}\)$/+\1/")
  fi

  if [[ "${GIT_TREE_STATE}" == "dirty" ]]; then
    # git describe --dirty only considers changes to existing files, but
    # that is problematic since new untracked .go files affect the build,
    # so use our idea of "dirty" from git status instead.
    GIT_VERSION+="-dirty"
  fi


  # Try to match the "git describe" output to a regex to try to extract
  # the "major" and "minor" versions and whether this is the exact tagged
  # version or whether the tree is between two tagged versions.
  if [[ "${GIT_VERSION}" =~ ^v([0-9]+)\.([0-9]+)(\.[0-9]+)?([-].*)?([+].*)?$ ]]; then
    GIT_MAJOR=${BASH_REMATCH[1]}
    GIT_MINOR=${BASH_REMATCH[2]}
    if [[ -n "${BASH_REMATCH[4]}" ]]; then
      GIT_MINOR+="+"
    fi
  fi

  : <<EOF
  # If GIT_VERSION is not a valid Semantic Version, then refuse to build.
  if ! [[ "${GIT_VERSION}" =~ ^v([0-9]+)\.([0-9]+)(\.[0-9]+)?(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$ ]]; then
      echo "GIT_VERSION should be a valid Semantic Version. Current value: ${GIT_VERSION}"
      echo "Please see more details here: https://semver.org"
      exit 1
  fi
EOF
}

# Get the value that needs to be passed to the -ldflags parameter of go build
edgemesh::version::ldflags() {
  edgemesh::version::get_version_info

  local -a ldflags
  function add_ldflag() {
    local key=${1}
    local val=${2}
    # If you update these, also update the list pkg/version/def.bzl.
    ldflags+=(
      "-X ${KUBEEDGE_GO_PACKAGE}/pkg/version.${key}=${val}"
    )
  }

  add_ldflag "buildDate" "$(date ${SOURCE_DATE_EPOCH:+"--date=@${SOURCE_DATE_EPOCH}"} -u +'%Y-%m-%dT%H:%M:%SZ')"
  if [[ -n ${GIT_COMMIT-} ]]; then
    add_ldflag "gitCommit" "${GIT_COMMIT}"
    add_ldflag "gitTreeState" "${GIT_TREE_STATE}"
  fi

  if [[ -n ${GIT_VERSION-} ]]; then
    add_ldflag "gitVersion" "${GIT_VERSION}"
  fi

  if [[ -n ${GIT_MAJOR-} && -n ${GIT_MINOR-} ]]; then
    add_ldflag "gitMajor" "${GIT_MAJOR}"
    add_ldflag "gitMinor" "${GIT_MINOR}"
  fi

  # The -ldflags parameter takes a single string, so join the output.
  echo "${ldflags[*]-}"
}


# edgemesh::binaries_from_targets take a list of build targets and return the
# full go package to be built
edgemesh::golang::binaries_from_targets() {
  local target
  for target in "$@"; do
    echo "${EDGEMESH_GO_PACKAGE}/${target}"
  done
}

edgemesh::check::env() {
  errors=()
  if [ -z $GOPATH ]; then
    errors+="GOPATH environment value not set"
  fi

  # check other env

  # check length of errors
  if [[ ${#errors[@]} -ne 0 ]] ; then
    local error
    for error in "${errors[@]}"; do
      echo "Error: "$error
    done
    exit 1
  fi
}

ALL_BINARIES_AND_TARGETS=(
  edgemesh:cmd/edgemesh
)

edgemesh::golang::get_target_by_binary() {
  local key=$1
  for bt in "${ALL_BINARIES_AND_TARGETS[@]}" ; do
    local binary="${bt%%:*}"
    if [ "${binary}" == "${key}" ]; then
      echo "${bt##*:}"
      return
    fi
  done
  echo "can not find binary: $key"
  exit 1
}

edgemesh::golang::get_all_targets() {
  local -a targets
  for bt in "${ALL_BINARIES_AND_TARGETS[@]}" ; do
    targets+=("${bt##*:}")
  done
  echo ${targets[@]}
}

edgemesh::golang::get_all_binaries() {
  local -a binaries
  for bt in "${ALL_BINARIES_AND_TARGETS[@]}" ; do
    binaries+=("${bt%%:*}")
  done
  echo ${binaries[@]}
}

IFS=" " read -ra EDGEMESH_ALL_TARGETS <<< "$(edgemesh::golang::get_all_targets)"
IFS=" " read -ra EDGEMESH_ALL_BINARIES<<< "$(edgemesh::golang::get_all_binaries)"

edgemesh::golang::build_binaries() {
  edgemesh::check::env
  local -a targets=()
  local binArg
  for binArg in "$@"; do
    targets+=("$(edgemesh::golang::get_target_by_binary $binArg)")
  done


  if [[ ${#targets[@]} -eq 0 ]]; then
    targets=("${EDGEMESH_ALL_TARGETS[@]}")
  fi


  local -a binaries
  while IFS="" read -r binary; do binaries+=("$binary"); done < <(edgemesh::golang::binaries_from_targets "${targets[@]}")

  local goldflags gogcflags

  # If GOLDFLAGS is unset, then set it to the a default of "-s -w".


  goldflags="${GOLDFLAGS=-s -w -buildid=} $(edgemesh::version::ldflags)"
  gogcflags="${GOGCFLAGS:-}"


  mkdir -p ${EDGEMESH_OUTPUT_BINPATH}

  for bin in ${binaries[@]}; do
    echo "building $bin"
    local name="${bin##*/}"
    set -x
    go build -o ${EDGEMESH_OUTPUT_BINPATH}/${name} -gcflags="${gogcflags:-}" -ldflags "${goldflags:-}" $bin
    set +x
  done

}
