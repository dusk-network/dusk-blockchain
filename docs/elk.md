1. Install Filebeat
```
wget -qO - https://artifacts.elastic.co/GPG-KEY-elasticsearch | sudo apt-key add -
echo "deb https://artifacts.elastic.co/packages/7.x/apt stable main" | sudo tee -a /etc/apt/sources.list.d/elastic-7.x.list
sudo apt-get update && sudo apt-get install filebeat
```

2. Install Docker and Docker compose

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
make build && watch "./bin/utils transactions --txtype=transfer --amount=1 --locktime=1 --grpchost=unix://$GOPATH/src/github.com/dusk-network/dusk-blockchain/devnet/dusk_data/dusk0/dusk-grpc.sock"
```
## Cleanup
To Remove the docker with:
```
docker-compose rm
```