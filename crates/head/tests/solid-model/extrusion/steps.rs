// SPDX-License-Identifier: GPL-2.0-only
//! Native entry into an extrusion when a new part has no sketch.
use super::ApplicationWorld;
use cucumber::{given, then, when};
use oblikovati_viewport::ViewportPhysicalKey;
mod acceptance;
#[cfg(target_os = "linux")]
mod viewport_interaction;

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

#[given("a rectangle was drawn in the sketch requested by Extrude")]
fn rectangle_for_waiting_extrusion(world: &mut ApplicationWorld) {
    new_metric_part(world);
    rectangle_in_waiting_extrusion(world);
}

fn rectangle_in_waiting_extrusion(world: &mut ApplicationWorld) {
    extrude_on_origin_plane(world);
    first_sketch_is_open(world);
    world.draw_rectangle();
    assert_eq!(world.edited_geometry().lines().len(), 4, "rectangle edges");
}

#[when("Finish Sketch is clicked")]
fn finish_requested_profile(world: &mut ApplicationWorld) {
    world.finish_sketch();
}

#[then("the Extrusion panel shows one selected profile")]
fn waiting_extrusion_shows_profile(world: &mut ApplicationWorld) {
    world.expect_text("Profiles");
    world.expect_text("1 Profile");
}

#[then("a solid preview displays Distance A of 10 mm")]
fn waiting_extrusion_shows_default_preview(world: &mut ApplicationWorld) {
    world.expect_text("Distance A");
    world.expect_text("10 mm");
    assert!(
        !world.app().window.scene_colors.is_empty(),
        "expected preview triangles submitted to the native viewport"
    );
    let document = world.active_id();
    let session = world
        .app()
        .owner
        .documents()
        .workspace()
        .session(document)
        .unwrap();
    assert!(
        session.snapshot().work_geometry.features().is_empty(),
        "preview must not commit a feature"
    );
    let capture = super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::diagnostics::window_occlusions();
    assert!(
        super::solid_visibility::visible_preview_fill(&capture, &windows),
        "expected a shaded extrusion preview outside the panels"
    );
}

#[given("Default direction with a selected profile")]
fn selected_rectangle_preview(world: &mut ApplicationWorld) {
    rectangle_for_waiting_extrusion(world);
    finish_requested_profile(world);
    world.pump(Vec::new());
    waiting_extrusion_shows_profile(world);
    waiting_extrusion_shows_default_preview(world);
}

fn replace_extrusion_distance(world: &mut ApplicationWorld, value: &str) {
    let label = world.run_in_window("Extrude", "DistanceA").clone();
    world.click([label.minimum[0] as i32 - 30, label.center()[1] as i32]);
    world.key(ViewportPhysicalKey::A, true);
    world.type_text(value);
    world.pump(Vec::new());
    let label = world.run_in_window("Extrude", "DistanceA").clone();
    assert!(
        world.frame.runs.iter().any(|run| {
            run.text == value
                && (run.center()[1] - label.center()[1]).abs() < 4.
                && run.maximum[0] <= label.minimum[0]
        }),
        "Distance A: expected {value:?} in the input field"
    );
}

#[when("Distance A is replaced with 0")]
fn replace_with_zero(world: &mut ApplicationWorld) {
    replace_extrusion_distance(world, "0");
}

#[when("the value is replaced with 8")]
fn replace_with_eight(world: &mut ApplicationWorld) {
    replace_extrusion_distance(world, "8");
}

#[then("a red error indicator appears and OK is disabled")]
fn zero_is_rejected(world: &mut ApplicationWorld) {
    let color = world.run("Invalid distance:").color;
    assert!(
        color[0] > color[1] && color[0] > color[2],
        "expected a red error, got {color:?}"
    );
    assert!(!world.run_is_enabled("OK"), "expected OK disabled");
}

#[then("the error clears and OK becomes available")]
fn positive_distance_is_accepted(world: &mut ApplicationWorld) {
    world.expect_no_text("Invalid distance:");
    assert!(world.run_is_enabled("OK"), "expected OK enabled");
}

#[given("the uncommitted rectangle extrusion preview")]
fn uncommitted_rectangle_preview(world: &mut ApplicationWorld) {
    selected_rectangle_preview(world);
}

#[when("Cancel is clicked")]
fn cancel_extrusion(world: &mut ApplicationWorld) {
    world.click_text("Cancel");
    world.pump(Vec::new());
}

#[then("the shaded preview disappears and Sketch1 remains")]
fn cancelled_preview_keeps_sketch(world: &mut ApplicationWorld) {
    assert!(
        !world.state().extrude_preview_active,
        "preview remains active"
    );
    assert!(
        world.app().window.scene_triangles.is_empty(),
        "preview scene remains"
    );
    world.expect_text("Sketch1");
    expect_unmodified_source(world);
}

fn expect_unmodified_source(world: &mut ApplicationWorld) {
    let document = world.active_id();
    let session = world
        .app()
        .owner
        .documents()
        .workspace()
        .session(document)
        .unwrap();
    let snapshot = session.snapshot();
    assert_eq!(snapshot.sketches.sketches().len(), 1, "source sketch count");
    assert_eq!(
        snapshot.sketches.sketches()[0].geometry().lines().len(),
        4,
        "source rectangle edges"
    );
    assert!(
        snapshot.work_geometry.features().is_empty(),
        "cancel committed a feature"
    );
}

#[when("the user presses E")]
fn press_extrude_key(world: &mut ApplicationWorld) {
    use oblikovati_viewport::{ViewportEvent, ViewportInput, ViewportModifiers};
    let event = |pressed| {
        ViewportEvent::Input(ViewportInput::PhysicalKey {
            key: ViewportPhysicalKey::E,
            layout_key: ViewportPhysicalKey::E,
            pressed,
            command: None,
            modifiers: ViewportModifiers::default(),
        })
    };
    world.pump(vec![
        event(true),
        ViewportEvent::Input(ViewportInput::Text('e')),
        event(false),
    ]);
    world.pump(Vec::new());
}

#[then("the extrusion panel reopens for the source sketch")]
fn extrusion_reopens_for_sketch(world: &mut ApplicationWorld) {
    world.run_in_window("Extrude", "Sketch1");
    world.expect_text("Profiles");
}

#[test]
fn unconstrained_rectangle_resumes_waiting_extrusion_with_preview() {
    let mut world = ApplicationWorld::new();
    rectangle_for_waiting_extrusion(&mut world);
    finish_requested_profile(&mut world);
    world.pump(Vec::new());
    waiting_extrusion_shows_profile(&mut world);
    waiting_extrusion_shows_default_preview(&mut world);
    world.close();
}
