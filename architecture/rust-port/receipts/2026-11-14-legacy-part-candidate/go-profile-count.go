// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"fmt"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/contentset"
	"oblikovati.org/model/doc"
	"oblikovati.org/persistence"
	"os"
)

func main() {
	d, e := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default()).Open(os.Args[1], true)
	if e != nil {
		panic(e)
	}
	s := d.Content().(*compdef.PartComponentDefinition).Sketches().Item(0)
	n := s.Profiles().Count()
	if n != 1 {
		panic(fmt.Sprintf("profiles %d: expected 1", n))
	}
	fmt.Println("Go profiles:", n)
}
