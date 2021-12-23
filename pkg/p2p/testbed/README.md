## Description
Testbed package should provide a platform for conducting transparent and replicable stress testing of Kadcast broadcast protocol.

## How to run
The testbed should be run manually until rusk service is fully integrated.

```bash
# Remove t.SkipNow()
RUSK_PATH="/dusk-network/rusk/target/release/rusk" CLUSTER_SIZE=100 MSG_SIZE=1000  go test -tags=testbed -run TestCluster
```

* CLUSTER_SIZE - number of nodes a cluster consists of 
* MSG_SIZE - size of the message payload to be broadcasted
* RUSK_PATH - rusk executable path that provides Kadcast service
* -tags=testbed - testbed package is not included in main executable.