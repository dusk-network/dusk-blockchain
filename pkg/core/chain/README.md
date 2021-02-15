# Chain component

The `Chain` is a component which is fully responsible for building the blockchain. It judges the validity of incoming blocks, appends them to the database, and disseminates newly approved blocks both internally (through the event bus) and externally (over the gossip network). Additionally, it is aware of the node's state in the network at all times, and controls the execution of the consensus by responding to perceived state changes.

## Structure

To the rest of the system, the `Chain` appears as quite a monolithic component. However, it is composed by a few different, more granular components, which all fulfill their own responsibilities.

### Loader

The `Loader` is responsible for initializing the `Chain` with the correct initial data, as well as checking for data integrity on startup. It abstracts over the DB, and is implemented by [the loader](./loader.go).

### Verifier

The `Verifier` abstracts over block verification mechanisms, and uses the [`verifiers` package](../verifiers/README.md) under the hood, in an attempt to simplify the API (as checks are supposed to be kept atomic from a function perspective, but always are called together). It is implemented by [the loader](./loader.go).

### Synchronizer

The `Synchronizer` is a stand-alone component which is fully responsible for processing blocks that come in from the network. Through the `Ledger` interface, it can direct the `Chain` into the right decisions based on the perceived state of the node against the network. More information can be found in the [synchronizer documentation](./synchronizer.md).

Below follows a visual workflow of how the chain and synchronizer take decisions on incoming blocks, which should help clarify the control flow.

![Block processing decision tree](./chain_processing_flow.jpg)

### Loop

The `Loop` allows the `Chain` to take control of consensus execution, by allowing it to easily start and stop the work being done.

### Proxy

The `Proxy` is used to contact the RUSK server. The `Chain` outsources certain operations to RUSK since the current codebase can not execute VM transactions, and this is needed for transaction validity checks and state transitions.
