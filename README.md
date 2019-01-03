# dusk-go


WIP

P2P TODO list:

Setup something like DNS peers (or permanent nodes)
Extend p2p syncing
Impl a combination of header and block sync algorithm
Keep local block height in memory
Version validation in handshake and older Version support
Batch database inserts LevelDB
Impl processing of all other messages

NON-FUNCTIONALS:
Externalize configuration
Configurable and dynamic logging
Documentation p2p processes (diagrams with Gliffy)
Decide on fields of Genesis block (like what timestamp)