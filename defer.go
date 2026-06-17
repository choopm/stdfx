/*
Copyright 2026 Christoph Hoopmann

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package stdfx

import (
	"fmt"
	"reflect"
)

var _errType = reflect.TypeFor[error]()

// Defer helps to capture errors of deferred funcs.
// It will invoke fn (with optional arguments) and append
// any error from return arguments to err.
// You need to make sure that you include err when returning from the
// calling func - even if it was nil deferred calls to Defer might change this.
// The argument index of error when returning does not matter.
// All error arguments will be checked and appended if not nil.
//
// Simple usage example without parameters:
//
//	f, err := os.Open("/tmp/file")
//	if err != nil {
//		return err
//	}
//	stdfx.Defer(&err, f.Close)
//
//	// ... do other things
//
//	return something, err // include err in return
//
// If fn requires parameters, just pass them as well:
//
//	// ...
//	defer stdfx.Defer(&err, myFuncWithArgs, arg1, arg2)
//	// ...
//
// When used in testing, you want to add an assert to the top of your
// test function because you can't return an error here. This check is
// executed last (first usage of defer) and checks err for the last time:
//
//	func TestMyFunc(t *testing.T) {
//		var err error
//		defer stdfx.Defer(&err, assert.NoError, t, err)
//
//		// ...
//	}
func Defer(err *error, fn any, params ...any) {
	// sanity check
	f := reflect.ValueOf(fn)
	if len(params) != f.Type().NumIn() {
		// we can't invoke fn if lacking the required paramaters
		panic("internal: invalid number of arguments for fn given to Defer")
	}

	// build parameters
	in := make([]reflect.Value, len(params))
	for i, param := range params {
		in[i] = reflect.ValueOf(param)
	}

	// call fn, check all return arguments
	r := f.Call(in)
	for _, r := range r {
		if r.Type() != _errType {
			continue
		}

		// error type
		_e := r.Interface()
		if _e == nil {
			continue
		}

		// sanity check
		e, ok := _e.(error)
		if !ok {
			continue
		}

		// append it
		if *err != nil {
			*err = fmt.Errorf("%s: %s", e, *err)
		} else {
			*err = e
		}
	}
}
