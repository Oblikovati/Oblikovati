# Inventor spatial sketch input

The persistence reader uses Corliss 0.1.0 to decode recognized 3D point and
line records in an Inventor part. It returns the sketch name, segment-local
node identities, coordinates in centimetres, and each line's point references.
It refuses incomplete nodes, malformed coordinates, and endpoints that do not
belong to the same sketch.

The reader does not claim to decode every spatial entity type. It does not
create a document, display geometry, edit an Inventor file, or save one. The
source bytes remain unchanged. Native document opening, solid conversion,
3D-sketch ownership, selection, and save/reopen remain part of #4092.

The regression fixture is the generated Inventor part supplied for viewport
entity selection. Tests check its two populated sketches, five points, two
lines, source names, coordinates, endpoint references, and malformed inputs.
