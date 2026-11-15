<!-- SPDX-License-Identifier: GPL-2.0-only -->

# OS-input Home preview check

On 2026-11-15, XTest sent mouse events to the Rust desktop on Linux Xvfb
with Mesa software Vulkan. The application source was
`f12341af1b1c0116eebd3903907091c2e3df4664`; a status document and diagnostic
receipt files had local changes. The binary SHA-256 was
`56aa12a0e3ed7b2c296d1dcd2d99fab6392236d3073b0fcb1f1800c0ab2c879f`.
The MCPBridge library SHA-256 was
`2942bdba7a756d20ea17f24fb18e9eb41f495e20b1d259caf9d28d77e237b20b`.

The test created a part, selected Home through its tab while leaving the
pointer on that tab, returned to the part, and hovered Home. MCPBridge captured
the resulting Home thumbnail and the state after moving the pointer away. The
captures show Home workspace content in the preview and its dismissal while
the part remains active. Both captures were inspected. The application window
was resized to 1900 × 1000 on a 1920 × 1080 display; input coordinates are
specific to this run. The application and Xvfb server were closed afterward.

This is a manual diagnostic, not BDD acceptance evidence. Build provenance
remains open (#3978). The test did not verify the complete recent-documents
content or cross-platform behavior.
