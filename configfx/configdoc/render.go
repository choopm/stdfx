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
	"bytes"
	"encoding"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

const (
	// lineWidth is how wide a rendered comment may get
	lineWidth = 80

	// minCommentWidth keeps a deeply nested comment readable
	minCommentWidth = 40
)

// Report is what [Schema.Render] changed about a file beyond its layout and
// its comments.
type Report struct {
	// Added are the paths of settings the file omitted, which were written
	// with their default
	Added []string

	// Dropped are the paths of keys no setting reads, which were removed
	Dropped []string

	// Mismatched are the paths of values of another shape than their
	// setting, which were kept as they were written
	Mismatched []string
}

// Render returns the yaml config file src laid out as s says and what that
// changed, or error.
//
// The settings follow the order of their struct fields, each under its doc
// comment, with a blank line wherever the struct has one. A value is kept as
// it was written, quoting and all - only its comments are replaced. A setting
// src omits is added with its default, and a key no setting reads is dropped.
func (s *Schema) Render(src []byte) ([]byte, *Report, error) {
	doc, err := Parse(src)
	if err != nil {
		return nil, nil, err
	}

	return s.RenderDocument(doc)
}

// RenderDocument returns doc laid out as s says and what that changed, or
// error. See [Schema.Render].
func (s *Schema) RenderDocument(doc *Document) ([]byte, *Report, error) {
	r := &renderer{
		report: &Report{},
		schema: s,
	}

	root, orphans, err := r.mapping(s.Fields, doc.root, "", 0, true, true)
	if err != nil {
		return nil, nil, err
	}
	root.Style &^= yaml.FlowStyle

	out := &yaml.Node{
		Content:     []*yaml.Node{root},
		HeadComment: joinComments(commentLines(s.Header, false), orphans),
		Kind:        yaml.DocumentNode,
	}

	var b bytes.Buffer
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(out); err != nil {
		return nil, nil, fmt.Errorf("render config: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, nil, fmt.Errorf("render config: %w", err)
	}

	// a blank line between nested keys comes out indented
	lines := strings.Split(b.String(), "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			lines[i] = ""
		}
	}

	return []byte(strings.Join(lines, "\n")), r.report, nil
}

// renderer renders a document as a [Schema] says
type renderer struct {
	report *Report
	schema *Schema
}

// mapping returns the mapping of the settings fields at path, indented by
// indent, holding the values of src, which may be nil, and the comment of
// what it holds no value for, or error.
//
// A setting src omits is added with its default where complete is true, and
// left out otherwise. Settings are commented where comment is true.
//
// A setting without a value to write - a nil default - is described only if
// it has an example, above the setting following it. The last settings of a
// mapping are described at its top instead, and those of a mapping holding
// nothing at all, which is written as {} and cannot hold a comment, by the
// comment returned for the key of the mapping.
func (r *renderer) mapping(fields []*Setting, src *yaml.Node, path string, indent int, complete, comment bool) (*yaml.Node, string, error) {
	entries, err := r.entries(src, path)
	if err != nil {
		return nil, "", err
	}

	out := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	var commentedOut []string
	for _, f := range fields {
		if f.Hidden {
			continue
		}

		var (
			value   *yaml.Node
			orphans string
		)
		if i := slices.IndexFunc(entries, func(e *entry) bool {
			return strings.EqualFold(e.key.Value, f.Key)
		}); i >= 0 {
			entries[i].used = true
			value, orphans, err = r.value(f, entries[i].value, join(path, f.Key), indent, complete, comment)
			if err != nil {
				return nil, "", err
			}
		} else if complete {
			value, orphans, err = r.absent(f, join(path, f.Key), indent)
			if err != nil {
				return nil, "", err
			}
			if value == nil {
				// nothing to write in place of a nil default
				if comment && f.Example != "" {
					commentedOut = append(commentedOut, r.comment(f, indent))
				}
				continue
			}
			if f.Kind != Struct {
				// a struct is reported by its settings
				r.report.Added = append(r.report.Added, join(path, f.Key))
			}
		} else {
			continue
		}

		key := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: f.Key}
		if comment {
			key.HeadComment = joinComments(append(commentedOut, r.comment(f, indent), orphans)...)
			commentedOut = nil
			if len(out.Content) > 0 && (indent == 0 || f.Doc.Block) {
				// a blank line is what separates a group of settings
				key.HeadComment = "\n" + key.HeadComment
			}

			if line := strings.TrimSpace(f.Doc.Line); line != "" {
				if value.Kind == yaml.ScalarNode {
					value.LineComment = "# " + line
				} else {
					key.LineComment = "# " + line
				}
			}
		}
		out.Content = append(out.Content, key, value)
	}

	for _, e := range entries {
		if !e.used {
			r.report.Dropped = append(r.report.Dropped, join(path, e.key.Value))
		}
	}

	orphans := joinComments(commentedOut...)
	if len(out.Content) == 0 {
		out.Style = yaml.FlowStyle
		return out, orphans, nil
	}
	// the comment of a mapping gives way to that of its first key
	out.Content[0].HeadComment = joinComments(orphans, out.Content[0].HeadComment)

	return out, "", nil
}

