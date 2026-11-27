// SPDX-License-Identifier: GPL-2.0-only
use super::super::diagnostics;
use super::{ApplicationWorld, first_sketch_is_open};
use cucumber::given;
use oblikovati_gui::GuiAuthoringContext;

#[given("a copy of `fixtures/cad/revolve/revolve-profile.opd` is open with Sketch1 being edited")]
fn open_profile_in_sketch_editor(world: &mut ApplicationWorld) {
    let file = "fixtures/cad/revolve/revolve-profile.opd";
    let source = super::super::scenario_root().join(file);
    let copy = world.scratch_folder().join("revolve-profile.opd");
    std::fs::copy(source, &copy).unwrap();
    world.open_fixture(&copy);
    world.right_click(diagnostics::pointer_pixels(world.run("Sketch1").center()));
    world.click_text("Edit Sketch");
    first_sketch_is_open(world);
    assert_eq!(world.edited_geometry().lines().len(), 4);
}

#[given("its rectangular profile has been selected by Extrude")]
fn extrude_waits_for_active_sketch(world: &mut ApplicationWorld) {
    let sketch = world.state().active_sketch.expect("active Sketch1");
    world.click_text("3D Model");
    assert!(world.run_is_enabled("Extrude"));
    world.click_text("Extrude");
    world.pump(Vec::new());
    assert_eq!(world.state().active_sketch, Some(sketch));
    assert_eq!(world.state().authoring, GuiAuthoringContext::Sketch);
    assert!(!world.state().extrude_preview_active);
    assert!(diagnostics::window_geometry("Extrude").is_none());
    world.click_text("Sketch");
}
