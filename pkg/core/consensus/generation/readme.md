## Block Generator

`Block Generator` is the first of the two full-node types eligible to participate in the consensus (the other being the `Provisioner`). To become a `Block Generator`, a full-node has to submit a `Bid Transaction`. The `Block Generator` is eligible to participate in one phase - the _block generation_ where the `Block Generators` participates in a non-interactive lottery to be able to forge a candidate block.

### Blind Bid Algorithm

The block generation makes use of `bulletproof`-based _zero-knowledge cryptography_ in order to prove the correct computation of a _score_ associated with a `block candidate` which gets validated and selected by Provisioners during the _score selection_ phase carried out by the `ScoreSelector` component.

#### Procedure

The `Blind Bid` algorithm is outlined in the following steps

1. A universal counter `N` is maintained for all bidding transactions in the lifetime.
2. Seed `S` is computed and broadcasted.
3. Bidder selects secret `K`.
4. Bidder sends a bidding transaction with data `M = H(K)`.
5. For every bidding transaction with d coins and data M an entry `X = H(d,M,N)` is added to `T`. Then `N` is increased.
6. Potential bidder computes `Y =H(S,X)`, score `Q=F(d,Y)`, and identifier `Z =H(S,M)`.
7. Bidder selects a bid root `RT` and broadcasts proof `π = Π(Z, RT, Q, S; K, d, N)`.

### Architecture

The zk-proof generation is delegated to a dedicated process written in _rustlang_ which communicates with the main _golang_ process through _named pipes_ (also known as `FIFO`). Named pipes have been chosen over other `IPC` methodologies due to the excellent performance they offer. According to our benchmarks, _named pipes_ is twice more performant than _CGO_ (which allows _golang_ to perform calls to _C_ libraries) and comparable to calling the _rust_ process directly.
