# beans-cli

Simple command-line interface for interaction with beanstalk server

## Installation
```bash
export BEANSCLI_SRC_PATH=/tmp/beans-cli-src
export BEANSCLI_INSTALL_PATH=/usr/local/bin
export GOPATH=$BEANSCLI_SRC_PATH/lib
export GOBIN=$GOPATH/bin

git clone https://github.com/webproducer/beans-cli.git $BEANSCLI_SRC_PATH
cd $BEANSCLI_SRC_PATH
mkdir -p lib/bin
go get
go build -o $BEANSCLI_INSTALL_PATH/beans-cli beans-cli.go
```

## Usage

Type `beans-cli help` to get instructions
