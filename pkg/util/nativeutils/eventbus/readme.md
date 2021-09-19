# eventbus

The `eventbus` is a message broadcast system for passing requests and returning
responses in a content-agnostic manner, using a subscription channel code to
associate publishers and subscribers.

The data is generally already binary encoded for wire or disk and thus lends
itself readily to enabling the use of pipes or sockets to connect processes
running eventbus connections, enabling the reduction of independent units of the
server to a single function.

This package is the central starting point for decomposing dusk-blockchain
server into a collection of worker units that are run and configured by parent
processes and using a star network topology, a central broker receives and
relays messages across the pipe/socket bridges to other processes, who also have
a broker relaying subscription requests to the upstream process which then
subscribes and proxies messages to the subscribers.

With this framework encapsulating individual services and allowing them to make
calls to each other after breaking down the subsystems in the server into units
connected via eventbus launched independently, largely a compositional process,
then you can approach porting of the individual units, much smaller and more
defined, much more easily into Rust, or for that matter, in the future, any
compliant implementation of `eventbus` socket/pipes for any language whatsoever.

## Interfaces

### Listener

- `Notify(message.Message) error`
- `Close()`

### Broker

#### Subscriber

- `Subscribe(topic topics.Topic, listener Listener) uint32`
  
  Adds the listener to the list of brokers to be sent messages with a given
  topic

  When messages arrive, the `Notify` method of the
  `Listener` is called with the message.

- `Unsubscribe(topic topics.Topic, id uint32)`

  Removes listener with given id from being delivered messages with topic.

#### Publisher

- `Publish(topics.Topic, message.Message) []error`

   Publishes a message on a given topic.

## Implementations

### Listener

- #### Interface

  - #### Listener

    - `Notify(message.Message) error`
    - `Close()`

#### CallbackListener

Invokes a processing function upon receiving a message

#### StreamListener

Loads an atomic (FIFO?) queue with messages to be picked up by worker threads 

#### ChanListener

Relays the messages to a channel

#### multiListener 

(not exported)

Combines multiple listeners

todo: maybe this will need to be exported or otherwise forces placing the 
listener implementation that connects two (such as needed for pipe/socket 
connections) listeners inside this package.

### Broker

