# Current native populated drawing layout

Both cancellation scenarios passed all 32 steps including their setup.
Canceling component selection kept the original Part and created no tab.
Canceling the nested Browse dialog restored the proposed component name.
Canceling the outer dialog then returned to the same Part.

The two confirmation scenarios still require drawing placement, a view
referencing the selected model, and populated parts-list results (#3899).
The component dialog disables OK. The Cucumber report lists the first
undefined setup or selection step for each blocked scenario.

The run used native build base `c9333c66e8db6caba5647a023fdc8498306b6d55` with the existing local
GUI filesystem, opening and saving edits on Linux X11 with Xvfb, Openbox,
and Vulkan Lavapipe. The published project, template, and document hashes
matched the isolated inputs. Strict build and runtime provenance is open
under #3978. See the [Cucumber report](cucumber.json),
[scenario inventory](scenario-inventory.json), and [input hashes](current-inputs.sha256).
