# Connected client applications — partial R2

Tracking: [#3697](https://github.com/Oblikovati/Oblikovati/issues/3697).
The workspace session owns external automation connections separately from
in-process add-ins and document snapshots. Registration order is stable; names
are exact and may repeat for independent connections. IDs start at one and are
never reused during the session. Empty names, unknown removal IDs, and exhausted
signed 64-bit IDs fail without registry mutation.

The typed session and file hosts implement the canonical `clientApps.register`,
`clientApps.unregister`, and `clientApps.list` methods. Immutable query hosts
support only listing. Empty lists encode as arrays. Connection changes do not
require a document and do not create document history or change focus. The
existing owner-borrow guard rejects synchronous reentry for these calls too.

Open-time interest discovery now checks the connected registry before asking
the workspace observer about registered add-ins or other host-provided clients.
A missing-client notice remains suppressed while at least one matching external
connection remains. Disconnecting the last matching client makes it unavailable
for the next newly opened document; existing documents are not reopened.

Tests cover registration order, duplicates, nonreused and exhausted IDs, exact
name matching, validation without mutation, canonical typed requests/responses,
malformed JSON, immutable-host refusal, document independence, reentry, and
open-time notice suppression/removal. The shared host inventory now has 31
queries, 23 document edits, six transaction methods, and two client commands.

Native connection transport, disconnect callbacks, in-process add-in loading,
CLI session startup, and UI notices remain pending. Clients currently register
and unregister explicitly, as in the Go registry contract.
