# Home Recent file properties, metadata, and view state

Five current scenarios passed with the published profile. The notice
preceded Part23's file properties in a fresh test process. Its General
page showed the actual copied file's name, location, and byte size.
The tests also verified mixed Part and Assembly selection, sorting
by file size, keeping a search and type filter across List and Grid,
and resetting the Location column.

The native runs used base `fa01d3b4e51cb614c42ead0af3b306ee1ad8af5f`, the current bindings, and
existing local GUI filesystem, opening and saving edits. The desktop
used Linux X11 with Xvfb, Openbox and Vulkan Lavapipe. The published
[input hashes](current-inputs.sha256) matched the isolated copies.
Strict build and runtime provenance remains open under #3978.
