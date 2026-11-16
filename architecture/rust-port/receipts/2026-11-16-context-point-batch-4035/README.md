# Context point projection diagnostic

Base `87ed4ad09`, Martensite `bef25f47548431c8f128f08f66a1f719bcee4982`.
The owner tests check a single projection batch, point distance and ID order,
empty input, an incomplete result, and an error. The red run confirms the
old scalar path does not satisfy the batch contract. The final owner run passed
10 tests. The native X11 XTEST rectangle-menu and overlapping-pick tests passed
under Xvfb with Lavapipe. The rectangle screenshot was visually inspected.

The broad `unit` run used an overbroad filter and selected native UI tests on
the desktop. One unrelated file-dialog test failed before the run was stopped;
it is not a passing suite result. The two intended native tests were run on
separate Xvfb displays. A native Select Other test also selected the line from
an overlapping point and line. Its menu screenshot was visually inspected.
Their windows closed with the displays.

This change enables the existing checked projection cache for the context
point batch. No timing comparison was made for this specific path, and these
checks do not establish 60 fps. They are diagnostic, not BDD acceptance.