// entry is a key of a mapping in the source
type entry struct {
	key   *yaml.Node
	used  bool
	value *yaml.Node
}

// entries returns the keys of the mapping src at path, or error for a key
// given twice
func (r *renderer) entries(src *yaml.Node, path string) ([]*entry, error) {
	if src == nil || src.Kind != yaml.MappingNode {
		return nil, nil
	}

	var entries []*entry
	for i := 0; i+1 < len(src.Content); i += 2 {
		k := src.Content[i]
		if slices.ContainsFunc(entries, func(e *entry) bool {
			return strings.EqualFold(e.key.Value, k.Value)
		}) {
			return nil, fmt.Errorf("%s: duplicate key (line %d)", join(path, k.Value), k.Line)
		}
		entries = append(entries, &entry{key: k, value: src.Content[i+1]})
	}

	return entries, nil
}

// value returns the value src of the setting f at path and the comment of
// what it holds no value for, or error. See [renderer.mapping] for the meaning
// of complete and comment.
func (r *renderer) value(f *Setting, src *yaml.Node, path string, indent int, complete, comment bool) (*yaml.Node, string, error) {
	null := src.Kind == yaml.ScalarNode && src.Tag == "!!null"

	switch f.Kind {
	case Struct:
		switch {
		case src.Kind == yaml.MappingNode:
			return r.mapping(f.Fields, src, path, indent+2, complete, comment)
		case null && f.Type.Kind() != reflect.Pointer:
			// the decoder leaves a struct given no value as it is
			return r.mapping(f.Fields, nil, path, indent+2, complete, comment)
		case null:
			return clone(src, true), "", nil
		}

	case List:
		switch {
		case src.Kind == yaml.SequenceNode && f.Elem.Kind == Struct:
			out := clone(src, true)
			for i, item := range src.Content {
				if item.Kind != yaml.MappingNode {
					r.report.Mismatched = append(r.report.Mismatched, join(path, fmt.Sprint(i)))
					continue
				}

				// an element is decoded from a zero value, so it is never
				// completed, and only the first one carries comments
				m, _, err := r.mapping(f.Elem.Fields, item, join(path, fmt.Sprint(i)), indent+4, false, comment && i == 0)
				if err != nil {
					return nil, "", err
				}
				if len(m.Content) > 0 {
					// the comment of the first key belongs before the dash
					m.HeadComment, m.Content[0].HeadComment = m.Content[0].HeadComment, ""
				}
				out.Content[i] = m
			}
			return out, "", nil
		case src.Kind == yaml.SequenceNode, null:
			return clone(src, true), "", nil
		case src.Kind == yaml.ScalarNode && f.Elem.Kind == Scalar:
			// the decoder splits a string at commas
			return clone(src, true), "", nil
		}

	case Map:
		switch {
		case src.Kind == yaml.MappingNode && f.Elem.Kind == Struct:
			out := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Style: src.Style}
			for i := 0; i+1 < len(src.Content); i += 2 {
				k, v := src.Content[i], src.Content[i+1]
				if v.Kind != yaml.MappingNode {
					r.report.Mismatched = append(r.report.Mismatched, join(path, k.Value))
					out.Content = append(out.Content, clone(k, true), clone(v, true))
					continue
				}
				m, _, err := r.mapping(f.Elem.Fields, v, join(path, k.Value), indent+2, false, comment && i == 0)
				if err != nil {
					return nil, "", err
				}
				out.Content = append(out.Content, clone(k, true), m)
			}
			return out, "", nil
		case src.Kind == yaml.MappingNode, null:
			return clone(src, true), "", nil
		}

	case Scalar:
		if src.Kind == yaml.ScalarNode {
			return clone(src, true), "", nil
		}

	case Any:
		return clone(src, true), "", nil
	}

	r.report.Mismatched = append(r.report.Mismatched, path)
	return clone(src, true), "", nil
}

