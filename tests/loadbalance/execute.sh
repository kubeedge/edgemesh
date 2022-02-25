#!/bin/bash

set -e

workdir=`pwd`
cd $workdir

curpath=$PWD

LB_DIR=${curpath}/tests/loadbalance
echo ${LB_DIR}
cd ${LB_DIR}

clean() {
  rm -rf loadbalance
}

clean

go build -o loadbalance .

echo "Round Robin Test"
./loadbalance -counter 10000 -deployment hostname-deploy.yaml -service hostname-svc.yaml

sleep 3

echo "Random Test"
./loadbalance -counter 10000 -deployment hostname-deploy.yaml -service hostname-svc.yaml -destination-rule hostname-dr-random.yaml

clean
