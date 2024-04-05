package errors

import (
	"fmt"
	"log/slog"
)

type WithCause interface{ Cause() error }

type WithHint interface{ Hint() string }

type WithMessage interface{ Message() string }

type Runtime struct {
	msg   string
	cause error
	hint  string
}

func NewRuntimeError(msg string, cause error, hint string) Runtime {
	return Runtime{msg: msg, cause: cause, hint: hint}
}

func (e Runtime) Error() string {
	msgFmt := "%s"
	args := []any{e.msg}
	if e.cause != nil {
		msgFmt += ": %s"
		args = append(args, e.cause.Error())
	}
	if e.hint != "" {
		msgFmt += " (%s)"
		args = append(args, e.hint)
	}
	return fmt.Sprintf(msgFmt, args...)
}

func (e Runtime) Cause() error {
	return e.cause
}

func (e Runtime) Hint() string {
	return e.hint
}

func (e Runtime) Message() string {
	return e.msg
}

// Errorf logs an error message, extracting a hint or cause field if available.
func Errorf(err error, args ...any) {
	msg := err.Error()
	if errh, ok := err.(WithMessage); ok {
		mmsg := errh.Message()
		if mmsg != "" {
			msg = mmsg
		}
	}
	if errh, ok := err.(WithHint); ok {
		hint := errh.Hint()
		if hint != "" {
			args = append([]any{"hint", hint}, args...)
		}
	}
	if errc, ok := err.(WithCause); ok {
		cause := errc.Cause()
		if cause != nil {
			args = append([]any{"cause", cause}, args...)
		}
	}

	slog.Error(msg, args...)
}
