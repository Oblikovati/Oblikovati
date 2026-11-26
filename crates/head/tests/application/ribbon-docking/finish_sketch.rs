// SPDX-License-Identifier: GPL-2.0-only
//! Finish the active sketch directly from the left-docked ribbon.
use super::*;
use cucumber::{given, then, when};
use oblikovati_gui::GuiAuthoringContext;

#[given(
    "`fixtures/cad/planar-sketch-empty-xy.opd` is open with `Sketch1` being edited and the ribbon docked at the top"
)]
#[given(
    "`fixtures/cad/planar-sketch-empty-xy.opd` is open with Sketch1 being edited in the MCAD Software"
)]
#[given(
    "`fixtures/cad/planar-sketch-empty-xy.opd` is open and active in the MCAD Software with Sketch1 being edited"
)]
fn open_top_docked_sketch(world: &mut ApplicationWorld) {
    let path = super::super::scenario_root().join("fixtures/cad/planar-sketch-empty-xy.opd");
    world.open_fixture(&path);
    edit_top_docked_sketch(world);
}

#[given("an existing sketch is being edited with the ribbon docked at the top")]
fn edit_existing_top_docked_sketch(world: &mut ApplicationWorld) {
    world.add_part();
    world.add_finished_sketch();
    edit_top_docked_sketch(world);
}

fn edit_top_docked_sketch(world: &mut ApplicationWorld) {
    world.previous_document = Some(world.active_id());
    super::super::workspace_layout::browser_feature::right_click_row(world, "Sketch1");
    super::super::workspace_layout::browser_feature::choose_menu_row(world, "Edit Sketch");
    assert_eq!(world.state().authoring, GuiAuthoringContext::Sketch);
    world.run_in_window("##application-ribbon", "FinishSketch");
    world.record_initial_sketch();
}

#[when("the user docks the Sketch ribbon on the left")]
fn dock_sketch_tab_left(world: &mut ApplicationWorld) {
    let at = world
        .run_in_window("##application-ribbon", "Sketch")
        .center();
    world.right_click(super::super::diagnostics::pointer_pixels(at));
    world.pump(Vec::new());
    open_docking_positions(world);
    world.click_text("Left");
    world.pump(Vec::new());
}

#[then("the sketch tools appear in a vertical group column")]
fn sketch_groups_stack(world: &mut ApplicationWorld) {
    expect_vertical(
        world,
        &["Create", "Constrain", "Modify", "Insert", "Format"],
    );
}

#[then("Finish remains visible in the Exit group at the bottom")]
fn finish_is_direct(world: &mut ApplicationWorld) {
    let finish = world.run_in_window(COLUMN, "FinishSketch");
    let format = world.run_in_window(COLUMN, "Format");
    let exit = world.run_in_window(COLUMN, "EXIT");
    assert!(finish.center()[1] > format.center()[1]);
    assert!(exit.center()[1] > finish.center()[1]);
    assert!(world.run_is_enabled("Finish Sketch"));
}

#[then("sketch editing ends and the part model view returns")]
fn returns_to_part(world: &mut ApplicationWorld) {
    assert_eq!(world.state().authoring, GuiAuthoringContext::Part);
    assert!(world.app().owner.active_sketch_edit().is_none());
    same_part(world);
}

#[then("the 3D Model tab and groups are active")]
fn left_model_groups_visible(world: &mut ApplicationWorld) {
    for label in ["3DModel", "Create", "Modify", "WorkFeatures"] {
        world.run_in_window(COLUMN, label);
    }
}

#[then("no sketch geometry has changed")]
fn finish_keeps_sketch_geometry(world: &mut ApplicationWorld) {
    let id = world.active_id();
    let original = world.observed_sketch.as_ref().unwrap().geometry.clone();
    let session = world
        .app()
        .owner
        .documents()
        .workspace()
        .session(id)
        .unwrap();
    let sketches = session.snapshot().sketches.sketches();
    let sketch = sketches
        .iter()
        .find(|sketch| sketch.name() == "Sketch1")
        .unwrap();
    assert_eq!(sketch.geometry(), &original);
}
