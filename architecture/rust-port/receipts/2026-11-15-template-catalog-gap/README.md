# Template catalog fixture gap

Two pinned scenarios were checked in diagnostic mode against the chooser at
`43dc00202`: adjacent Part template navigation and clicking a folder after
keyboard highlight. Both are blocked by the missing named template catalog.
The reports include later diagnostic steps, which are tainted and are not
acceptance evidence.

An earlier apparent pass for the folder-click scenario substituted the built-in
standard metric Part for the named `Standard (mm)` asset. That substitution is
not supported by the captured fixture, so its acceptance claim is withdrawn.
The scenario owner's `document-creation/BLOCKERS.md` requests the needed assets
and configuration. Host catalog work is tracked in
[issue #3952](https://github.com/Oblikovati/Oblikovati/issues/3952).
