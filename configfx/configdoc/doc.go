/*
Copyright 2024 Christoph Hoopmann

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

// Package configdoc describes a config struct as the file an operator edits:
// which keys it has, in which order, what each of them means and what it
// defaults to - and writes that file.
//
// The words come from the doc comments of the struct fields, which a program
// does not carry at runtime. The configdoc command extracts them into a
// generated file holding [Docs], [NewSchema] combines them with what reflection
// tells about the struct, and [Render] lays a config file out as that schema
// says while keeping every value it holds as it was written.
package configdoc

import "reflect"

// Key identifies a field by the struct type declaring it.
type Key struct {
	// Field is the name of the field
	Field string

	// Package is the import path of the package declaring Type
	Package string

	// Type is the name of the struct type
	Type string
}

// TypeKey identifies a named type.
type TypeKey struct {
	// Name is the name of the type
	Name string

	// Package is the import path of the package declaring the type
	Package string
}

// Doc documents a field.
type Doc struct {
	// Block reports a blank line between the field and the field declared
	// before it, which is how a struct groups its fields
	Block bool

	// Line is the comment trailing the field on its line
	Line string

	// Text is the doc comment of the field
	Text string
}

// Enum is a constant a named string type declares, one value a setting of
// that type takes.
type Enum struct {
	// Doc is the doc comment of the constant
	Doc string

	// Value is the value of the constant
	Value string
}

// Docs documents a config struct and every type it reaches.
//
// Enums, Fields and Types are extracted from source by the configdoc command.
// The remaining fields are what the application adds on top of that.
type Docs struct {
	// Enums are the constants of named string types
	Enums map[TypeKey][]Enum

	// Fields are the docs of struct fields
	Fields map[Key]Doc

	// Types are the doc comments of named types
	Types map[TypeKey]string

	// Examples are written as a comment below the doc of a setting, keyed by
	// its path - see [Setting.Path]. An example is yaml as it would appear in
	// place of the setting.
	Examples map[string]string

	// Header is the comment heading a rendered file
	Header string

	// Hidden are the paths of settings which are never written. A hidden key
	// found in a file is dropped like a key nothing reads.
	Hidden []string

	// Leaves are types written as a single value rather than as the settings
	// of their fields. A type which implements [encoding.TextUnmarshaler] is
	// one without being listed.
	Leaves []reflect.Type

	// Overrides replace the doc comment of a setting, keyed by its path. They
	// are for a setting whose struct belongs to a library, where the
	// application has something more specific to say than the library can.
	Overrides map[string]string

	// SkipUntagged leaves out exported fields without a mapstructure tag,
	// which mapstructure would otherwise decode under their field name. It is
	// for an application whose untagged fields are derived at runtime rather
	// than configured.
	SkipUntagged bool
}
