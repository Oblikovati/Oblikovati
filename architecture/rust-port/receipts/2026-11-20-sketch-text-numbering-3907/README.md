# Sketch Text numbering and symbols

Pinned `text-22` passed all seven steps in the native Format Text window.
The X11 test selected the draft, checked the five numbering choices, chose
Numeric, pressed End, and inserted the diameter sign from the Symbols list.
It checked the editable prefix and character, the selected font's curved
diameter outline, the canvas pixel change, and the absence of committed text.

- [Pinned Cucumber report](pinned-text-22/cucumber.json)
- [Editor and canvas capture](numbered-diameter-preview.png)
- [Enlarged canvas detail](canvas-detail.png)

Numbering adds editable markers to the selected paragraphs. The test fixture
uses DejaVu Sans because the default B-rep font can produce a square for U+2300
even when another installed font supplies it. See
[Martensite #575](https://github.com/Oblikovati/Martensite/issues/575).

The tests used Linux X11 under Xvfb and Vulkan Lavapipe. The focused run used
`OBK_BDD_FEATURE=text` and `OBK_BDD_SCENARIO=text-22` with snapshot
`b576445b98cffd686dcc05aa8c017de2a591c8c8`. The test binary used the Cargo
compiler arguments with temporary output in memory because the disk was busy.
Strict acceptance still requires bounded build and runtime inputs under #3978.

The unfiltered live Text run used the same binary and unchanged feature and
support files. Nine of 38 scenarios passed. The other 29 stopped at undefined
steps. [Cucumber report](cucumber.json)
