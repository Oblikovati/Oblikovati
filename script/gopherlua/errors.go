// SPDX-License-Identifier: GPL-2.0-only

package gopherlua

import "fmt"

// errNoCaller reports that oblikovati.call was invoked but no host CallFunc was wired
// into Globals — a host misconfiguration, named with the offending method.
func errNoCaller(method string) error {
	return fmt.Errorf("oblikovati.call(%q): no host caller configured", method)
}
