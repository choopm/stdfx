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
	"errors"
	"fmt"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

var (
	// ErrNotFound is returned for a path leading nowhere
	ErrNotFound = errors.New("no such key")

	// ErrNotMapping is returned for a document whose top level is not a
	// mapping of keys
	ErrNotMapping = errors.New("config is not a mapping")
)

// Document is a config file as a tree of yaml nodes. It is what an editor
// changes before [Schema.RenderDocument] writes it.
//
// Every alias of the file is replaced by a copy of what it names and every
// merge key by the keys it merges, so that each value has exactly one place.
type Document struct {
	// root is the mapping at the top of the file
	root *yaml.Node
}

// Parse returns the document of the yaml file b, or error. An empty b is an
// empty document.
func Parse(b []byte) (*Document, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	root := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if len(doc.Content) > 0 {
		root = doc.Content[0]
	}
	if root.Kind == yaml.ScalarNode && root.Tag == "!!null" {
		root = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	}
	if root.Kind != yaml.MappingNode {
		return nil, ErrNotMapping
	}

	resolved, err := resolve(root, 0)
	if err != nil {
		return nil, err
	}

	return &Document{root: resolved}, nil
}

// Root returns the mapping at the top of d.
func (d *Document) Root() *yaml.Node {
	return d.root
}

// Clone returns a copy of d sharing no node with it.
func (d *Document) Clone() *Document {
	return &Document{root: clone(d.root, true)}
}

// Get returns the node at path or nil. A key is matched regardless of case, as
// the decoder matches it, and a segment below a sequence is an index.
func (d *Document) Get(path ...string) *yaml.Node {
	node := d.root
	for _, key := range path {
		_, value := child(node, key)
		if value == nil {
			return nil
		}
		node = value
	}

	return node
}

// Set places value at path, creating the mappings on the way, or returns
// error. An index one past the end of a sequence appends to it.
func (d *Document) Set(value *yaml.Node, path ...string) error {
	if len(path) == 0 {
		return fmt.Errorf("set: %w", ErrNotFound)
	}

	node := d.root
	for i, key := range path {
		last := i == len(path)-1

		// a null or empty value on the way becomes the mapping it leads into
		if node.Kind == yaml.ScalarNode && (node.Tag == "!!null" || node.Value == "") {
			*node = yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		}

		switch node.Kind {
		case yaml.MappingNode:
			k, v := child(node, key)
			if k == nil {
				v = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
				if last {
					v = value
				}
				node.Content = append(node.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, v)
			} else if last {
				// keep the key and its comments, replace the value
				idx := index(node, k)
				node.Content[idx+1] = value
				v = value
			}
			node = v

		case yaml.SequenceNode:
			n, err := strconv.Atoi(key)
			if err != nil || n < 0 || n > len(node.Content) {
				return fmt.Errorf("set %s: %w", strings.Join(path[:i+1], "."), ErrNotFound)
			}
			if n == len(node.Content) {
				node.Content = append(node.Content, &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"})
			}
			if last {
				node.Content[n] = value
			}
			node.Style &^= yaml.FlowStyle
			node = node.Content[n]

		default:
			return fmt.Errorf("set %s: %w", strings.Join(path[:i+1], "."), ErrNotFound)
		}
	}

	return nil
}

// Delete removes the key or element at path and reports whether there was
// one.
func (d *Document) Delete(path ...string) bool {
	if len(path) == 0 {
		return false
	}

	parent := d.Get(path[:len(path)-1]...)
	if parent == nil {
		return false
	}
	key := path[len(path)-1]

	switch parent.Kind {
	case yaml.MappingNode:
		k, _ := child(parent, key)
		if k == nil {
			return false
		}
		i := index(parent, k)
		parent.Content = append(parent.Content[:i], parent.Content[i+2:]...)
		return true

	case yaml.SequenceNode:
		n, err := strconv.Atoi(key)
		if err != nil || n < 0 || n >= len(parent.Content) {
			return false
		}
		parent.Content = append(parent.Content[:n], parent.Content[n+1:]...)
		return true
	}

	return false
}

// StringNode returns a node holding the string value, quoted where yaml would
// read it as something other than a string.
func StringNode(value string) *yaml.Node {
	node := &yaml.Node{}
	if err := node.Encode(value); err != nil {
		// a string always encodes
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value, Style: yaml.SingleQuotedStyle}
	}
	if value == "" {
		node.Style = yaml.SingleQuotedStyle
	}

	return node
}

