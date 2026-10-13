# Exact reference binding — partial R2/R3

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563),
under ADR-0005. Source identity is the Go `model/identity` triple of context,
entity kind, and opaque lineage payload. This is distinct from raw Go
`kernel/topo` key bytes and that package's different entity-kind numbers.

The model can now admit an immutable exact-name index for a document evaluation.
A producer supplies encoded keys and owned entities together. Admission uses the
existing envelope codec, retains supported Go entity kinds, and refuses duplicate
names. Legacy and current envelopes normalize to the same identity, as Go
`RefKey.Equal` requires. Binary payload bytes are not interpreted or converted to
text. Name enumeration has explicit byte order and is independent of producer
order. No new lineage encoding, namespace convention, or key scheme is introduced.

Binding requires the requested document, the exact owning history, its current
evaluation generation, the requested entity kind, and an exact encoded identity.
A different context or payload is a miss. No ancestor, nearest entity, suffix,
ordinal, or first-match recovery is present. A duplicate key refuses the entire
candidate index. Producer order cannot choose the winner.

The index uses the existing sealed evaluation scope. Admission checks freshness
before and after consuming the producer. Binding and key enumeration reject
stale or closed owners. Reopening the same document or undoing to identical recipe
bytes does not make an old native index current. A new evaluation must build a
new index. Returned entity borrows also borrow the history, so safe Rust cannot
edit that history while consuming an exact result.

The existing Go byte fixture gates canonical encoding and exact legacy/current
binding. Tests also cover malformed and unsupported inputs, wrong kind/context,
wrong document and history, duplicate canonical/legacy aliases, stable ordering,
edit/undo/close invalidation, and mutation during index admission. A compile-fail
test checks the history borrow. The storage owner must explicitly migrate a
known legacy magic-prefix collision with `decode_legacy`; automatic decoding
never retries a failed current envelope as legacy.

This index does not generate or certify lineage. Producers must supply names
from verified operation history and stable feature namespaces, and preserve
occurrence context. Native topology publication, raw Go topology-key migration,
write-time verification against actual kernel entities, split/merge provenance,
and body/face assignment API integration remain open. The current tests bind
named model values; they do not prove CAD face persistence across recompute.
Run `cargo test -p oblikovati-model --test reference_bindings` and `make gate`.
