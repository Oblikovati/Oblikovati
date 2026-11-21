# Viewport Pan on the native desktop

The [pinned run](pinned/cucumber.json) passed all four scenarios and 32 steps
at application commit `f058f8b8c00d7da96efbd240dc11018083a3cefb`.
The clean scenario snapshot remained at
`b576445b98cffd686dcc05aa8c017de2a591c8c8`, with feature SHA-256
`4303c8a64b9262841dba3b8b48a74e51581a6a0ef9d98798286d514c53155647`.

The [native draft](native-draft/cucumber.json) also passed four scenarios
and 32 steps on the local fixture binding. Its four Given steps reference
the existing `fixtures/cad/body-properties-start.opd`. This is the only
change to the [published feature](native-draft/published-original.sha256).
The [draft and fixture hashes](native-draft/draft-source.sha256) stayed
the same throughout the run. The published feature still names the
Inventor path, and its inventory entry stays blocked until that path
changes and the feature is rerun.

Both runs used Xvfb and Vulkan Lavapipe. Separate [X11 input tests](x11-pan.log.gz)
drove Pan drag and the context menu through the native window. The menu
test used a saved Part and checked that Done and Zoom preserved the Part
and its saved state. The [shared fixture Orbit test](x11-orbit.log.gz) also passed.
The [Pan capture](pan-context-x11.png) shows the running command and the
visible solid. The [earlier baseline](../2026-11-20-viewport-pan-r0-3881/README.md)
has the Go comparison and R0 records. Strict build and runtime provenance
remains open under [#3978](https://github.com/Oblikovati/Oblikovati/issues/3978).
