# Ellipse baseline against the pinned source

The nine new baseline records use scenario commit
`b576445b98cffd686dcc05aa8c017de2a591c8c8` and its Ellipse feature hash
`af53af88f7a32b441d76154fd75aeb36e68d4e673c47d21ecefa5be14cef5a6f`.

Two focused runs passed on application commit `04a58abd4`:

- `ellipse-15` retained the placed minor dimension and its original expression
  after Cancel.
- `ellipse-16` accepted the 4 mm minor dimension and retained the 8 mm major
  dimension.

The runner created the starting geometry through the sketch commands. The
pinned source remained clean. These two rows have implementation evidence.
The other seven new baseline rows describe the pinned starting geometry or
prior dialog state that their scenarios require. The published source supplies
those prerequisites through native Part fixtures and scenario steps.

The ledger keeps [the existing rotated Measure defect][measure] open.

[measure]: https://github.com/Oblikovati/Martensite/issues/406
