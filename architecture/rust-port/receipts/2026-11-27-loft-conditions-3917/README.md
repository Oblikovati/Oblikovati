# Edit a circular Loft surface's end condition

On the Conditions tab for LoftSrf1, Sketch1 starts with the Free condition
and zero weight. Selecting Angle sets a 90° takeoff and a unitless weight
of 1. The native runs set the weight to 2, accept it, and reopen Sketch1
with weight 2.000 ul.

Changing the angle to 45° reshapes the surface. The viewport checks the
increased radius and shaded preview. After acceptance and package reopen,
the surface has the same browser and body identity, the new shape, and
the saved 45.00° and 2.000 ul values. A separate released-kernel probe
checks the analytic surface areas and both generated boundary trims.

| Scenario | Wayland | X11 |
| --- | --- | --- |
| Published 09, change and retain weight | [7 passed](wayland/current-loft-09.xml) | [7 passed](x11/current-loft-09.xml) |
| Published 11, change and retain angle | [7 passed](wayland/current-loft-11.xml) | [7 passed](x11/current-loft-11.xml) |
| Pinned 09 | [6 passed](wayland/pinned-loft-09.json) | [6 passed](x11/pinned-loft-09.json) |
| Pinned 11 | [5 passed](wayland/pinned-loft-11.json) | [5 passed](x11/pinned-loft-11.json) |
