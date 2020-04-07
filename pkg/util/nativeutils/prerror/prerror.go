package prerror

// Priority is the priority High,Medium,Low
type Priority int

//nolint:golint
type PrError struct {
	Priority
	Err error
}

//nolint:golint
func (e *PrError) Error() string {
	return e.Err.Error()
}
