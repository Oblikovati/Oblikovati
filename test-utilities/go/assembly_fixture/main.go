// SPDX-License-Identifier: GPL-2.0-only
// Generate a real Go assembly package for the Rust compatibility test.
// Run from ../Oblikovati: go run ../oblikovati-rust/test-utilities/go/assembly_fixture/main.go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/contentset"
	"oblikovati.org/model/doc"
	"oblikovati.org/persistence"
)

func main() {
	dir, err := os.MkdirTemp("", "obk-assembly-fixture-")
	must(err)
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "assembly.oad")
	writeAssembly(path)
	contents, err := os.ReadFile(path)
	must(err)
	fmt.Print(string(contents))
}

func writeAssembly(path string) {
	workspace := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	document, err := compdef.AddAssembly(workspace, path, true)
	must(err)
	assembly := document.Content().(*compdef.AssemblyComponentDefinition)
	part := compdef.NewPartComponentDefinition()
	first := assembly.Occurrences().AddByComponentName("widget:1", part, "parts/widget.opd", math.Identity4())
	first.SetGrounded(true)
	second := assembly.Occurrences().AddByComponentName("widget:2", part, "parts/widget.opd", math.Translation4(math.V3(3, 0, 0)))
	second.SetSuppressed(true)
	must(workspace.Save(document))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
