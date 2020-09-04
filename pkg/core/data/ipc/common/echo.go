package common

// EchoRequest is used to ping the Rusk server.
type EchoRequest struct {
	Message string `json:"message"`
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (e *EchoRequest) Copy() *EchoRequest {
	m := ""
	m += e.Message
	return &EchoRequest{m}
}

// EchoResponse is what the Rusk server sends back after an `EchoRequest`.
type EchoResponse struct {
	Message string `json:"message"`
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (e *EchoResponse) Copy() *EchoResponse {
	m := ""
	m += e.Message
	return &EchoResponse{m}
}
