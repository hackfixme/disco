package types

type ErrNoResult struct {
	Msg string
}

func (e ErrNoResult) Error() string {
	return e.Msg
}
