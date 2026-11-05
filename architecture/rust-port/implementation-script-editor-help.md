# Contract-generated script method help — partial R7

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
This ports the Go console's API documentation source and the signature/hover
lookup from `head/ui/code_editor_sighelp.go`. Native tooltip drawing and popup
visibility/input rules remain open.

## Contract source and generation

`make script-docs` uses a Rust generator over the exact API package resolved by
`cargo metadata --locked`. It requires the owner, version and commit recorded in
`release.toml`, then reads that Git checkout's canonical Go `wire` and `client`
source. No sibling API checkout, Go application source, Go compiler or runtime
source lookup supplies production help. This uses the contract source shared by
both API languages; it does not redeclare request DTOs in the application.

The parser library stays inside the owned `GoSource` boundary. It uses
[Tree-sitter's Go grammar](https://docs.rs/tree-sitter-go/0.25.0/tree_sitter_go/)
and Rust parser bindings. The grammar predates Go 1.27 generic methods. A tested
adapter masks only method declaration type-parameter lists while preserving byte
positions and line endings. Nested brackets, quoted tags and comments are skipped
correctly. Literal/comment examples that contain method declarations stay intact.
A final line terminator accounts for Go's implicit EOF terminator. Parse errors,
unknown request shapes and missing wire constants/DTOs are named declines.

Generation joins wire-name constants, top-level request fields and client
`mcp:summary` annotations. It handles package-level and receiver call helpers,
DTO parameters, local declarations, addressed literals, nil and empty-struct
requests. JSON tags supply Lua keys; omitted/embedded fields are excluded and
field order is retained. Source and method traversal have explicit sorted order.
Repeated clients for one wire name merge canonical options without duplicate
fields. Conflicting field types are refused; the first nonempty summary is kept.
The catalog is compact JSON to reduce embedded resource size, with explicit
API version/commit provenance. It is metadata, not a second wire schema.

`make gate` regenerates the expected catalog and compares it with the committed
resource. A changed API pin, source annotation or DTO field requires regeneration.
The gate fixture still checks the original workspace, documentation, workflow and
duplication commands; the help check does not enable product publication.

## Runtime editor help

`MethodHelpCatalog::published` filters the full catalog to the host's injected
available-method snapshot. A documented API method is not a promise that the
application implements it. Lookup is exact and immutable. Signatures use Lua's
`method{ field, field }` form or `method{}` for an empty request. Signature help
resolves the nearest enclosing `oblikovati` call; hover help resolves the hovered
method chain. Unknown methods, bare Lua calls and out-of-range hits produce no help.
There is no host call, model access or source edit in these queries.

## Verification

The pinned API v0.158.0 produces 628 unique methods, all with summaries. A full
comparison against the Go `luadoc.Methods` oracle finds the same 628 names and
argument field names/types, except that Rust retains fixed array dimensions that
Go's renderer flattens to slices. The Rust parser also chooses the annotation
nearest `drawingDimensions.setTextStyle`, where Go selects an earlier adjacent
Delete annotation. Both intentional differences have regression tests.

Generator tests cover stable regeneration, stale-file refusal, owner/version
checks, syntax/request/tag declines, overload consolidation, generic-method
adaptation, literal examples, field ordering and fixed arrays. Runtime tests cover
signature formatting, duplicate refusal, the complete embedded catalog, available
method filtering, enclosing calls and hover queries. The generator commands use
named command/source fixtures; no network call supplies test catalog content.
Run `cargo test -p oblikovati-build` and
`cargo test -p oblikovati-script --test help`, then full Linux `make gate`.
Native tooltip drawing and foreground UI acceptance remain open. Windows and
macOS remain deferred.
