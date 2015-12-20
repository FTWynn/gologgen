package loghelper

import (
	"runtime"
)

// Log15LazyFunctionName is a helper function to get the name of the current function
func Log15LazyFunctionName() (functionName string) {
	tempStorage := make([]uintptr, 10)
	runtime.Callers(14, tempStorage)
	functionName = runtime.FuncForPC(tempStorage[0]).Name()
	return
}
