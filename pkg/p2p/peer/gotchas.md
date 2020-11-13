# Message processing

It is required that a message which comes from the wire implements the `payload.Safe` interface and has an unmarshalling function that can be called through `message.Unmarshal`. Otherwise, the message will be decoded as nil, and more often than not cause a panic. 
