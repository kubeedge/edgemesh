#!/bin/bash

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

# Developers can run `hack/local-up-edgemesh.sh` to setup up a local environment:
# 1. a local k8s cluster with a master node.
# 2. a kubeedge node.
# 3. our edgemesh.

# It does:
# 1. build the edgemesh image.
# 2. use kind install a k8s cluster
# 3. use keadm install kubeedge
# 4. prepare our k8s env.
# 5. config edgemesh config and start edgemesh.
# 6. add cleanup.

set -o errexit
set -o nounset
set -o pipefail

# ENABLE_DAEMON will
ENABLE_DAEMON=${ENABLE_DAEMON:-false}
EDGEMESH_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"

cd "$EDGEMESH_ROOT"

NO_CLEANUP=${NO_CLEANUP:-false}

IMAGE_TAG=localup

CLUSTER_NAME=test
MASTER_NODENAME=${CLUSTER_NAME}-control-plane
HOST_IP=`hostname -I | awk '{print $1}'`
EDGE_NODENAME=edge-node
KUBEEDGE_VERSION=1.8.2
NAMESPACE=kubeedge
LOG_DIR=${LOG_DIR:-"/tmp"}
TIMEOUT=${TIMEOUT:-120}s
KUBEAPI_PROXY_PORT=8090
KUBEAPI_PROXY_ADDR=""

if [[ "${CLUSTER_NAME}x" == "x" ]];then
    CLUSTER_NAME="test"
fi

export CLUSTER_CONTEXT="--name ${CLUSTER_NAME}"


TMP_DIR="$(realpath local-up-tmp)"

get_kubeedge_pid() {
  ps -e -o pid,comm,args |
   grep -F "$TMP_DIR" |
   # match executable name and print the pid
   awk -v bin="${1:-edgecore}" 'NF=$2==bin'
}

# spin up cluster with kind command
function kind_up_cluster {
  echo "Running kind: [kind create cluster ${CLUSTER_CONTEXT}]"
  kind create cluster ${CLUSTER_CONTEXT}
  add_cleanup '
    echo "Running kind: [kind delete cluster ${CLUSTER_CONTEXT}]"
    kind delete cluster ${CLUSTER_CONTEXT}
  '
}

function check_control_plane_ready {
  echo "wait the control-plane ready..."
  kubectl wait --for=condition=Ready node/${CLUSTER_NAME}-control-plane --timeout=${TIMEOUT}
  MASTER_IP=`docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' test-control-plane`
}

function proxy_kube_apiserver {
    set -x
    echo "proxy kube-apiserver on master"
    nohup kubectl proxy --address='0.0.0.0' --port=${KUBEAPI_PROXY_PORT} --accept-hosts='^*$' >/dev/null 2>&1 &
    PROXY_PID=$!
    add_cleanup 'sudo kill $PROXY_PID'
    KUBEAPI_PROXY_ADDR=${HOST_IP}:${KUBEAPI_PROXY_PORT}
    echo ${KUBEAPI_PROXY_ADDR}
    sleep 5
    curl ${KUBEAPI_PROXY_ADDR}
}

function check_node_ready {
  echo "wait the $1 node ready"
  kubectl wait --for=condition=Ready node/${1} --timeout=${TIMEOUT}
}

localup_kubeedge() {
  set -x
  # init cloudcore
  add_cleanup 'rm `ls /etc/kubeedge | grep -v "kubeedge"` -rf'
  add_cleanup 'sudo -E keadm reset --force --kube-config=${KUBECONFIG}'
  sudo -E keadm init --advertise-address=${HOST_IP} --kubeedge-version=${KUBEEDGE_VERSION} --kube-config=${KUBECONFIG}

  # ensure tokensecret is generated
  for ((i=1;i<20;i++)) ; do
      sleep 3
      kubectl get secret -n kubeedge| grep -q tokensecret && break
  done

  # join edgenode
  sleep 5
  add_cleanup 'sudo keadm reset --force --kube-config=${KUBECONFIG}'
  add_cleanup 'sudo rm /etc/systemd/system/edgecore.service'
  token=$(sudo keadm gettoken --kube-config=${KUBECONFIG})
  echo $token

  # turn off edgemesh and turn on local apiserver featuren and resart edgeocre
  export CHECK_EDGECORE_ENVIRONMENT="false"
  sudo -E keadm join --cloudcore-ipport=${HOST_IP}:10000 --kubeedge-version=${KUBEEDGE_VERSION} --token=${token} --edgenode-name=${EDGE_NODENAME}

  EDGE_BIN=/usr/local/bin/edgecore
  EDGE_CONFIGFILE=/etc/kubeedge/config/edgecore.yaml
  EDGECORE_LOG=${LOG_DIR}/edgecore.log
  sudo sed -i 's/clusterDNS:\ \"\"/clusterDNS:\ 169.254.96.16/g' ${EDGE_CONFIGFILE}
  sudo sed -i 's/clusterDomain:\ \"\"/clusterDomain:\ cluster.local/g' ${EDGE_CONFIGFILE}

  ps -aux | grep edgecore

  sudo pkill edgecore
  nohup sudo -E ${EDGE_BIN} --config=${EDGE_CONFIGFILE} > "${EDGECORE_LOG}" 2>&1 &
  EDGECORE_PID=$!
  sleep 15
  ps -aux | grep edgecore
  check_node_ready ${EDGE_NODENAME}
}

