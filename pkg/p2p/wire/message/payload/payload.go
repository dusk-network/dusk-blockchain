package payload

// SafePayload is used for protection from race conditions
type SafePayload interface {
	// Copy performs a deep copy of an object
	Copy() SafePayload
}
