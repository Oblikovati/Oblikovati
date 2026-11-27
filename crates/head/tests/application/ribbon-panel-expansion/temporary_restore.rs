// SPDX-License-Identifier: GPL-2.0-only
//! Restore the full ribbon after opening a temporary expansion (#3856).
use super::{ApplicationWorld, diagnostics};
use cucumber::{given, then};

#[given("the 3D Model tab is selected")]
fn model_tab_selected(world: &mut ApplicationWorld) {
    super::super::ribbon_display::model_selected(world);
}
#[then("its ribbon panels appear temporarily")]
fn temporary_model_panels(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("##ribbon-temporary-panel").is_some());
    world.expect_text("Start 2D Sketch");
}

#[then("the temporary ribbon and pinned additional area remain visible")]
fn temporary_pinned_expansion_remains(world: &mut ApplicationWorld) {
    temporary_model_panels(world);
    super::pinned_visible(world);
}

#[then("the additional commands appear below the temporary ribbon")]
fn additional_below_temporary(_: &mut ApplicationWorld) {
    let (area, _) = diagnostics::window_geometry("##ribbon-additional-PartModify").unwrap();
    let (ribbon, size) = diagnostics::window_geometry("##ribbon-temporary-panel").unwrap();
    assert!(area[1] >= ribbon[1] + size[1]);
}

#[then("the temporary ribbon and its additional command area close")]
fn temporary_model_closes(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("##ribbon-temporary-panel").is_none());
    assert!(diagnostics::window_geometry("##ribbon-additional-PartModify").is_none());
    world.expect_no_text("Start 2D Sketch");
    world.expect_text("3D Model");
}
