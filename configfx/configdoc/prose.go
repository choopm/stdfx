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
	"go/doc/comment"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Describe returns what the doc comment of setting says, written for the
// operator of a config file and wrapped at width: the name of the field it
// starts with is replaced by its key, and a doc link to another field reads as
// the key of that field. A setting without a doc comment is described by the
// doc comment of its type.
func (s *Schema) Describe(setting *Setting, width int) string {
	text, ident := setting.Doc.Text, setting.GoName
	if strings.TrimSpace(text) == "" {
		text, ident = setting.TypeDoc, deref(setting.Type).Name()
	}
	if strings.TrimSpace(text) == "" {
		return ""
	}
	if setting.Key != ElemKey {
		text = replaceIdent(text, ident, setting.Key)
	}

	parser := &comment.Parser{
		// every [Name] and [Type.Field] is a link, which is then either
		// resolved to a key or read as the plain name it is
		LookupSym: func(recv, name string) bool { return true },
	}
	doc := parser.Parse(text)
	for _, block := range doc.Content {
		s.resolveLinks(block, setting.declaring.Package)
	}

	printer := &comment.Printer{
		TextCodePrefix: "  ",
		TextWidth:      width,
	}

	return strings.TrimRight(string(printer.Text(doc)), "\n")
}

// resolveLinks replaces the doc links of block by plain text, declared in the
// package pkg
func (s *Schema) resolveLinks(block comment.Block, pkg string) {
	switch b := block.(type) {
	case *comment.Heading:
		b.Text = s.plainLinks(b.Text, pkg)
	case *comment.List:
		for _, item := range b.Items {
			for _, content := range item.Content {
				s.resolveLinks(content, pkg)
			}
		}
	case *comment.Paragraph:
		b.Text = s.plainLinks(b.Text, pkg)
	}
}

// plainLinks returns text with its doc links replaced by what they name: the
// key of a field, else the name
func (s *Schema) plainLinks(text []comment.Text, pkg string) []comment.Text {
	for i, t := range text {
		link, ok := t.(*comment.DocLink)
		if !ok {
			continue
		}

		name := link.Name
		if link.Recv != "" {
			name = link.Recv + "." + link.Name
			if key, ok := s.keys[Key{Field: link.Name, Package: pkg, Type: link.Recv}]; ok && link.ImportPath == "" {
				name = key
			}
		}
		text[i] = comment.Plain(name)
	}

	return text
}

// replaceIdent returns text with its leading word replaced by key if that word
// is ident, which is how a doc comment starts by the convention of Go
func replaceIdent(text, ident, key string) string {
	rest, ok := strings.CutPrefix(text, ident)
	if !ok || ident == "" {
		return text
	}
	if r, _ := utf8.DecodeRuneInString(rest); rest != "" && (unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_') {
		// a longer word merely starting with ident
		return text
	}

	return key + rest
}