// ParseValue returns the node of the yaml text value, as it would be written
// after the key of a setting, or error.
func ParseValue(value string) (*yaml.Node, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(value), &doc); err != nil {
		return nil, err
	}
	if len(doc.Content) == 0 {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: ""}, nil
	}

	return resolve(doc.Content[0], 0)
}

// Text returns node as yaml text, as it would be written after a key.
func Text(node *yaml.Node) string {
	var b bytes.Buffer
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(clone(node, true)); err != nil {
		return node.Value
	}
	_ = enc.Close() // an encoder writing into memory fails on nothing but Encode

	return strings.TrimSuffix(b.String(), "\n")
}

// child returns the key and the value of key in mapping, or the element at
// index key of a sequence, or nils
func child(node *yaml.Node, key string) (*yaml.Node, *yaml.Node) {
	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			if strings.EqualFold(node.Content[i].Value, key) {
				return node.Content[i], node.Content[i+1]
			}
		}

	case yaml.SequenceNode:
		n, err := strconv.Atoi(key)
		if err != nil || n < 0 || n >= len(node.Content) {
			return nil, nil
		}
		return nil, node.Content[n]
	}

	return nil, nil
}

// index returns the position of the key k in mapping
func index(mapping, k *yaml.Node) int {
	for i := 0; i < len(mapping.Content); i += 2 {
		if mapping.Content[i] == k {
			return i
		}
	}

	return -1
}

// maxAliasDepth bounds how deep aliases may nest, which keeps a file built to
// expand exponentially from exhausting memory
const maxAliasDepth = 32

// resolve returns a copy of node with every alias replaced by a copy of what
// it names and every merge key by the keys it merges, or error
func resolve(node *yaml.Node, depth int) (*yaml.Node, error) {
	if depth > maxAliasDepth {
		return nil, errors.New("parse config: aliases nest too deep")
	}

	if node.Kind == yaml.AliasNode {
		if node.Alias == nil {
			return nil, errors.New("parse config: alias without anchor")
		}
		return resolve(node.Alias, depth+1)
	}

	out := *node
	out.Anchor = ""
	out.Content = nil
	if node.Kind != yaml.MappingNode {
		for _, c := range node.Content {
			r, err := resolve(c, depth)
			if err != nil {
				return nil, err
			}
			out.Content = append(out.Content, r)
		}
		return &out, nil
	}

	// keys of the mapping itself win over merged ones, and an earlier merged
	// mapping over a later one
	var merged []*yaml.Node
	for i := 0; i+1 < len(node.Content); i += 2 {
		k, v := node.Content[i], node.Content[i+1]
		if k.Tag == "!!merge" || (k.Value == "<<" && k.Style == 0) {
			sources := []*yaml.Node{v}
			if v.Kind == yaml.SequenceNode {
				sources = v.Content
			}
			for _, source := range sources {
				r, err := resolve(source, depth)
				if err != nil {
					return nil, err
				}
				if r.Kind != yaml.MappingNode {
					return nil, errors.New("parse config: merging a value which is not a mapping")
				}
				merged = append(merged, r.Content...)
			}
			continue
		}

		rk, err := resolve(k, depth)
		if err != nil {
			return nil, err
		}
		rv, err := resolve(v, depth)
		if err != nil {
			return nil, err
		}
		out.Content = append(out.Content, rk, rv)
	}
	for i := 0; i+1 < len(merged); i += 2 {
		if k, _ := child(&out, merged[i].Value); k == nil {
			out.Content = append(out.Content, merged[i], merged[i+1])
		}
	}

	return &out, nil
}

// CloneNode returns a deep copy of node.
func CloneNode(node *yaml.Node) *yaml.Node {
	return clone(node, false)
}

// clone returns a deep copy of node, without any comment if bare is true
func clone(node *yaml.Node, bare bool) *yaml.Node {
	if node == nil {
		return nil
	}

	out := *node
	if bare {
		out.HeadComment, out.LineComment, out.FootComment = "", "", ""
	}
	out.Content = make([]*yaml.Node, 0, len(node.Content))
	for _, c := range node.Content {
		out.Content = append(out.Content, clone(c, bare))
	}
	if len(out.Content) == 0 {
		out.Content = nil
	}

	return &out
}
