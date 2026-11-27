# Open dialog design views and properties

Two current Document Opening scenarios passed all 27 steps on Linux X11 with
Xvfb, Openbox, and Vulkan Lavapipe. The named view case saved Open Dialog Blue
and Open Dialog Neutral on a disposable copy of the native extrusion. The Open
dialog showed both names and the Primary view. Escape dismissed the list and
preserved Last Active, the preview, and the dialog itself.

The properties case selected the original published extrusion and inspected
its General, Summary, Project, and Thumbnail tabs. Cancelling left the same
selection and preview in the Open dialog.

The runs used the corresponding bindings and existing local GUI filesystem,
opening and saving edits. The published [input checksums](current-inputs.sha256)
matched the isolated copies. See the [scenario inventory](scenario-inventory.json)
and individual Cucumber and JUnit reports. Issue #3978 tracks strict build
and runtime provenance.
