# Scene depth after a model edit

Base `44725c062`, Martensite `bef25f47548431c8f128f08f66a1f719bcee4982`.
Linux, Xvfb and Lavapipe. The temporary viewport diagnostic is retained in
`depth-probe.rs.txt`. It submitted a 10 mm prism, fitted Home, then submitted a
20 mm prism without changing the camera. The edit kept near/far at
16.6321116/23.3678884 and reproduced the triangular cutout. An explicit depth
fit changed near to 16.0547613, kept eye, target, up and scale unchanged, and
removed the cutout. Reversing triangle order did not change pixels. The two
`obk-depth` images show this comparison.

The head-level regression failed before the fix and passed after it. Six
native tests cover positive and negative extrusion edits, resize, empty scene,
projection stability, camera generation, Previous View and marked-view restore.
Twelve unit tests cover near and far clipping, perspective and rotated axes, empty geometry,
a camera inside geometry, invalid ranges, a failed depth operation and a fit
that still leaves geometry clipped. A cached axis-aligned box is built once per
scene submission. It is checked on discrete restore; exact vertex scans occur
only if that box may clip. Pointer motion and ordinary navigation do not scan
the scene. The fit failure uses a named fake; it does
not inject a native renderer failure. Existing native edit/undo and face-pick
tests passed before the final equivalent helper refactor.

The native X11 edit scenario passed 10 steps. Its source was the live,
unversioned feature with SHA-256
`34abe4989ff7020bb1a3a7a5207f551b96535cfbbb33c8212e1da77ecfa503aa`
before and after the run. This is a diagnostic, not verified BDD acceptance.
`bdd-before.png` and `bdd-after.png` show the cutout before and after the fix.
The large dark face remains a separate lighting observation. No 60 fps claim
is made. Test windows closed after their runs.
