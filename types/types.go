package types

type ResponseState int

const (
	SERVED    ResponseState = 0
	MODIFIED  ResponseState = 1
	UNTOUCHED ResponseState = 2
)
