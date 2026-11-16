# Fillet radius editing

The typed feature API exposes a Fillet radius as scalar zero. Radius edits
accept a positive length or a named length parameter. A private candidate is
replayed by the certified Fillet service before the edit enters history.
Invalid or unsupported input is refused without committing the candidate.
The desktop opens the same Fillet panel from a selected browser feature. It
previews an isolated candidate, cancels without an edit, and commits through
the typed API. The draft is tied to the document generation and is refused
after another edit. See `receipts/2026-11-17-fillet-ui-edit-4045/` for the
native input test and its limits.
