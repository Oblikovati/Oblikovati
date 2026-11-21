<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Selected-face menu and viewport selection

The published feature used the native Part and the application code in
`61967b9d8`. All three solid-face scenarios passed with X11 pointer input.
The complete run passed 29 steps and failed four at the first unavailable
3D-sketch step in each deferred scenario. The feature, scenario notes, and
native Part hashes matched before and after.

The standard and radial menus appeared as separate rendered regions.
Select Other, Select Tangencies, and Export Face As were visible and disabled.
A separate native test clicked all three without changing the selection and
then opened Face Properties. Ten existing native Part and sketch marking-menu
tests passed, as did the four Part menu unit tests. The unit tests include
custom slot preservation.

The complete run remains incomplete because the four 3D-sketch scenarios are
deferred. The source and runtime are unsealed. See the [previous run and
command](../2026-11-21-viewport-face-selection-4055/README.md) and
[#4055](https://github.com/Oblikovati/Oblikovati/issues/4055).
