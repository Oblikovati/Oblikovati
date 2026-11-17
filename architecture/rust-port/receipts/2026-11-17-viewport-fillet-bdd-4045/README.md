# Viewport feature interaction diagnostic (#4045)

On 2026-11-17, the live feature ran under Xvfb with Lavapipe. All five
scenarios and 35 steps passed. Native X11 input opened Fillet, picked an edge,
changed the radius from 2 mm to 3 mm, and confirmed it. The test compared
visible preview pixels after the radius change and checked the browser entry.
The other two scenarios exercised extrusion edit and cancel.

The source feature SHA-256 was
`34abe4989ff7020bb1a3a7a5207f551b96535cfbbb33c8212e1da77ecfa503aa`
before and after the run. The live source has no recorded Git revision and
retains fixture and capture tags. This is diagnostic evidence, not verified
BDD acceptance. The extrusion fixture sketch has four degrees of freedom.
The captured model has a dark face; shading is tracked by #4044.

The shared sketch selection step also passed its focused native diagnostic
(one scenario, eight steps) after the binding change. The run did not measure
frame rate and does not establish the 60 fps target.
