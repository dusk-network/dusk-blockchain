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

## Specification

### Broker

A broker processes subscriptions and relays messages to `Listener`s

#### Interfaces

- #### Subscriber

    - `Subscribe(topic topics.Topic, listener Listener) uint32`

      Adds the listener to the list of brokers to be sent messages with a given
      topic

      When messages arrive, the `Notify` method of the
      `Listener` is called with the message.

    - `Unsubscribe(topic topics.Topic, id uint32)`

      Removes listener with given id from being delivered messages with topic.

- #### Publisher

    - `Publish(topics.Topic, message.Message) []error`

      Publishes a message on a given topic.

- #### Multicaster

    - `AddDefaultTopic(topic topics.Topic)`

      Add a topic to the multilistener (note there is no remove, so a new
      `Multicaster` is required to implement a remove)

    - `SubscribeDefault(listener Listener) (topic uint32)`

      Subscriber adds a Listener to the default multilistener

#### Implementations

- #### EventBus

  This implementation of the `Broker` and `Multicaster` interfaces serves to
  implement the basic in-process message dispatch bus in the `eventbus` package.

- #### IPCBus

  **TODO:**

  This will composit the EventBus type and extend its methods with shadowing to
  implement a local broker for client processes that mirrors requests to the
  IPCBus by proxying protocol messages of interest back and forth over the 
  connection (ReaderWriterCloser interface implementation) which is created 
  by running a command that attaches via stdin/out and exposes interrupt and 
  kill methods for use in interrupt processing of the application, including 
  shutting down if the connection is closed, so the process terminates when 
  the controller terminates regardless of whether it formally signals this 
  (shutdown signal allows cleanup of network and pipe buffers).

  Because this is entirely connected via interfaces this implementation will 
  live in [dusk-ipc](https://github.com/dusk-network/dusk-ipc), where future 
  work will go on this.

### Listener

#### Interface

- `Notify(message.Message) error`

  When a message is published the `Notify` method of a listener is called,
  relaying the message.

- `Close()`

  Disconnects the `Listener` to stop receiving messages.

#### Implementations

- #### CallbackListener

  Invokes a processing function upon receiving a message

- #### StreamListener

  Loads an atomic (FIFO?) queue with messages to be picked up by worker threads

- #### ChanListener

  Relays the messages to a channel

- #### multiListener

  Combines multiple listeners. This is part of the `Multicaster` interface
