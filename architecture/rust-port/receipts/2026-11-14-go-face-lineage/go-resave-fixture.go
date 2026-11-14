// SPDX-License-Identifier: GPL-2.0-only
package main

import (
	"fmt"
	"oblikovati.org/model/contentset"
	"oblikovati.org/model/doc"
	"oblikovati.org/persistence"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		panic("expected fixture path")
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	d, err := ws.Open(os.Args[1], true)
	if err != nil {
		panic(err)
	}
	if err = ws.Save(d); err != nil {
		panic(err)
	}
	fmt.Println("Go reopened and saved the modified recipe")
}
