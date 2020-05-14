#!/bin/bash

echo "5g-mec-cloud-gaming verify test"

if [ -z "${GO_URL}" ]; then
    GO_URL='https://dl.google.com/go/'
fi

if [ -z "${GO_VERSION}" ]; then
    GO_VERSION='go1.13.4.linux-amd64.tar.gz'
fi

set -e -u -x -o pipefail

echo "---> Installing golang from ${GO_URL} with version ${GO_VERSION}"

# install go
wget ${GO_URL}/${GO_VERSION}
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf ${GO_VERSION}

ls /usr/local

export GOPATH=$HOME/go
export PATH=$PATH:/usr/bin/
export PATH=$PATH:$GOPATH/bin
export PATH=$PATH:/usr/local/go/bin

go get github.com/onsi/ginkgo/ginkgo
go get github.com/onsi/gomega/

git submodule update --init --recursive
#make -C 5GCEmulator/ngc build
#echo `ginkgo version`
#ginkgo version
make -C 5GCEmulator/ngc nef
ginkgo -v -cover ./pkg/nef
#sudo make -C 5GCEmulator/ngc test-unit-nef
