# Pinned Loft scenario evidence

These native runs used the clean scenario revision
`b576445b98cffd686dcc05aa8c017de2a591c8c8`. The scoped journeys ran
46 steps on each desktop. Their Given steps create an empty metric Part
or open a disposable copy of the [parallel-circle fixture](fixture.sha256).
The fixture is identical to the native Part used by the published Loft
scenarios.

| Scenario | Steps | Wayland | X11 with Openbox |
| --- | ---: | --- | --- |
| Empty Part guidance | 6 | [01](wayland/loft-01.json) | [01](x11/loft-01.json) |
| Separate point and circle | 3 | [02](wayland/loft-02.json) | [02](x11/loft-02.json) |
| Solid creation | 5 | [03](wayland/loft-03.json) | [03](x11/loft-03.json) |
| Automatic Mapping | 5 | [04](wayland/loft-04.json) | [04](x11/loft-04.json) |
| Reopen and cancel | 7 | [06](wayland/loft-06.json) | [06](x11/loft-06.json) |
| Path mode switching | 7 | [07](wayland/loft-07.json) | [07](x11/loft-07.json) |
| Open surface creation | 6 | [08](wayland/loft-08.json) | [08](x11/loft-08.json) |
| Cancel the incomplete point set | 7 | [10](wayland/loft-10.json) | [10](x11/loft-10.json) |

The [source record](source-revision.json) gives the Git revision and status
before and after the run. The unfiltered strict gate has separate coverage
requirements for the remaining pinned scenarios.
