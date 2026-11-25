# Confirm a Part in a populated drawing layout

All four published scenarios passed with 65 steps on each of Wayland
and X11. Both confirmation cases opened a new drawing after the warning
was acknowledged. The proposed Part's Front view retained its previous
40 mm width after the working Part was edited to 60 mm. The alternate
case selected a different open file by path and showed its view and
parts list.

A separate 10-step Wayland run confirmed that a current Part creates
the drawing directly. The 38 document-creation GUI unit tests passed.
[Current-Part scenario](current-part.feature) · [Current-Part report](current-part-cucumber.json) · [GUI tests](gui-tests.txt)

The runs used `0d533413f` with the implementation and bindings in this
commit plus existing local filesystem and sketch-marking test edits.
Wayland used headless Weston and frame-loop input. X11 used Xvfb and
Openbox. Both used Vulkan Lavapipe. The published source and fixture
hashes matched before and after the runs. Strict build and runtime
provenance remains open under #3978.
[Wayland report](wayland-cucumber.json) · [X11 report](x11-cucumber.json) · [Input hashes](inputs.sha256) · [Alternate drawing](alternate-layout.png)
