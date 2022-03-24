package config

import (
	"errors"
	"reflect"
)

type StubMapping map[string]interface{}

type FunctionManager struct {
	stubStorage StubMapping
}

func NewFunctionManager(stubStorage StubMapping) *FunctionManager {
	return &FunctionManager{stubStorage: stubStorage}
}

func (m *FunctionManager) Call(funcName string, params ...interface{}) (result interface{}, err error) {
	f := reflect.ValueOf(m.stubStorage[funcName])
	if len(params) != f.Type().NumIn() {
		err = errors.New("The number of params is out of index.")
		return
	}
	in := make([]reflect.Value, len(params))
	for k, param := range params {
		in[k] = reflect.ValueOf(param)
	}

	_ = f.Call(in)
	//result = res[0].Interface()
	return
}
