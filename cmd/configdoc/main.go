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

// Command configdoc writes the doc comments of a config struct, and of every
// type it reaches, into a Go file declaring a function which returns them as
// configdoc.Docs. It is meant for go:generate:
//
//	//go:generate go tool configdoc -type example.com/app/server.Config -package docs -out docs_gen.go
//
// With -missing it lists the configured fields without a doc comment instead.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/choopm/stdfx/configfx/configdoc/extract"
)

func main() {
	opts := extract.Options{}
	flag.StringVar(&opts.Type, "type", "", "config struct as `<import path>.<name>`")
	flag.StringVar(&opts.Package, "package", "", "package `name` of the generated file")
	flag.StringVar(&opts.Func, "func", "generated", "`name` of the generated function")
	flag.BoolVar(&opts.SkipUntagged, "skip-untagged", false, "leave out exported fields without a mapstructure tag")
	out := flag.String("out", "", "`file` to write, standard output if empty")
	missing := flag.Bool("missing", false, "list the configured fields without a doc comment and exit")
	flag.Parse()

	if err := run(opts, *out, *missing); err != nil {
		fmt.Fprintln(os.Stderr, "configdoc:", err)
		os.Exit(1)
	}
}

// run extracts as opts says and writes the result to out, or lists the
// fields missing a doc comment, or returns error
func run(opts extract.Options, out string, missing bool) error {
	if opts.Type == "" || (opts.Package == "" && !missing) {
		return fmt.Errorf("-type and -package are required")
	}

	result, err := extract.Extract(opts)
	if err != nil {
		return err
	}

	if missing {
		for _, field := range result.Missing {
			fmt.Println(field)
		}
		return nil
	}

	src, err := extract.Generate(opts, result.Docs)
	if err != nil {
		return err
	}
	if out == "" {
		_, err := os.Stdout.Write(src)
		return err
	}

	return os.WriteFile(out, src, 0o644)
}
