package svcpanic

type ErrorHandler interface {
	HandlePanic(r interface{}) error
	HandleError(err error) error
}

func RecoverPanic(errPtr *error, handler ErrorHandler) {
	if r := recover(); r != nil {
		*errPtr = handler.HandlePanic(r)
	} else if *errPtr != nil {
		*errPtr = handler.HandleError(*errPtr)
	}
}
