// SPDX-License-Identifier: GPL-2.0-only
//! Open the published native sketch Parts in the stated authoring context.
use super::super::super::{scenario_root, workspace_layout::native_part::edit_named_sketch};
use super::*;

#[given("`fixtures/cad/construction/regular-circle-12mm.opd` is open with Sketch1 being edited")]
fn open_circle_sketch(world: &mut ApplicationWorld) {
    let path = scenario_root().join("fixtures/cad/construction/regular-circle-12mm.opd");
    world.open_fixture(&path);
    world.previous_document = Some(world.active_id());
    edit_named_sketch(world);
    active_editable_sketch(world);
    assert_eq!(world.edited_geometry().circles().len(), 1);
}

#[given("`fixtures/cad/planar-sketch-empty-xy.opd` is open and active outside sketch edit")]
fn open_planar_part(world: &mut ApplicationWorld) {
    let path = scenario_root().join("fixtures/cad/planar-sketch-empty-xy.opd");
    world.open_fixture(&path);
    world.previous_document = Some(world.active_id());
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Part
    );
    sketch_edit_inactive(world);
}