// absent returns the value of the setting f at path when the file omits it -
// its default - or nil where there is nothing to write, and the comment of
// what it holds no value for, or error
func (r *renderer) absent(f *Setting, path string, indent int) (*yaml.Node, string, error) {
	if f.Kind == Struct {
		if isNil(f.Default) {
			return nil, "", nil
		}
		return r.mapping(f.Fields, nil, path, indent+2, true, true)
	}

	node, err := encode(f, f.Default)
	return node, "", err
}

// comment returns the comment of f at indent, or nothing
func (r *renderer) comment(f *Setting, indent int) string {
	var parts []string
	if text := r.schema.Describe(f, max(lineWidth-indent-2, minCommentWidth)); text != "" {
		parts = append(parts, text)
	}
	if f.Example != "" {
		parts = append(parts, strings.TrimRight(f.Example, "\n"))
	}

	return commentLines(strings.Join(parts, "\n\n"), true)
}

// joinComments returns the comments which are not empty as one, an empty
// comment line between two of them
func joinComments(comments ...string) string {
	return strings.Join(slices.DeleteFunc(comments, func(c string) bool { return c == "" }), "\n#\n")
}

// commentLines returns text as yaml comment lines, an empty one marked too
// where marked is true - an empty line would otherwise end the comment
func commentLines(text string, marked bool) string {
	if text == "" {
		return ""
	}

	lines := strings.Split(text, "\n")
	for i, line := range lines {
		switch {
		case line != "":
			lines[i] = "# " + line
		case marked:
			lines[i] = "#"
		}
	}

	return strings.Join(lines, "\n")
}

// NewElementNode returns a fresh element of the list or map s as a node, the
// defaults of its struct tags written out - see [Setting.NewElement] - or
// error.
func (s *Setting) NewElementNode() (*yaml.Node, error) {
	v, err := s.NewElement()
	if err != nil {
		return nil, err
	}

	node, err := encode(s.Elem, v)
	if err != nil {
		return nil, err
	}
	if node == nil {
		node = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}
	}
	if node.Kind == yaml.MappingNode && len(node.Content) > 0 {
		node.Style &^= yaml.FlowStyle
	}

	return node, nil
}

// textMarshaler is the interface a value written as its text implements
var textMarshaler = reflect.TypeFor[encoding.TextMarshaler]()

