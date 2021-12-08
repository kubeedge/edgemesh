#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

readonly caPath=${CA_PATH:-/tmp/edgemesh/ca}
readonly caSubject=${CA_SUBJECT:-/C=CN/ST=Zhejiang/L=Hangzhou/O=KubeEdge/CN=kubeedge.io}
readonly certPath=${CERT_PATH:-/tmp/edgemesh/certs}
readonly subject=${SUBJECT:-/C=CN/ST=Zhejiang/L=Hangzhou/O=KubeEdge/CN=my-nginx.com}

EDGEMESH_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"

genCA() {
  openssl genrsa -out ${caPath}/rootCA.key 2048
  openssl req -new -out ${caPath}/rootCA.csr -key ${caPath}/rootCA.key -subj ${caSubject}
  openssl x509 -req -in ${caPath}/rootCA.csr -out ${caPath}/rootCA.crt -signkey ${caPath}/rootCA.key -CAcreateserial -days 365
}

ensureCA() {
  if [ ! -e ${caPath}/rootCA.key ] || [ ! -e ${caPath}/rootCA.crt ]; then
    genCA
  fi
}

ensureFolder() {
  if [ ! -d ${caPath} ]; then
    mkdir -p ${caPath}
  fi
  if [ ! -d ${certPath} ]; then
    mkdir -p ${certPath}
  fi
}

genCertAndKey() {
  local name=$1
  openssl genrsa -out ${certPath}/${name}.key 2048
  openssl req -new -out ${certPath}/${name}.csr -key ${caPath}/rootCA.key -subj ${subject}
  openssl x509 -req -in ${certPath}/${name}.csr -out ${certPath}/${name}.crt -signkey ${certPath}/${name}.key -CA ${caPath}/rootCA.crt -CAkey ${caPath}/rootCA.key -CAcreateserial -days 365
}

install() {
  # create cert and key
  ensureFolder
  ensureCA
  genCertAndKey nginx
  genCertAndKey client
  # create nginx secret
  kubectl create -f- <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: nginxsecret
stringData:
  rootCA.crt: |
$(pr -T -o 4 ${caPath}/rootCA.crt)
  nginx.crt: |
$(pr -T -o 4 ${certPath}/nginx.crt)
  nginx.key: |
$(pr -T -o 4 ${certPath}/nginx.key)
EOF
  # create nginx configmap
  kubectl create configmap nginxconfigmap --from-file=${EDGEMESH_ROOT}/examples/nginx-https/nginx.conf
  # create nginx service and deployment
  kubectl create -f ${EDGEMESH_ROOT}/examples/nginx-https/nginx.yaml

  modifyTestPod
  echo "create https example success!"
}

cleanup() {
  rm -rf ${caPath} ${certPath}
  kubectl delete secret nginxsecret --wait
  kubectl delete configmap nginxconfigmap --wait
  kubectl delete -f ${EDGEMESH_ROOT}/examples/nginx-https/nginx.yaml
  restoreTestPod
  echo "delete https example success!"
}

modifyTestPod() {
  # copy cert and key
  kubectl cp ${caPath}/rootCA.crt alpine-test:/rootCA.crt
  kubectl cp ${certPath}/client.crt alpine-test:/client.crt
  kubectl cp ${certPath}/client.key alpine-test:/client.key
  # backup /etc/hosts in test pod
  kubectl exec alpine-test -- cp /etc/hosts /etc/hosts.bak
  # inject domain
  nginxIP=$(kubectl get svc | grep nginx-https | awk '{print $3}')
  kubectl exec -it alpine-test -- sh -c "echo ${nginxIP}$'\t'"my-nginx.com" >> /etc/hosts"
}

restoreTestPod() {
  # delete cert and key
  kubectl exec alpine-test -- rm /rootCA.crt /client.crt /client.key
  # restore /etc/hosts
  kubectl exec -it alpine-test -- sh -c "cat /etc/hosts.bak > /etc/hosts"
  kubectl exec alpine-test -- rm /etc/hosts.bak
}

$@
