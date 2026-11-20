# Local navigation wheel scenario snapshot

This bundle captures the navigation wheel feature, its supporting documents and
SVG files, and the native `.opd` fixture from the live scenario directory. The
local Git commit is `94a3cf8d6a10d7502b7cc276973488bf3bff1b83`. It is not an
upstream commit. The bundle SHA-256 is
`20e3002bc9d39b44dcdc3abac93ed8574566517be3f1d2060d2f154b0ea9cdcc`.

The snapshot tool checked the selected file hashes before and after the copy.
The snapshot contains 12 scenarios. A checkout restored from the bundle passed
revision and file-hash verification. The previous full-feature run used the
live directory; its feature, support files, and fixture hashes match the new
snapshot. It did not run from this snapshot.

This snapshot can supply `--local-snapshot` to the experimental build and run
chain. It does not replace the global pinned source or establish build and
runtime provenance. Results remain diagnostic until those inputs are bounded
and verified under #3978.
