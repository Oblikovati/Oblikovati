// SPDX-License-Identifier: GPL-2.0-only
//! Verify the assembly and drawing ribbons without starting a command (#3956).
use super::super::scenario_root;
use super::{ApplicationWorld, diagnostics};
use cucumber::{given, then, when};
use oblikovati_api::types::{DOCUMENT_ASSEMBLY, DOCUMENT_DRAWING, DOCUMENT_PART, DocumentType};

const RIBBON: &str = "##application-ribbon";
const ASSEMBLY: &str = "fixtures/cad/blank-metric-assembly.oad";
const DRAWING: &str = "fixtures/cad/blank-drawing.odd";
pub(super) const SKETCH_PART: &str = "fixtures/cad/revolve/revolve-profile.opd";

#[given("`fixtures/cad/revolve/revolve-profile.opd` is open and active in the MCAD Software")]
fn open_sketch_part(world: &mut ApplicationWorld) {
    open_document_kind(world, SKETCH_PART, DOCUMENT_PART);
    let browser = world.state().browser.expect("expected the Part browser");
    assert_eq!(browser.sketches.len(), 1);
    assert_eq!(browser.sketches[0].name, "Sketch1");
    assert!(browser.bodies.is_empty());
}

#[given(
    "the saved empty assembly fixture \"fixtures/cad/blank-metric-assembly.oad\" is active in the MCAD Software"
)]
fn open_assembly(world: &mut ApplicationWorld) {
    open_document_kind(world, ASSEMBLY, DOCUMENT_ASSEMBLY);
}

#[given("`fixtures/cad/blank-drawing.odd` is open and active in the MCAD Software")]
fn open_drawing(world: &mut ApplicationWorld) {
    open_document_kind(world, DRAWING, DOCUMENT_DRAWING);
}

fn open_document_kind(world: &mut ApplicationWorld, fixture: &str, expected: DocumentType) {
    world.open_fixture(&scenario_root().join(fixture));
    let active = world.active_id();
    assert_eq!(world.state().document_type, Some(expected));
    assert!(!world.state().document_storage[&active].unsaved_edits);
    world.previous_document = Some(active);
}

#[then("the assembly document remains active")]
fn assembly_remains_active(world: &mut ApplicationWorld) {
    assert_eq!(
        Some(world.active_id()),
        world.previous_document,
        "expected the same saved assembly to remain active"
    );
    assert_eq!(world.state().document_type, Some(DOCUMENT_ASSEMBLY));
}

#[then("no assembly command has been started")]
fn assembly_idle(world: &mut ApplicationWorld) {
    assert!(!world.state().transaction_open);
    assert!(diagnostics::window_geometry("Place Component").is_none());
    let document = world.active_id();
    assert!(!world.state().document_storage[&document].unsaved_edits);
}

#[when("the user selects the Place Views ribbon tab")]
fn select_place_views(world: &mut ApplicationWorld) {
    select_ribbon_tab(world, "PlaceViews");
}

#[when("the user selects the Annotate ribbon tab")]
fn select_annotate(world: &mut ApplicationWorld) {
    select_ribbon_tab(world, "Annotate");
}

fn select_ribbon_tab(world: &mut ApplicationWorld, label: &str) {
    let position = world.run_in_window(RIBBON, label).center();
    world.click(diagnostics::pointer_pixels(position));
    world.pump(Vec::new());
}

#[then("the Place Views ribbon displays Create, Modify, Sketch and Sheets groups")]
fn drawing_view_groups(world: &mut ApplicationWorld) {
    expect_ribbon_groups(world, &["Create", "Modify", "Sketch", "Sheets"]);
}

#[then("the Annotate ribbon displays Dimension, Text and Symbols commands")]
fn drawing_annotation_groups(world: &mut ApplicationWorld) {
    expect_ribbon_groups(world, &["Dimension", "Text", "Symbols"]);
}

fn expect_ribbon_groups(world: &ApplicationWorld, labels: &[&str]) {
    for label in labels {
        world.run_in_window(RIBBON, &label.to_ascii_uppercase());
    }
}
