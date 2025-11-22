package stdfx

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"syscall"
)

// ContainerEntrypointDefaultTools are the default tools for [ContainerEntrypoint]
var ContainerEntrypointDefaultTools = []string{"sh", "/bin/sh", "bash", "/bin/bash"}

// ContainerEntrypoint might be used with [fx.Invoke] and tooling when the calling
// go program is packaged into a container where it is used as the entrypoint.
// This will execute any value of `tools` when given as the first argument and if found in $PATH.
// If tools is empty it will use a default list: [ContainerEntrypointDefaultTools].
// A special value of '*' allows for any tool.
// There is extra handling when the first argument is the binary name itself:
// For such cases that argument is silbently shifted out and execution continues.
//
// Example usage:
//   - fx.Invoke(stdfx.ContainerEntrypoint())
//   - fx.Invoke(stdfx.ContainerEntrypoint("sh", "bash", "whoami"))
//   - fx.Invoke(stdfx.ContainerEntrypoint("*"))
//
// Resulting container image invocations:
//   - docker run --rm -it ghcr.io/choopm/myproject:latest sh -c 'echo hello world'
//   - docker run --rm -it ghcr.io/choopm/myproject:latest bash -i
//   - docker run --rm -it ghcr.io/choopm/myproject:latest whoami
//   - docker run --rm -it ghcr.io/choopm/myproject:latest myproject -c ...
func ContainerEntrypoint(tools ...string) func() {
	// use default tools if nothing was provided
	if len(tools) == 0 {
		tools = ContainerEntrypointDefaultTools
	}

	// return constructor
	return func() {
		if len(os.Args) < 2 {
			// only care when atleast one argument was given to cli
			return
		}

		wildcardTool := slices.Contains(tools, "*")

		// container image argument handling
		switch {
		case os.Args[1] == filepath.Base(os.Args[0]):
			// First argument is the same as binary name -> remove it, continue
			os.Args = append(os.Args[0:0], os.Args[1:]...)

		case wildcardTool || slices.Contains(tools, os.Args[1]):
			// Chain to the first argument given by looking it up in $PATH.
			path, err := exec.LookPath(os.Args[1])
			if err != nil && wildcardTool {
				// wildcard tool is allowed, so the failing lookup might be
				// caused by first argument not being any tool, continue
				break
			} else if err != nil {
				panic(err)
			}
			err = syscall.Exec(path, os.Args[1:], syscall.Environ())
			if err != nil {
				panic(err)
			}
		}
	}
}
