# Template material gap

Application base: `9a8729c95`. Source scenario:
`features/application/document-creation/document-creation.feature`,
“Inspect the current session's initial template selection”, pinned at
`48987af7b5cbb06515bc6f30385f508382f34064`.

A temporary native regression opened the chooser, selected `Standard metric
part`, verified the selection, and asserted the rendered text `Material:
Generic`. On Linux with Xvfb and Vulkan Lavapipe, this command failed at that
assertion after compiling:

```sh
XMODIFIERS=@im=none VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
  xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  selected_metric_part_template_shows_its_material_in_details -- --nocapture
```

The selected template details contained `Document type: Part` and `Length:
millimeters (mm)`, but no material. The complete output is in `native-red.log`. The test was temporary because the
scenario fixture does not provide a material definition to build a truthful
template. Its body, inserted in the existing `gallery_secondary` test module,
was:

```rust
#[test]
fn selected_metric_part_template_shows_its_material_in_details() {
    let mut world = ApplicationWorld::new();
    world.key(ViewportPhysicalKey::N, true);
    world.click_text(METRIC_PART);
    expect_selected_template(&world, METRIC_PART);
    world.expect_text("Material: Generic");
    world.close();
}
```

The built-in chooser creates an empty Part with no material assignment. The
bundled material catalog and Go reference do not provide a `Generic` material
asset. The scenario's `FIXTURES.md` describes an observed configured session,
not a fresh-install default or a transferable template asset.

Issue [#3932](https://github.com/Oblikovati/Oblikovati/issues/3932) tracks the
missing template metadata and display. A correct fix must use the selected
template's actual material and preserve it in the created document. A hard-coded
label or invented physical properties would not verify that behavior.

An unassigned material is valid in the current model. The chooser may show that
no material is assigned; it must not label that state as a missing-asset error.
If a template declares an identity that cannot resolve, report that as a named
missing-material diagnostic.

The chooser currently receives only a kind, units, and an encoded-size estimate.
Actual creation reaches `creation_package`, `DocumentPackage::empty`, and
`restore_assets` through the document owner. There is no shared template
package or descriptor exposed to the chooser. A reliable implementation must
use the same authoritative source for creation and details; a separately
constructed empty package could diverge when file templates are supported.
