# Retained Point Text defaults

The [first text-08 run](before.log) at `30a723da3` failed when the
scenario expected an empty font family. The native editor correctly
selected Tahoma when the font was available.

The scenario now records the empty editor before typing and compares it
to the new editor after Cancel. This checks the actual font, height,
alignment, rotation and formatting defaults, with empty content. All four
steps and cleanup passed twice against the published feature SHA-256
`0e8f627cd0eb991bc18c2615f759ca92be1e1c121eb0c7a819b68f2b08706a3d`.

- [System font report](system/cucumber.json) and [empty editor](system/editor.png).
- [Tahoma report](tahoma/cucumber.json) and [empty editor](tahoma/editor.png).
  The run used the installed Wine Tahoma files through a private
  Fontconfig configuration. [Font hashes](tahoma-inputs.sha256).

The [binding diagnostic](published-bindings.json.gz) checked 38 scenarios
and 294 steps without a window. It found 65 undefined bindings in nine
scenarios. The remaining source and kernel dependencies are recorded
against the same hash in the behavior ledger. The binding result measures
step coverage. The separate native report supplies behavior evidence.

The [RIGHT-view rotation test](right-view.log.gz) also passed. It now picks
the insertion point inside the current viewport. The [preview](right-view.png)
shows the readable horizontal result at 270 degrees and centered alignment.
The full Text Box and Sheet Color scenario remains blocked on source detail.
