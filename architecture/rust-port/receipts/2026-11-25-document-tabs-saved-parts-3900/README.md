# Saved PartOne and PartTwo document tabs

Nine current scenarios passed all 72 steps. They used disposable copies
of the published Parts for keyboard switching, the tab context menu,
hiding and restoring the strip, inactive Properties, and closing from
both the close mark and the tab menu. The close checks compare the saved
file bytes. Closing a newly created empty Part restored PartOne's browser
and viewport without a save prompt. The extended window-list case added
blank Parts after opening the two published files so the list overflowed
and could be scrolled.

The runs used base revision `065ff90fe1b916d8d8bed5fdb5bade4155d50ae8`, the saved-fixture bindings,
and existing local GUI filesystem, opening and saving edits. The Linux
desktop used X11 with Xvfb, Openbox and Vulkan Lavapipe. Keyboard tab
switches use X11 input. The shared files and the isolated inputs matched
their [hashes](current-inputs.sha256). Strict build and runtime provenance
remains open under #3978.
