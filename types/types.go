package types

type ResponseState bool

const (
	SERVED    ResponseState = true
	UNTOUCHED ResponseState = false
)
