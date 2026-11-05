//go:build ignore

// SPDX-License-Identifier: GPL-2.0-only
// Read-only palette oracle from Go aa8297d1.
package main

import (
    "encoding/json"
    "os"
    "oblikovati.org/model/colorscheme"
)
func main() {
    registry := colorscheme.NewRegistry()
    if err := json.NewEncoder(os.Stdout).Encode(registry.Schemes()); err != nil {
        panic(err)
    }
}
