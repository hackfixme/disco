package types

type ConnType uint8

const (
	ConnTypeHTTP ConnType = iota + 1
	ConnTypeTLS
)

const ConnTypeKey = "connectionType"
