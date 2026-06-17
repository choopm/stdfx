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

package stdfx_test

import (
	"errors"
	"testing"

	"github.com/choopm/stdfx"
	"github.com/stretchr/testify/assert"
)

func TestDefer(t *testing.T) {
	fn0 := func() (string, error) {
		return "okay", nil
	}
	fn1 := func() error {
		return errors.New("fn1")
	}
	fn2 := func(v string) error {
		return errors.New(v)
	}
	fn3 := func(v string, _ bool) (string, error) {
		return v, errors.New(v)
	}
	fn4 := func() (bool, error) {
		return true, nil
	}

	// dummy call to instantiate the err variable
	a, err := fn0()
	assert.Equal(t, "okay", a)
	assert.NoError(t, err)

	// last func to be executed, shall assert all errors are collected
	defer func() {
		assert.Error(t, err)
		assert.ErrorContains(t, err, "fn1")
		assert.ErrorContains(t, err, "fn2")
		assert.ErrorContains(t, err, "fn3")
		assert.Equal(t, err.Error(), "fn1: fn2: fn3")
	}()

	defer stdfx.Defer(&err, fn1)
	defer stdfx.Defer(&err, fn2, "fn2")
	defer stdfx.Defer(&err, fn3, "fn3", true)
	defer stdfx.Defer(&err, fn4)
}
