// SPDX-License-Identifier: GPL-2.0-only
//! Native entry into an extrusion when a new part has no sketch.
use super::ApplicationWorld;
use cucumber::{given, then, when};
use oblikovati_viewport::ViewportPhysicalKey;

#[given("a new Standard (mm) part with no sketch")]
fn new_metric_part(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::N, true);
    world.click_text("Standard metric part");
    world.expect_text("millimeters (mm)");
    world.click_text("Create");
    let browser = world
        .state()
        .browser
        .expect("new part: expected a model browser");
    assert!(
        browser.sketches.is_empty(),
        "new part: expected no sketches"
    );
}
#[when("Extrude is clicked and a displayed origin plane is selected")]
fn extrude_on_origin_plane(world: &mut ApplicationWorld) {
    world.click_text("Extrude");
    world.pump(Vec::new());
    world.click_text("XY Plane");
    world.pump(Vec::new());
}
#[then("Sketch1 opens for profile creation")]
fn first_sketch_is_open(world: &mut ApplicationWorld) {
    let state = world.state();
    assert_eq!(state.authoring, oblikovati_gui::GuiAuthoringContext::Sketch);
    let browser = state.browser.expect("sketch: expected a model browser");
    assert_eq!(browser.sketches.len(), 1, "new part: expected one sketch");
    assert_eq!(browser.sketches[0].name, "Sketch1");
}
