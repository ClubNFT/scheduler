package task

import (
	"fmt"
	"reflect"
	"runtime"
)

// Function is a pointer to the callback function
type Function interface{}

// FunctionMeta holds information about function such as name and parameters.
type FunctionMeta struct {
	Name string
}

func Translate(function Function) (FunctionMeta, error) {
	funcValue := reflect.ValueOf(function)
	if funcValue.Kind() != reflect.Func {
		return FunctionMeta{}, fmt.Errorf("Provided function value is not an actual function")
	}
	name := runtime.FuncForPC(funcValue.Pointer()).Name()
	return FunctionMeta{
		Name: name,
	}, nil
}
