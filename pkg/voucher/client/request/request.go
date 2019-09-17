package request

// Request will parametrize all the communication with the voucher
type Request interface {
	Operation() *Operation
	Payload() *[]byte
}
