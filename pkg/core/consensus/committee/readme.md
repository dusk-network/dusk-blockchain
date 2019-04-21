## Committee

The `Committee` represents the subset of Provisioners extracted at each Step in order to perform _block hash_ voting. The `Committee` gets chosen according to a _deterministic sortition_.

### Deterministic Sortition

Deterministic sortition is an optimization of cryptographic sortition introduced by Micali et al. It extends the functionality of cryptographic sortition in a Random Oracle setting in a non-interactive fashion, improving both the network throughput and space-efficiency.

Deterministic sortition is an algorithm that recursively hashes the public seed with situational parameters of each step, mapping the outcome to the current stakes of the Provisioners in order to extract a pseudo-random `Committee`, per step.

### Values

#### NewProvisioner Event

| Field | Type |
| opcode | uint8 |
| PubKeyED | []byte |
| PubKeyBLS | []byte |
| stake | uint64 |

### API

`Committee` is an interface that exposes the following functionalities:

    - `IsMember([]byte, uint64, uint8)` bool: returns whether the ID of a provisioner (basically the `BLS Public Key`) is included in the committee
    - `Verify(eventHeader)` error: Verify if the `EventHeader` has been propagated by a `Committee` member and performs general validation of the event itself (i.e. checking for duplicates, verifying signatures, etc)
    - `Quorum()` int: returns the number of Committee members needed to form a _quorum_. This quantity depends on the amount of available `Provisioners` but it is constant after a certain threshold (normally 50)

The `committee` package also exposes the following function: - `NewCommitteeStore(eventbus)` - creates a new `CommitteeStore` subscribed to the `NewProvisionerTopic` topic

### Architecture

![](docs/Committee.jpg)

The package includes a `CommitteeStore`, a `Committee` interface implementation that is wired to the `EventBus` and follows the _event-driven_ approach of the general Provisioner's architecture. The `CommitteeStore` listens to the `NewProvisionerTopic` topic and keeps track of the known `Provisioners` and their stake.
