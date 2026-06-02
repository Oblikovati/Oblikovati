//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package native

import "runtime"

// GLFW and the underlying window system require that init, window creation, and event
// polling all happen on the main OS thread — on macOS/Cocoa, calling them off the main
// thread aborts the process. Package init() runs on the startup (main) thread before
// main(), so locking here pins the main goroutine to that thread for the whole process
// and every head binary that imports this package (headsmoke, oblikovati-head) inherits
// the guarantee without repeating it.
func init() { runtime.LockOSThread() }