build_component_image() {
  local bin
  for bin; do
    echo "building $bin"
    make -C "${EDGEMESH_ROOT}" ${bin}image IMAGE_TAG=$IMAGE_TAG
    eval ${bin^^}_IMAGE="'kubeedge/edgemesh-${bin}:${IMAGE_TAG}'"
  done
  # no clean up for images
}

load_images_to_master() {
  kind load --name $CLUSTER_NAME docker-image $AGENT_IMAGE
}

prepare_k8s_env() {
  kind get kubeconfig --name $CLUSTER_NAME > $TMP_DIR/kubeconfig
  export KUBECONFIG=$(realpath $TMP_DIR/kubeconfig)
  # prepare our k8s environment

}

start_edgemesh() {
  echo "using helm to install edgemesh"
  helm install edgemesh --namespace kubeedge \
    --set agent.image=${AGENT_IMAGE} \
    --set agent.kubeAPIConfig.master=${KUBEAPI_PROXY_ADDR} \
    --set agent.modules.edgeDNS.cacheDNS.enable=true \
    --set agent.psk="edgemesh e2e test" \
    --set agent.relayNodes[0].nodeName=${MASTER_NODENAME},agent.relayNodes[0].advertiseAddress={${MASTER_IP}} \
    ./build/helm/edgemesh

  kubectl wait --timeout=${TIMEOUT} --for=condition=Ready pod -l kubeedge=edgemesh-agent -n kubeedge

  add_debug_info "See edgemesh status: kubectl get pod -n $NAMESPACE"
}

declare -a CLEANUP_CMDS=()
add_cleanup() {
  CLEANUP_CMDS+=("$@")
}

cleanup() {
  if [[ "${NO_CLEANUP}" = true ]]; then
    echo "No clean up..."
    return
  fi

  set +o errexit

  echo "Cleaning up edgemesh..."

  sudo rm -rf /etc/kubeedge /var/lib/kubeedge

  local idx=${#CLEANUP_CMDS[@]} cmd
  # reverse call cleanup
  for((;--idx>=0;)); do
    cmd=${CLEANUP_CMDS[idx]}
    echo "calling $cmd:"
    eval "$cmd"
  done

  set -o errexit
}

check_healthy() {
  true
}

debug_infos=""
add_debug_info() {
  debug_infos+="$@
"
}

check_prerequisites() {
  true
}

NO_COLOR='\033[0m'
RED='\033[0;31m'
GREEN='\033[0;32m'
green_text() {
  echo -ne "$GREEN$@$NO_COLOR"
}

red_text() {
  echo -ne "$RED$@$NO_COLOR"
}

label_node() {
  kubectl label nodes ${EDGE_NODENAME} lan=edge-lan-01
}

create_istio_crd() {
  echo "creating the istio crds..."
  kubectl apply -f ${EDGEMESH_ROOT}/build/crds/istio/
}

do_up() {
  cleanup

  mkdir -p "$TMP_DIR"
  add_cleanup 'rm -rf "$TMP_DIR"'

  kind_up_cluster

  prepare_k8s_env

  check_control_plane_ready

  kubectl create ns kubeedge

  proxy_kube_apiserver

  # here local up kubeedge before building our images, this could avoid our
  # images be removed since edgecore image gc would be triggered when high
  # image usage(>=80%), see https://github.com/kubeedge/sedna/issues/26 for
  # more details
  localup_kubeedge

  check_prerequisites

  create_istio_crd

  build_component_image agent
  load_images_to_master

  start_edgemesh

  label_node
}

do_up_fg() {
  do_up

  echo "Local cluster is $(green_text running).
  Currently local-up script only support foreground running.
  Press $(red_text Ctrl-C) to shut it down!
  You can use it with: kind export kubeconfig --name ${CLUSTER_NAME}
  $debug_infos
  "
  while check_healthy; do sleep 5; done
}

main() {
  if [ "${ENABLE_DAEMON}" = false ]; then
    trap cleanup EXIT
    trap clean ERR
    do_up_fg
  else  # DAEMON mode, for run e2e
    trap clean ERR
    trap clean INT
    do_up
  fi
}

main
