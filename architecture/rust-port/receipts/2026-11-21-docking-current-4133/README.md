# Docking System on the published source

All 11 scenarios and 108 steps passed on application revision
`0107fd3e5414fabdc7a21743ec3acee48df794b1`. The Linux run used Xvfb at
1440 × 900, Openbox, and Vulkan Lavapipe. The
[Cucumber report](cucumber.json), [JUnit report](junit.xml.gz), and
[source report](source-revision.json) identify the cases and source. The
[feature and fixture hashes](inputs-before.sha256) matched before and after
the run. Every scenario passed its cleanup hook and produced a final capture.

The [left docking capture](left-docking-preview.png) shows the target before
release. The restart driver launched two separate desktop processes with
one isolated profile. The [first process](before.png) closed through its
title-bar X. The [second process](after.png) opened with the same sizes,
positions, and tabs. The dock region had zero changed pixels. The
[restart report](restart-summary.json) records the dock ID, tab order, and
fixture and executable hashes. Both screenshots show the open Part in the
Model browser and its solid in the viewport. A pixel check confirmed that
the face and background regions match before and after restart.

The [focused restart scenario](viewport-restart-cucumber.json) also passed
with the new visibility check in the restart driver. In each capture, every
pixel in the sampled face exceeded the brightness threshold, and every pixel
in the sampled background stayed below it. The
[viewport measurements](viewport-restart-summary.json) bind both samples to
the same native fixture and desktop executable.

The Go reference passed its [panel registry and shared close tests](go-reference.log)
and its [native docked viewport interaction test](go-viewport.log). The
two-process restart and the other current cases are covered by the Rust run.

## Modal window audit

The separate audit under [#4073](https://github.com/Oblikovati/Oblikovati/issues/4073)
remains open. In the [native drag run](file-properties-pointer.log.gz),
File Properties stayed floating when dragged onto the Material Browser dock.
Dear ImGui 1.89.2 rejects a dock drop for a window begun inside a popup.
The [forced dock run](file-properties-content-close.log.gz) attached File
Properties to the shared node through the dock builder. Clicking its Cancel
button did not close it within eight frames. In the
[parent close run](modal-parent-close.log.gz), clicking Cancel in the Open
dialog also left File Properties open within the same bound.

On the committed revision, the [existing controls](baseline-modal-controls.log.gz)
showed the same close failure for Projects with a restored dock assignment.
The floating File Properties close test passed and retained the selected
file in Open.

These audit cases are separate from the 11 published Docking System
scenarios. Strict build and runtime provenance remains open under
[#3978](https://github.com/Oblikovati/Oblikovati/issues/3978).
