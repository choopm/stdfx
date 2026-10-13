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

package configdoc

import (
	"encoding"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/creasty/defaults"
)

// Kind is how a setting is written in a config file.
type Kind int

const (
	// Scalar is a single value
	Scalar Kind = iota

	// Struct is a mapping of the settings in [Setting.Fields]
	Struct

	// List is a sequence of elements as [Setting.Elem] describes them
	List

	// Map is a mapping of keys an operator chooses to elements as
	// [Setting.Elem] describes them
	Map

	// Any is a value of no fixed shape
	Any
)

// ElemKey is the key of the [Setting.Elem] of a list or map, and the segment
// standing for any element in a [Setting.Path].
const ElemKey = "*"

// Setting is one key of a config file.
type Setting struct {
	// Default is what the setting decodes to when its key is omitted while
	// its parent is written. Below an element of a list or map, and below a
	// pointer which defaults to nil, that is the zero value: such a struct is
	// allocated by the decoder rather than by the defaults of its tags.
	Default reflect.Value

	// Doc documents the setting, an override of [Docs.Overrides] applied
	Doc Doc

	// Elem describes the elements of a [List] or [Map]
	Elem *Setting

	// Enum are the values the setting takes, if it names them
	Enum []Enum

	// Example is yaml standing in for the setting, see [Docs.Examples]
	Example string

	// Fields are the settings of a [Struct] in the order they are declared
	Fields []*Setting

	// GoName is the name of the struct field
	GoName string

	// Hidden reports a setting which is never written, see [Docs.Hidden]
	Hidden bool

	// Key is the key of the setting in its mapping, [ElemKey] for an element
	Key string

	// Kind is how the setting is written
	Kind Kind

	// Parent is the setting containing this one, nil at the top level
	Parent *Setting

	// Path are the keys from the top level to this setting, joined by dots
	Path string

	// Tag are the struct tags of the field
	Tag reflect.StructTag

	// Type is the type of the field
	Type reflect.Type

	// TypeDoc is the doc comment of the named type of the setting
	TypeDoc string

	// declaring is the field in the struct declaring it, which is where its
	// doc comment and its doc links are resolved
	declaring Key

	// index leads from the struct containing the setting to its field,
	// through any squashed struct on the way
	index []int
}

// Schema is a config file as a struct describes it.
type Schema struct {
	// Fields are the settings at the top level
	Fields []*Setting

	// Header is the comment heading the file
	Header string

	// keys are the keys of the fields of every struct reached, which is what
	// resolves a doc link to a field into the key an operator writes
	keys map[Key]string
}

// textUnmarshaler is the interface making a type a leaf
var textUnmarshaler = reflect.TypeFor[encoding.TextUnmarshaler]()

// NewSchema returns the schema of a config file decoding to T, documented by
// docs, or error. A nil docs documents nothing.
func NewSchema[T any](docs *Docs) (*Schema, error) {
	if docs == nil {
		docs = &Docs{}
	}

	t := new(T)
	if err := defaults.Set(t); err != nil {
		return nil, fmt.Errorf("setting config defaults: %w", err)
	}

	rt := reflect.TypeFor[T]()
	if rt.Kind() != reflect.Struct {
		return nil, fmt.Errorf("config type %s is not a struct", rt)
	}

	b := &builder{
		docs:   docs,
		hidden: map[string]bool{},
		schema: &Schema{
			Header: docs.Header,
			keys:   map[Key]string{},
		},
		visiting: map[reflect.Type]bool{},
	}
	for _, path := range docs.Hidden {
		b.hidden[path] = true
	}

	fields, err := b.structFields(nil, rt, reflect.ValueOf(t).Elem(), "")
	if err != nil {
		return nil, err
	}
	b.schema.Fields = fields

	return b.schema, nil
}

// Lookup returns the setting at path or nil. A segment below a list or map
// names an element, whatever it is: [ElemKey], an index or a key.
func (s *Schema) Lookup(path ...string) *Setting {
	fields := s.Fields
	var found *Setting
	for _, key := range path {
		if found != nil && (found.Kind == List || found.Kind == Map) {
			found = found.Elem
			fields = found.Fields
			continue
		}

		i := slices.IndexFunc(fields, func(f *Setting) bool {
			return strings.EqualFold(f.Key, key)
		})
		if i < 0 {
			return nil
		}
		found = fields[i]
		fields = found.Fields
	}

	return found
}

