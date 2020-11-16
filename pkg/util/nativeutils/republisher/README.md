# Republisher and race conditions

Although not immediately evident when looking at the code, when tests fail due to a race condition seemingly triggered within the Republisher, this is most often just a false positive. Concurrent reads/writes on messages happening in the `Republisher` and `consensus.Components` are triggered primarily because the test does not reproduce the whole extent of production reality. 

## Marshaled Form Of Messages Is Cached At Reception

Before the messages hit the `Republisher`, they are unmarshaled into a `message.simple` by the `peer.messageRouter` and pushed further. `message.simple` calls `initPayloadBuffer(messageBuffer)` which essentially just caches the marshaled form as it was received by the peer. When `message.Marshal` is triggered, the function first type-checks if it is handling a `message.simple`. If this is the case, the cached buffer gets immediately returned and no actual marshaling happens. 

This means that the republisher is extremely unlikely to ever trigger a race condition, since it unknowngly handles a `message.simple` most (if not all) of the time.

## Marshaling a message.simple cannot trigger a race condition

The `simple.marshaled` as a field is not exposed and can only be accessed during the sequential `Unmarshal` and `Marshal` actions within the `message` package. This means that it cannot be the targe of race conditions between the `Republisher` and the various consensus components, which are both external to the `message` package. 

## Read-only race conditions are possible only within the validation callbacks

The `Republisher` gets injected validation callback functions. These are the only places where concurrent reads can happen (as the validation does not write on the message itself). However, validation functions call-back into the components and therefore read-locks are a responsibility of the components and executed outside of the `Republisher` knowledge.

## Race conditions in the `Republisher` are possible only when the Republisher handles messages other than `message.simple`
 
A message is generally not a `message.simple` instance when it gets generated internally and not received by the peer. However, messages generated internally do not usually get propagated through the `Republisher`, but gossiped directly by the consensus components and therefore they do not really represent a problem. This require double checking though.

## Future improvements

Since it is always best to be explicit about things, we could simply export a `Cached() *bytes.Buffer` from the `message.Message` interface and let the `Republisher` either throw an error when the method returns an empty buffer, or lock the message when calling `message.Marshal`.
