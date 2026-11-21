# Windows desktop

The desktop owner, GUI, viewport scene and Martensite rendering code run on
Windows and Linux. The Win32 backend owns the window, converts native messages
to viewport events, wakes the owner thread for worker completions and exposes
the Unicode system clipboard. The add-in host loads trusted DLLs through the
published C ABI.

The Martensite Vulkan revision pinned by this branch exposes offscreen
rendering on Windows. Its direct Win32 surface factory is a stub. The
application renders to a virtual target, reads the completed RGBA frame and
paints it as a Win32 DIB. This incurs a CPU copy on each presented frame.
Linux continues to use its native Vulkan surface.

On Windows x64, an interactive smoke run opened the saved extrusion fixture
`architecture/rust-port/receipts/2026-11-15-live-extrusion-edit/cube.opd`
with Vulkan validation enabled. F6 changed the front view to a fitted
isometric view and Alt+F4 closed the clean document with exit code 0. The Lua worker runs with a Win32 job that limits process memory, CPU time,
and the number of processes. The shared worker protocol and VM quotas
continue to enforce script-level limits. Express caches on Windows still
require a verified running executable handle as described in
[assembly producer](assembly-cache-producer.md).
