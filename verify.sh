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
docker version

sudo service docker start
echo "docker start"
#mkdir -p /etc/docker
#tee /etc/docker/daemon.json <<-'EOF'
#{
#  "registry-mirrors": ["https://registry.docker-cn.com","http://hub-mirror.c.163.com"]
#}
#EOF
#sudo service docker restart

#install docker-compose
case "$OS" in
   	 Ubuntu)
   		 ;;
    	CentOS|RedHat)
       		sudo yum -y install epel-release
		sudo yum install python-pip
   		 ;;
esac
cd /usr/local/bin/
sudo curl -L https://get.daocloud.io/docker/compose/releases/download/1.25.0/docker-compose-`uname -s`-`uname -m` >  /usr/local/bin/docker-compose

sudo rename docker-compose-Linux-x86_64 docker-compose docker-compose-Linux-x86_64
sudo chmod +x /usr/local/bin/docker-compose
docker-compose version

mkdir $GOPATH/src/go.ectd.io
cd  $GOPATH/src/go.ectd.io/
git clone https://github.com/etcd-io/bbolt.git

git submodule update --init --recursive

cd $DIR
make -C 5GCEmulator/ngc build
sudo --preserve-env=PATH make -C 5GCEmulator/ngc test-unit-nef
make -C edgenode networkedge
make -C edgecontroller build-dnscli &&  make -C edgecontroller test-dnscli
