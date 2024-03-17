package types

type ErrNoResult struct {
	Msg string
}

func (e ErrNoResult) Error() string {
	return e.Msg
}

type ErrReference struct {
	Msg   string
	Cause error
}

func (e ErrReference) Error() string {
	return e.Msg
}
