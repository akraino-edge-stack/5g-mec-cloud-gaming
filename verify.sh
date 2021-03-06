#!/bin/bash

echo "5g-mec-cloud-gaming verify test"

DIR=$(cd `dirname $0`;pwd)
OS=$(facter operatingsystem)
#install go
if [ -z "${GO_URL}" ]; then
  GO_URL='https://dl.google.com/go/'
fi

if [ -z "${GO_VERSION}" ]; then
 GO_VERSION='go1.13.4.linux-amd64.tar.gz'
fi

set -e -u -x -o pipefail

echo "---> Installing golang from ${GO_URL} with version ${GO_VERSION}"

wget ${GO_URL}/${GO_VERSION}
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf ${GO_VERSION}

ls /usr/local
export PATH=$PATH:/usr/local/go/bin/
export PATH=$PATH:/usr/bin/
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
go version
#export GOPROXY=https://goproxy.io

go get github.com/onsi/ginkgo/ginkgo
go get github.com/onsi/gomega/
ginkgo version

#install docker
#curl -sSL https://get.daocloud.io/docker | sh

case "$OS" in
	Ubuntu)
     		sudo apt install docker.io
   		 ;;
    	CentOS|RedHat)
       		sudo yum install -y yum-utils  device-mapper-persistent-data  lvm2

		sudo yum-config-manager    --add-repo   https://download.docker.com/linux/centos/docker-ce.repo

		sudo yum install -y  docker-ce docker-ce-cli containerd.io 
		 ;;
esac

sudo service docker start
sudo docker version
sudo mkdir -p /etc/docker
sudo tee /etc/docker/daemon.json <<-'EOF'
{
  "registry-mirrors": ["https://registry.docker-cn.com","http://hub-mirror.c.163.com"]
}
EOF
sudo service docker restart

#install docker-compose
sudo curl -L "https://github.com/docker/compose/releases/download/1.24.1/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
sudo ln -s /usr/local/bin/docker-compose /usr/bin/docker-compose
docker-compose --version

mkdir $GOPATH/src/go.ectd.io
cd  $GOPATH/src/go.ectd.io/
git clone https://github.com/etcd-io/bbolt.git

cd $DIR
git submodule update --init --recursive

make -C 5GCEmulator/ngc build
sudo --preserve-env=PATH make -C 5GCEmulator/ngc test-unit-nef
sudo systemctl daemon-reload
sudo systemctl restart docker
sudo --preserve-env=PATH make -C edgenode networkedge
make -C edgecontroller build-dnscli &&  make -C edgecontroller test-dnscli
