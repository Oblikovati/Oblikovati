# Fillet radius editing

The typed feature API exposes a Fillet radius as scalar zero. Radius edits
accept a positive length or a named length parameter. A private candidate is
replayed by the certified Fillet service before the edit enters history.
Invalid or unsupported input is refused without committing the candidate.
The desktop radius-edit tool remains open. See
`receipts/2026-11-16-fillet-history-4045/` for the owner test and its limits.
