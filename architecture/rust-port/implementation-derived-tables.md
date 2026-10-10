# Derived parameter tables — partial R2

## Source resolution and enumeration

[#3656](https://github.com/Oblikovati/Oblikovati/issues/3656) adds
`FileWorkspace::resolve_parameter_source` and the file-owner query
`parameters.derivedTables.list`. Sources resolve against owned file paths or a
unique unbound document title. Bound display titles are not path aliases.
Resolution performs no file I/O and does not activate the source. Duplicate
unbound titles produce an ambiguity error; missing sources return no owner.
Only part and assembly documents supply derived parameter sources.

The canonical API response retains table order, linked names, produced-parameter
references, and feature ownership. Candidate names come from the current source
recipe in parameter order and include numeric user parameters. Model parameters
and text/boolean parameters are excluded. Missing sources remain in the result
with a health diagnostic. A retained table that has not been synchronized reports
that state even if its source is now open; enumeration does not change the recipe.
An empty collection uses the canonical null table list, as in Go.

Typed-client tests cover ordered candidates, live source edits, provenance,
resolved/missing/unavailable link health, empty workspaces and collections,
drawing refusal, ambiguous names, and focus/history preservation. The existing
model synchronization operation supplies health updates in the tests; this is
not evidence of automatic workspace propagation.

## Remaining work

Connect table creation and link editing to source resolution, stable ID
reservation, source parameter export, and the document reference graph. Propagate
source edits, close/reopen, and undo/redo through that graph with cycle handling.
Keep failed source values distinct from valid current quantities, and convert
between document working scales at the source boundary. No live cross-document
recompute or geometry integration is claimed by the enumeration path.