// Walk calls fn for every setting depth first in the order of the file, and
// skips the settings below one for which fn returns false.
func (s *Schema) Walk(fn func(*Setting) bool) {
	var walk func([]*Setting)
	walk = func(settings []*Setting) {
		for _, setting := range settings {
			if !fn(setting) {
				continue
			}
			walk(setting.Fields)
			if setting.Elem != nil && fn(setting.Elem) {
				walk(setting.Elem.Fields)
			}
		}
	}
	walk(s.Fields)
}

// NewElement returns a fresh element of the list or map s with the defaults
// of its struct tags set.
//
// The decoder allocates an element as a zero value, so those defaults only
// apply where they are written - which is why an editor adding an element
// writes them out.
func (s *Setting) NewElement() (reflect.Value, error) {
	if s.Elem == nil {
		return reflect.Value{}, fmt.Errorf("%s is neither a list nor a map", s.Path)
	}

	v := reflect.New(deref(s.Elem.Type))
	if v.Elem().Kind() == reflect.Struct {
		if err := defaults.Set(v.Interface()); err != nil {
			return reflect.Value{}, fmt.Errorf("setting defaults of %s: %w", s.Path, err)
		}
	}
	if s.Elem.Type.Kind() == reflect.Pointer {
		return v, nil
	}

	return v.Elem(), nil
}

// builder builds a [Schema]
type builder struct {
	docs   *Docs
	hidden map[string]bool
	schema *Schema

	// visiting are the struct types on the way to the current one, which
	// turns a recursive type into [Any] instead of recursing forever
	visiting map[reflect.Type]bool
}

// structFields returns the settings of the struct type t whose fields default
// to those of v, below parent at path, or error.
func (b *builder) structFields(parent *Setting, t reflect.Type, v reflect.Value, path string) ([]*Setting, error) {
	b.visiting[t] = true
	defer delete(b.visiting, t)

	var settings []*Setting
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}

		name, options, _ := strings.Cut(f.Tag.Get("mapstructure"), ",")
		if name == "-" {
			continue
		}

		fv := field(v, i)
		if slices.Contains(strings.Split(options, ","), "squash") {
			st := deref(f.Type)
			if st.Kind() != reflect.Struct {
				return nil, fmt.Errorf("%s.%s: squashing %s, which is no struct", t, f.Name, f.Type)
			}

			squashed, err := b.structFields(parent, st, indirect(fv, st), path)
			if err != nil {
				return nil, err
			}
			if len(squashed) > 0 {
				// the group the squashed field starts is the group of its
				// first field
				squashed[0].Doc.Block = squashed[0].Doc.Block ||
					b.docs.Fields[Key{Field: f.Name, Package: t.PkgPath(), Type: t.Name()}].Block
			}
			for _, s := range squashed {
				s.index = append([]int{i}, s.index...)
			}
			settings = append(settings, squashed...)
			continue
		}
		if slices.Contains(strings.Split(options, ","), "remain") {
			return nil, fmt.Errorf("%s.%s: ',remain' is not supported", t, f.Name)
		}
		if name == "" {
			if b.docs.SkipUntagged {
				continue
			}
			name = f.Name
		}

		declaring := Key{Field: f.Name, Package: t.PkgPath(), Type: t.Name()}
		s := &Setting{
			Default:   fv,
			Doc:       b.docs.Fields[declaring],
			GoName:    f.Name,
			Key:       name,
			Parent:    parent,
			Path:      join(path, name),
			Tag:       f.Tag,
			Type:      f.Type,
			declaring: declaring,
			index:     []int{i},
		}
		b.schema.keys[declaring] = name
		if err := b.describe(s); err != nil {
			return nil, err
		}
		settings = append(settings, s)
	}

	return settings, nil
}

