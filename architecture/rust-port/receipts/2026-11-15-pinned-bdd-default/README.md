# Pinned native scenario source

On a task checkout based at `56befef58`, the default source resolver regression
first failed because it returned the old unversioned benchmark directory. The
new resolver restores the pinned snapshot and checks its revision and contents.
The test-thread checkout is removed after the run.

The default application binding inventory used the clean pinned snapshot
`48987af7b5cbb06515bc6f30385f508382f34064`: it found 24 features and 753
expanded scenarios, with 2,789 unresolved steps. Its intentional failure and
the retained full inventory are diagnostic, not acceptance. It did not filter
unresolved steps.

A native frame-loop run of `Create a standard metric part` passed all eight
steps using the default restored snapshot. The checked source revision, Cucumber
and JUnit reports, and inspected capture are retained here. This focused run
does not establish the full feature or operating-system input acceptance.
A separate unversioned synthetic override ran one step and recorded a null
commit rather than claiming the pin.

The strict gate test command requires the pin and refuses diagnostic or filtered
runs. Focused tests cover a dirty or missing override, changed and missing
files, source-result invalidation, and source selection. An occupied report
folder was refused with its previous artifact intact. Unique default report
folders remain available after the test process exits. Snapshot, report,
source-revision and gate-command tests, 53 Python tests, strict head and build
Clippy, formatting and `make docs-lint` passed. The full gate is still blocked
by the unreleased API pin and missing scenario evidence (#3965, #3865).

`OBK_BDD_ROOT` overrides remain available for diagnostic source checkouts. Only
strict gate evidence requires the exact clean pin. The full command output and
negative controls are retained in the compressed logs.
