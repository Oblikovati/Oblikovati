# Template material details

The Part template chooser shows the material intent from the same recipe factory
used by `documents.create`. An unassigned recipe shows `Material: Unassigned`.
An assigned identity shows its resolved display name, with document definitions
before project definitions and the built-in catalog. An identity that cannot be
resolved shows `Material unavailable` and its identity. A catalog load failure
is retained with the material identity and shown in the chooser instead of being
presented as an unassigned material. Recipe and size inspection failures are
also retained for the chooser; they do not prevent other GUI frames. Built-in
recipe inspection is cached, while project and catalog resolution uses the
current workspace.

The built-in Part and sheet-metal recipes do not assign a material. The current
chooser does not yet load the configured `Standard (mm).ipt` fixture described
by the scenario owner. This change does not invent the `Generic` asset or its
physical properties. The scenario remains open under #3932.

Native tests cover the built-in chooser and created Part. Encoded recipe tests
cover an embedded assignment, scope precedence, an absent assignment, and an
unresolved identity. The fixture with an assigned material is not yet selectable
in the native chooser because file-backed template selection is not implemented.
