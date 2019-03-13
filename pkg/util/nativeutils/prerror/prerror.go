package prerror

type Priority int

const (
	High = iota
	Medium
	Low
)

type PrError struct {
	Priority
	Err error
}

func New(prio Priority, err error) *PrError {
	return &PrError{
		prio,
		err,
	}
}

func (e *PrError) Error() string {
	return e.Err.Error()
}
