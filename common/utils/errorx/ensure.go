package errorx

import (
	"fmt"
)

func EnsureStack(inputErr interface{}, skip int) error {
	var err error
	switch newErrTyped := inputErr.(type) {
	case error:
		if _, ok := newErrTyped.(StackTraceWrapper); ok {
			return newErrTyped
		}
		err = WithStack(newErrTyped)
	default:
		err = New(fmt.Sprintf("%+v", inputErr))
	}
	errWithStack := err.(StackTraceWrapper)
	errWithStack.GetStack().Skip(skip)
	return err
}
