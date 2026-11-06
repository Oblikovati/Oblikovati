# Current display names in saved manifests

Tracking: [#3752](https://github.com/Oblikovati/Oblikovati/issues/3752).
Source: Go `persistence/store.go`, revision `aa8297d1`.

The native CLI new/info test found that saving a derived title wrote an empty
manifest name. Go PackageStore writes the current Document.DisplayName.
The file owner now supplies the effective name to the snapshot save candidate.
The retained package and path change only after successful atomic replacement.

Ordinary Save derives the title from the bound filename when no authored
name exists. Save As derives it from the destination and retains derived
ownership for later renames. Save Copy keeps the source's effective title;
copy metadata can override it. Authored titles remain unchanged by path edits.
No second filename or title owner is introduced.

The persistence target applies the title during candidate preparation. Changing
its title after preparation invalidates the earlier candidate. Failed storage
cannot publish title changes to the retained package. Model recipe capture and
property checks remain shared with the ordinary save pipeline.

Regression tests cover ordinary save, repeated Save As, authored titles,
copy inheritance and overrides, and failures before persistence. Native CLI
creation and full package loads verify the new/info path for all four kinds.
The earlier empty-name copy assertion was inconsistent with Go's save contract;
it now checks the source's current derived title. Full Linux `make gate` remains
the acceptance command. Kernel behavior and other platforms are unchanged.
