# Scenario source correction sync

The local snapshot advanced from `48987af7b5cbb06515bc6f30385f508382f34064`
to `8ec8d2c1e569fbc5c8cd2ec85e9b9380c62c147f`. The source is not a Git
checkout; both hashes identify local snapshot commits, not upstream commits.
The 1,292 included feature, Markdown and SVG files matched the live source
before and after copying. `BLOCKERS.md` and captured media are not in the
snapshot. `source-changes.txt` lists the file changes.

The source now marks 38 declarations `@capture_only`. It marks the entire
39-scenario document-creation feature `@capture_host_only`; five of those
also have `@capture_only`. The ledger classifies the 38 as excluded and the
remaining 34 host-only cases as blocked, with explicit reasons. Twenty-five
previous current receipt links are retained as historical records. No skipped
or blocked case is counted as a pass.

A native-runner check selected the document-creation feature without a host
attestation. It executed zero steps, reported 34 host-only and five
capture-only cases, and failed the nonempty-run assertion. The diagnostic
report separately lists the same 39 cases and all 429 steps as not executed.
These are expected checks of the exclusion boundary, not acceptance results.

The corrected `rectangle-two-point-10` scenario is no longer capture-only.
Its first step was unbound in a focused native run. `rectangle-red.log.gz`
retains that failure for the implementation follow-up. No fixture or product
behavior was changed to make it pass.

The focused binding inventory also reports 34 host-only and five capture-only
cases with zero runnable steps. It fails the nonempty-step check, so an empty
binding report cannot be mistaken for coverage.
The 31 historical receipt links attached to blocked or excluded classifications
were checked against their stored SHA-256 hashes after the sync.
