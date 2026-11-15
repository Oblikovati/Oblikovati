# Sketch ribbon availability (#3956)

The Sketch ribbon shows an Insert group. Image and Points remain disabled until
those operations exist. Project Geometry is also shown disabled. The Line
tooltip describes line placement and directs arc placement to Center Arc.

The full ribbon reserves room for Finish while other groups collapse. When
sketch edit ends, the previously selected part tab returns unless the user
selected another tab during sketch edit. Restoration is limited to the same
document.

The pinned `ribbon-command-availability` source is revision
`48987af7b5cbb06515bc6f30385f508382f34064`. The first two scenarios expect Create
commands to be enabled outside sketch edit. Their source does not describe the
result of selecting one in that state. The Go reference requires sketch edit.
The question is recorded for the scenario owner; those commands remain disabled
until their behavior can be implemented and verified.