// describe fills in what s is from its type, its default and the docs, or
// returns error.
func (b *builder) describe(s *Setting) error {
	if text, ok := b.docs.Overrides[s.Path]; ok {
		s.Doc.Text = text
	}
	s.Example = b.docs.Examples[s.Path]
	s.Hidden = b.hidden[s.Path]

	t := deref(s.Type)
	if t.Name() != "" {
		s.TypeDoc = b.docs.Types[TypeKey{Name: t.Name(), Package: t.PkgPath()}]
	}

	switch {
	case b.leaf(s.Type):
		s.Kind = Scalar
		s.Enum = b.enum(t, s.Tag, false)

	case t.Kind() == reflect.Struct:
		if b.visiting[t] {
			s.Kind = Any
			return nil
		}
		s.Kind = Struct
		fields, err := b.structFields(s, t, indirect(s.Default, t), s.Path)
		if err != nil {
			return err
		}
		s.Fields = fields

	case t.Kind() == reflect.Slice || t.Kind() == reflect.Array:
		s.Kind = List
		return b.element(s, t.Elem())

	case t.Kind() == reflect.Map:
		if t.Key().Kind() != reflect.String {
			s.Kind = Any
			return nil
		}
		s.Kind = Map
		return b.element(s, t.Elem())

	case t.Kind() == reflect.Interface:
		s.Kind = Any

	case slices.Contains([]reflect.Kind{reflect.Chan, reflect.Func, reflect.UnsafePointer}, t.Kind()):
		return fmt.Errorf("%s: %s cannot be configured", s.Path, s.Type)

	default:
		s.Kind = Scalar
		s.Enum = b.enum(t, s.Tag, false)
	}

	return nil
}

// element describes the elements of type t of the list or map s, or returns
// error.
func (b *builder) element(s *Setting, t reflect.Type) error {
	s.Elem = &Setting{
		// the decoder allocates an element as a zero value
		Default:   reflect.Zero(t),
		Key:       ElemKey,
		Parent:    s,
		Path:      join(s.Path, ElemKey),
		Type:      t,
		declaring: s.declaring,
	}
	if err := b.describe(s.Elem); err != nil {
		return err
	}
	if s.Elem.Kind == Scalar && len(s.Elem.Enum) == 0 {
		s.Elem.Enum = b.enum(deref(t), s.Tag, true)
	}

	return nil
}

// leaf reports whether a value of type t is written as a single value rather
// than as the settings of its fields or elements.
func (b *builder) leaf(t reflect.Type) bool {
	d := deref(t)
	if slices.Contains(b.docs.Leaves, t) || slices.Contains(b.docs.Leaves, d) {
		return true
	}
	if reflect.PointerTo(d).Implements(textUnmarshaler) {
		return true
	}

	// a byte slice is a string to a decoder
	return d.Kind() == reflect.Slice && d.Elem().Kind() == reflect.Uint8
}

// enum returns the values of the named string type t, or else those a
// `validate:"oneof=..."` tag names - for the elements of a slice those
// following `dive` when element is true.
func (b *builder) enum(t reflect.Type, tag reflect.StructTag, element bool) []Enum {
	if t.Kind() == reflect.String && t.Name() != "" {
		if enums := b.docs.Enums[TypeKey{Name: t.Name(), Package: t.PkgPath()}]; len(enums) > 0 {
			return enums
		}
	}

	rules := strings.Split(tag.Get("validate"), ",")
	if dive := slices.Index(rules, "dive"); dive >= 0 {
		if element {
			rules = rules[dive+1:]
		} else {
			rules = rules[:dive]
		}
	} else if element {
		return nil
	}

	for _, rule := range rules {
		values, ok := strings.CutPrefix(rule, "oneof=")
		if !ok {
			continue
		}

		var enums []Enum
		for _, value := range splitOneOf(values) {
			enums = append(enums, Enum{Value: value})
		}
		return enums
	}

	return nil
}

// splitOneOf splits the values of a oneof rule, which are separated by spaces
// unless single quoted
func splitOneOf(values string) []string {
	var (
		out     []string
		current strings.Builder
		quoted  bool
	)
	flush := func() {
		if current.Len() > 0 {
			out = append(out, current.String())
			current.Reset()
		}
	}
	for _, r := range values {
		switch {
		case r == '\'':
			quoted = !quoted
		case r == ' ' && !quoted:
			flush()
		default:
			current.WriteRune(r)
		}
	}
	flush()

	return out
}

// deref returns t without any pointers
func deref(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	return t
}

// indirect returns the value v points to, or a zero value of t where there is
// none - which is what a decoder allocates in its place
func indirect(v reflect.Value, t reflect.Type) reflect.Value {
	for v.IsValid() && v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return reflect.Zero(t)
		}
		v = v.Elem()
	}
	if !v.IsValid() {
		return reflect.Zero(t)
	}

	return v
}

// field returns field i of the struct v, or an invalid value if v is invalid
func field(v reflect.Value, i int) reflect.Value {
	if !v.IsValid() {
		return reflect.Value{}
	}

	return v.Field(i)
}

// join returns the path of key below path
func join(path, key string) string {
	if path == "" {
		return key
	}

	return path + "." + key
}
