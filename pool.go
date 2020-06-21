// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package jsonutils

import (
	"runtime"
	"sync"
)

var (
	poolInt    *sync.Pool
	poolFloat  *sync.Pool
	poolString *sync.Pool
	poolArray  *sync.Pool
)

func init() {
	poolInt = &sync.Pool{
		New: func() interface{} {
			v := &JSONInt{}
			return v
		},
	}
	poolFloat = &sync.Pool{
		New: func() interface{} {
			v := &JSONFloat{}
			return v
		},
	}
	poolString = &sync.Pool{
		New: func() interface{} {
			v := &JSONString{}
			return v
		},
	}
	poolArray = &sync.Pool{
		New: func() interface{} {
			v := &JSONArray{}
			return v
		},
	}
}

func jsonObjFinalize(obj JSONObject) {
	switch objv := obj.(type) {
	case *JSONString:
		objv.data = ""
		poolString.Put(obj)
	case *JSONInt:
		poolInt.Put(obj)
	case *JSONFloat:
		poolFloat.Put(obj)
	case *JSONDict:
		for k, v := range objv.data {
			objv.data[k] = nil
			jsonObjFinalize(v)
		}
		objv.data = nil
	case *JSONArray:
		for k, v := range objv.data {
			objv.data[k] = nil
			jsonObjFinalize(v)
		}
		if len(objv.data) > 0 {
			objv.data = objv.data[:0]
		}
		poolArray.Put(obj)
	}
}

func poolSetFinalizer(obj JSONObject) {
	switch obj.(type) {
	case *JSONDict, *JSONArray:
		runtime.SetFinalizer(obj, jsonObjFinalize)
	}
}

func poolGetInt() *JSONInt {
	return poolInt.Get().(*JSONInt)
}

func poolGetFloat() *JSONFloat {
	return poolFloat.Get().(*JSONFloat)
}

func poolGetString() *JSONString {
	return poolString.Get().(*JSONString)
}

func poolGetArray() *JSONArray {
	return poolArray.Get().(*JSONArray)
}

func poolNewInt(data int64) *JSONInt {
	v := poolInt.Get().(*JSONInt)
	v.data = data
	return v
}

func poolNewFloat(data float64) *JSONFloat {
	v := poolFloat.Get().(*JSONFloat)
	v.data = data
	return v
}

func poolNewString(data string) *JSONString {
	v := poolString.Get().(*JSONString)
	v.data = data
	return v
}

func poolNewArray(data ...JSONObject) *JSONArray {
	v := poolArray.Get().(*JSONArray)
	v.data = data
	return v
}
