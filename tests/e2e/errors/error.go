package errors

type timeout interface {
	Timeout() bool
}

func IsTimeout(err error) bool {
	t, ok := err.(timeout)
	if !ok {
		return false
	}
	return t.Timeout()
}

type TimeoutErr struct {
	err string
}

func (t *TimeoutErr) Error() string {
	return t.err
}

func (t *TimeoutErr) Timeout() bool {
	return true
}

func NewTimeoutErr(err string) *TimeoutErr {
	return &TimeoutErr{err: err}
}
