// SPDX-License-Identifier: GPL-2.0-only
//! Open part settings from a width-compacted ribbon panel.
use super::*;

#[given("a part is active in the MCAD Software with the full Tools ribbon displayed")]
fn part_with_tools(world: &mut ApplicationWorld) {
    world.previous_document = Some(world.add_part());
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
    world.expect_text("Length precision");
}

#[then(
    "the dialog shows Standard, Units, Sketch, Modeling, Bill of Materials and Default Tolerance tabs"
)]
fn settings_tabs_visible(world: &mut ApplicationWorld) {
    let contents = world.window_text("Document Settings");
    for tab in [
        "Standard",
        "Units",
        "Sketch",
        "Modeling",
        "BillofMaterials",
        "DefaultTolerance",
    ] {
        assert!(
            contents.contains(tab),
            "Document Settings: expected {tab:?} in {contents:?}"
        );
    }
}

#[when("the user closes the dialog without changing settings")]
fn close_settings(world: &mut ApplicationWorld) {
    let at = super::super::diagnostics::pointer_pixels(world.run("Close").center());
    super::super::x11_input::click(at);
    world.pump(Vec::new());
}

#[then("the dialog and temporary Options panel are closed")]
fn settings_and_options_closed(world: &mut ApplicationWorld) {
    assert!(window_geometry("Document Settings").is_none());
    world.expect_no_text("Length precision");
    world.expect_no_text("Document Settings");
}
