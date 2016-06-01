package types

type ResponseState int

const (
	UNTOUCHED ResponseState = 0
	MODIFIED  ResponseState = 1
	SERVED    ResponseState = 2
)
