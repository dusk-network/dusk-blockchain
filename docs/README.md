# Documentation for Dusk Blockchain Node

This will be the central point from which users can orient in order to
discover the relevant documentation for their questions and purposes.

<!-- ToC start -->

## Contents

1. [Contents](#contents)
2. [Software Licences](#software-licences)
3. [Internal use documents ](#internal-use-documents-)
4. [Elastic Stack on Docker](#elastic-stack-on-docker)

<!-- ToC end -->

## Software Licences

[LICENCE](./LICENCE) contains the licences referring to code which has been
redistributed in whole or part within this repository.

## Internal use documents

[internal/zenhub.md](./internal/zenhub.md) regarding the special
considerations for facilitating the use of Zenhub for project management.

## Elastic Stack on Docker

Notes about using elastic stack and kibana for log processing.

1. Install Filebeat

```
wget -qO - https://artifacts.elastic.co/GPG-KEY-elasticsearch | sudo apt-key add -
echo "deb https://artifacts.elastic.co/packages/7.x/apt stable main" | sudo tee -a /etc/apt/sources.list.d/elastic-7.x.list
sudo apt-get update && sudo apt-get install filebeat
```

2. Install Docker and Docker compose

```
sudo apt-get update

sudo apt-get install \
    apt-transport-https \
    ca-certificates \
    curl \
    gnupg-agent \
    software-properties-common

sudo apt install docker.io

sudo curl -L "https://github.com/docker/compose/releases/download/1.26.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose


```

3. Clone and run compose

```
git clone https://github.com/dusk-network/docker-elk.git

docker-compose up
```

4. Start devnet

```
./devnet -f true
```

5. Start metrics

```
make build && ./bin/utils metrics
```

6. Send some txs to node 0

```
make build && watch "./bin/utils transactions --txtype=transfer --amount=1 --locktime=1 --grpcaddr=unix://$GOPATH/src/github.com/dusk-network/dusk-blockchain/devnet/dusk_data/dusk0/dusk-grpc.sock"
```

7. Cleanup

Remove the docker with:

```
docker-compose rm
```

<!-- 
# to regenerate this file's table of contents:
markdown-toc README.md --replace --skip-headers 2 --inline --header "##  Contents"
-->
