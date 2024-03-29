#!/usr/bin/env bash

sudo apt-get update
sudo apt-get -y upgrade

sudo apt install make -y


if [ -d "/usr/local/go" ] 
then
    echo "Go already installed" 
    echo "skip installing go"
else
    wget https://dl.google.com/go/go1.16.6.linux-amd64.tar.gz
    tar -xvf go1.16.6.linux-amd64.tar.gz
    sudo mv go /usr/local
    rm -f go1.16.6.linux-amd64.tar.gz
fi

echo 'export GOROOT=/usr/local/go' >> ~/.profile
echo 'export GOPATH=$HOME/go' >> ~/.profile
echo 'export PATH=/usr/local/go/bin:$HOME/go/bin:/usr/local/bin:$PATH' >> ~/.profile
source ~/.profile
