//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

// normalDebugOn toggles the viewport's normal-debug render (Tools ▸ Normal Debug): shaded
// triangles draw front-facing GREEN and back-facing RED (gl_FrontFacing in mesh.frag), so
// inverted-winding / flipped-normal triangles — which the normal two-sided shading hides — are
// obvious. UI state, so it lives here rather than on the session.
var normalDebugOn bool
