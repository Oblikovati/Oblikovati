# Hole face oracle

`occt-7.6.3.csv` is the output of `occt.cpp` from Ubuntu's OCCT 7.6.3
packages. The program builds a unit cube and cuts a cylinder of radius 0.1
from its upper face. It records the surface type, analytic area, and wire
count of each face for through and blind cuts.

The Rust replay test compares every result face with these values. It also
checks volume, source names, undo, redo, parameter changes, and native YAML
reopen separately. The CSV is a test oracle, not a runtime dependency.

To regenerate it, compile `occt.cpp` with the OCCT foundation, modeling-data,
and modeling-algorithms development libraries, then save standard output to
`occt-7.6.3.csv`. The program writes the OCCT version to standard error.
