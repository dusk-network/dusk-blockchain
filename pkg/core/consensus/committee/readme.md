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
    - `Quorum()` int: returns the number of Committee members needed to form a _quorum_. This quantity depends on the amount of available `Provisioners` but it is constant after a certain threshold

Additionally, the `committee` package exposes the `NewExtractor` function. It returns the basis for an implementation of the `Committee` interface, made up of a `Store`, and an `Extractor`.

### Architecture

![](docs/Committee.jpg)

The `Store` is responsible for maintaining an up-to-date set of provisioners. The `Extractor` wraps around the `Store`, exposing functionality needed to create voting committees from this `Store` for any point in the consensus. The combination of the two provides a basis for other packages to create their own specific implementations of the `Committee` interface, so that the `committee` package may be re-used across the codebase.
