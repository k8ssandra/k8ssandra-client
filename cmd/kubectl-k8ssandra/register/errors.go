package register

type RetryableError struct {
	Message string
}

func (e RetryableError) Error() string {
	return e.Message
}

type NonRecoverableError struct {
	Message string
}

func (e NonRecoverableError) Error() string {
	return e.Message
}
