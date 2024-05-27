package register

func NonRecoverable(err string) NonRecoverableError {
	return NonRecoverableError{Message: err}
}

type NonRecoverableError struct {
	Message string
}

func (e NonRecoverableError) Error() string {
	return e.Message
}
