// SPDX-License-Identifier: GPL-2.0-only
//! Open part settings from a width-compacted ribbon panel.
use super::*;

#[given("a part is active in the MCAD Software with the full Tools ribbon displayed")]
fn part_with_tools(world: &mut ApplicationWorld) {
    world.previous_document = Some(world.add_part());
    full_tools(world);
}

#[given(
    "`fixtures/cad/blank-metric-part.opd` is open with the full Tools ribbon displayed in the MCAD Software"
)]
fn named_part_with_tools(world: &mut ApplicationWorld) {
    let path = super::super::scenario_root().join("fixtures/cad/blank-metric-part.opd");
    world.open_fixture(&path);
    world.previous_document = Some(world.active_id());
    full_tools(world);
}

#[given("the narrow window shows Options as a compact panel button")]
fn narrow_options(world: &mut ApplicationWorld) {
    super::window_compaction::narrow_window(world);
    world.expect_text("Options");
    world.expect_no_text("Document Settings");
}

#[when("the user opens the Options panel")]
fn open_options(world: &mut ApplicationWorld) {
    super::view_compaction::click_compact_panel(world, "Options");
    world.expect_text("Document Settings");
}

#[when("the user selects Document Settings")]
fn open_document_settings(world: &mut ApplicationWorld) {
    let at = super::super::diagnostics::pointer_pixels(world.run("Document Settings").center());
    super::super::x11_input::click(at);
    world.pump(Vec::new());
}

#[then("the Document Settings dialog opens for the active part")]
#[then("the active part's Document Settings dialog opens")]
fn settings_open(world: &mut ApplicationWorld) {
    assert!(window_geometry("Document Settings").is_some());
    assert_eq!(world.active_id(), world.previous_document.unwrap());
}

#[then(
    "the dialog shows Standard, Units, Sketch, Modeling, Bill of Materials and Default Tolerance tabs"
)]
fn settings_tabs_visible(world: &mut ApplicationWorld) {
    for tab in [
        "Standard",
        "Units",
        "Sketch",
        "Modeling",
        "BillofMaterials",
        "DefaultTolerance",
    ] {
        world.run_in_window("Document Settings", tab);
    }
}

#[then("Standard is the selected tab")]
fn standard_tab_selected(world: &mut ApplicationWorld) {
    let contents = world.window_text("Document Settings");
    assert!(contents.contains("PartMaterial:"));
    assert!(contents.contains("PartAppearance:"));
    assert!(!contents.contains("Lengthprecision"));
}

#[then("Close is available while Cancel and Apply are unavailable")]
fn initial_settings_actions(world: &mut ApplicationWorld) {
    assert_eq!(world.run_in_window("Document Settings", "Close").alpha, 255);
    for label in ["Cancel", "Apply"] {
        assert!(world.run_in_window("Document Settings", label).alpha < 255);
    }
}

#[when("the user closes the dialog without changing settings")]
fn close_settings(world: &mut ApplicationWorld) {
    let close = world.run_in_window("Document Settings", "Close");
    let at = super::super::diagnostics::pointer_pixels(close.center());
    super::super::x11_input::click(at);
    world.pump(Vec::new());
    assert!(window_geometry("Document Settings").is_none());
}

#[then("the dialog and temporary Options panel are closed")]
fn settings_and_options_closed(world: &mut ApplicationWorld) {
    assert!(window_geometry("Document Settings").is_none());
    world.expect_no_text("Length precision");
    world.expect_no_text("Document Settings");
}
