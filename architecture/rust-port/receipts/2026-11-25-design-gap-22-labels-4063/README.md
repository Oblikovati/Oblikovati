# ViewCube face label projection

The ViewCube maps each glyph along the face's projected horizontal and
vertical edges. The geometry checks cover the face-on identity and the
isometric side axes and top face. Dark uses three directional face fills.

The native desktop checks used Linux under Xvfb, Openbox, and Vulkan
Lavapipe. Five published scenarios and 48 steps passed, covering
face-on labels, quarter-turn rotation, corner selection, hybrid
projection, and orientation lock. The Quick Access selection popup
uses padded rows so the Faces entry is fully inside its clip area.
[Front capture](front-viewcube.png) · [Isometric capture](isometric-viewcube.png) ·
[Cucumber](cucumber.json) · [JUnit](junit.xml)

The exported viewport orientation scenario source has SHA-256
`4368a0cdb1767d8a80ddad9966f82709f610fdbd6c48987cc4073273cc939f62`.
[Source report](source-revision.json)
