# Sketch Text bullets and numbering

Pinned `text-33` passed all eight steps in the native Format Text window.
The X11 test typed two paragraphs separately, selected them, applied Bullets,
and changed them to numeric numbering. It checked the actual editor text and
canvas outlines and confirmed that the numbered Text kept its specified
anchor and remained visible at Ready with 42 dimensions needed.

- [Pinned Cucumber report](pinned-text-33/cucumber.json)
- [Bullet preview](bulleted-paragraphs.png)
- [Numbered Text at Ready](numbered-paragraphs-at-ready.png)

The fixture uses a native sketch with 40 degrees of freedom. The committed
Text anchor adds two. The numbered and bulleted markers are editable text.

The focused Linux run used X11 under Xvfb, Vulkan Lavapipe, the pinned
snapshot `b576445b98cffd686dcc05aa8c017de2a591c8c8`, and
`OBK_BDD_FEATURE=text OBK_BDD_SCENARIO=text-33`.
Strict acceptance still requires bounded build and runtime inputs under #3978.

The unfiltered live Text run passed 10 of 38 scenarios. The other 28 stopped
at undefined steps. The feature and support hashes matched before and after
execution. [Cucumber report](cucumber.json)

Native `text-07` and `text-22` passed 17 regression steps in total.
All 10 GUI Text unit tests passed.