// encode returns the value v of the setting f as a node, or nil where there is
// nothing to write, or error
func encode(f *Setting, v reflect.Value) (*yaml.Node, error) {
	if isNil(v) {
		if f.Kind == List {
			return &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Style: yaml.FlowStyle}, nil
		}
		if f.Kind == Map {
			return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Style: yaml.FlowStyle}, nil
		}
		return nil, nil
	}

	switch f.Kind {
	case Struct:
		v = reflect.Indirect(v)
		out := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		for _, c := range f.Fields {
			if c.Hidden {
				continue
			}
			cv, err := v.FieldByIndexErr(c.index)
			if err != nil {
				// a nil squashed pointer on the way
				continue
			}
			node, err := encode(c, cv)
			if err != nil {
				return nil, err
			}
			if node != nil {
				out.Content = append(out.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: c.Key}, node)
			}
		}
		if len(out.Content) == 0 {
			out.Style = yaml.FlowStyle
		}
		return out, nil

	case List:
		v = reflect.Indirect(v)
		out := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		if f.Elem.Kind == Scalar || v.Len() == 0 {
			out.Style = yaml.FlowStyle
		}
		for i := range v.Len() {
			node, err := encode(f.Elem, v.Index(i))
			if err != nil {
				return nil, err
			}
			if node == nil {
				node = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}
			}
			out.Content = append(out.Content, node)
		}
		return out, nil

	case Map:
		v = reflect.Indirect(v)
		out := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		if v.Len() == 0 {
			out.Style = yaml.FlowStyle
		}
		keys := v.MapKeys()
		slices.SortFunc(keys, func(a, b reflect.Value) int {
			return strings.Compare(a.String(), b.String())
		})
		for _, k := range keys {
			node, err := encode(f.Elem, v.MapIndex(k))
			if err != nil {
				return nil, err
			}
			if node == nil {
				node = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}
			}
			out.Content = append(out.Content, StringNode(k.String()), node)
		}
		return out, nil
	}

	if text, ok := integerDefault(f, v); ok {
		// the notation of the author, which keeps 0644 a file mode
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: text}, nil
	}

	return encodeScalar(v)
}

// integerDefault returns the default tag of f if f is an integer and the tag
// reads as v
func integerDefault(f *Setting, v reflect.Value) (string, bool) {
	text, ok := f.Tag.Lookup("default")
	if !ok || f.Type == reflect.TypeFor[time.Duration]() {
		return "", false
	}

	v = reflect.Indirect(v)
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(text, 0, 64)
		return text, err == nil && n == v.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(text, 0, 64)
		return text, err == nil && n == v.Uint()
	}

	return "", false
}

// encodeScalar returns the single value v as a node, or nil where there is
// nothing to write, or error
func encodeScalar(v reflect.Value) (*yaml.Node, error) {
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil, nil
		}
		v = v.Elem()
	}

	if v.Type() == reflect.TypeFor[time.Duration]() {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: FormatDuration(time.Duration(v.Int()))}, nil
	}

	var marshaler encoding.TextMarshaler
	switch {
	case v.Type().Implements(textMarshaler):
		marshaler, _ = v.Interface().(encoding.TextMarshaler)
	case v.CanAddr() && v.Addr().Type().Implements(textMarshaler):
		marshaler, _ = v.Addr().Interface().(encoding.TextMarshaler)
	case reflect.PointerTo(v.Type()).Implements(textMarshaler):
		p := reflect.New(v.Type())
		p.Elem().Set(v)
		marshaler, _ = p.Interface().(encoding.TextMarshaler)
	}
	if marshaler != nil {
		if v.IsZero() {
			// whatever the zero value is, it is what an omitted key decodes to
			return nil, nil
		}
		text, err := marshaler.MarshalText()
		if err != nil {
			return nil, fmt.Errorf("encoding %s: %w", v.Type(), err)
		}
		return StringNode(string(text)), nil
	}

	switch v.Kind() {
	case reflect.String:
		return StringNode(v.String()), nil
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return StringNode(string(v.Bytes())), nil
		}
	case reflect.Struct:
		// a leaf the decoder builds by a hook of its own, which has no text
		// to write back
		return nil, nil
	}

	node := &yaml.Node{}
	if err := node.Encode(v.Interface()); err != nil {
		return nil, fmt.Errorf("encoding %s: %w", v.Type(), err)
	}

	return node, nil
}

// FormatDuration returns d as the shortest text a duration decoder reads back:
// "1h" rather than "1h0m0s".
func FormatDuration(d time.Duration) string {
	text := d.String()
	if strings.HasSuffix(text, "m0s") {
		text = strings.TrimSuffix(text, "0s")
	}
	if strings.HasSuffix(text, "h0m") {
		text = strings.TrimSuffix(text, "0m")
	}

	return text
}

// isNil reports whether v holds nothing: no value, or a nil pointer, interface,
// map or slice
func isNil(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice:
		return v.IsNil()
	}

	return false
}
