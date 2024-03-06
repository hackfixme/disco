package errors

type WithCause interface{ Cause() error }

type WithHint interface{ Hint() string }

type Runtime struct {
	msg   string
	cause error
	hint  string
}

func NewRuntimeError(msg string, cause error, hint string) Runtime {
	return Runtime{msg: msg, cause: cause, hint: hint}
}

func (e Runtime) Error() string {
	return e.msg
}

func (e Runtime) Cause() error {
	return e.cause
}

func (e Runtime) Hint() string {
	return e.hint
}
