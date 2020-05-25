package payload

// Safe is used for protection from race conditions
type Safe interface {
	// Copy performs a deep copy of an object
	Copy() Safe
}
