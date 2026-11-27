# Export applied interface settings

Current case 114 passed all 10 steps in the native X11 window. Export
wrote a new XML file with the ribbon layout, custom panels, Quick Access,
keyboard mappings, marking menus, and mini-toolbar visibility. Customize
remained open with a clean draft.

XML version 8 contains a custom panel section and separate snapshots of
the other interface areas. The snapshots use JSON objects in XML state
attributes. Import accepts versions 1 through 8 and stages only the
ribbon settings, including panels, built-in placement, and appearance.
The installed ribbon settings file continues to use versions 5 through 7.

The build used base `01b65db15` with the export changes and existing local
GUI filesystem, document-opening, and document-saving edits. The run
used a writable copy of the published Normal preferences. The 52 focused
GUI tests include a snapshot with changed values and XML format checks.
The native round trip passed 15 steps, the unapplied-edit export passed
14 steps, and the invalid-version import kept its original draft.

[Cucumber](case-114/cucumber.json) · [JUnit](case-114/junit.xml) ·
[Scenario arguments](case-114/scenario-arguments.json) ·
[Source revision](case-114/source-revision.json)
