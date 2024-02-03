#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

SPIDERPOOL_REPO_NAME="spiderpool"
SPIDERPOOL_REPO_URL="https://spidernet-io.github.io/spiderpool"
SPIDERPOOL_VERSION="0.9.1"

if ! which helm &>/dev/null ; then
    echo "error, please install 'helm'"
    exit 1
fi

CURRENT_FILENAME=$( basename $0 )
CURRENT_DIR_PATH=$(cd $(dirname $0); pwd)
PROJECT_ROOT_PATH=$( cd ${CURRENT_DIR_PATH}/.. && pwd )
SPIDERPOOL_HELM_DIR=${PROJECT_ROOT_PATH}/build/helm/edgemesh/charts
SPIDERPOOL_HELM_PATH=${SPIDERPOOL_HELM_DIR}/spiderpool

echo "generate helm chart for spiderpool ${SPIDERPOOL_VERSION}"
rm -rf ${SPIDERPOOL_HELM_PATH}
helm repo add ${SPIDERPOOL_REPO_NAME} ${SPIDERPOOL_REPO_URL}
helm repo update ${SPIDERPOOL_REPO_NAME}
cd ${SPIDERPOOL_HELM_DIR}
helm pull ${SPIDERPOOL_REPO_NAME}/${SPIDERPOOL_REPO_NAME} --untar --version ${SPIDERPOOL_VERSION}
echo "succeeded to generate chart to $SPIDERPOOL_HELM_PATH"
exit 0
