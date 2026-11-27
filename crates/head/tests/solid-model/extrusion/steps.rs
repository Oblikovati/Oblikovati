// SPDX-License-Identifier: GPL-2.0-only
//! Native entry into an extrusion when a new part has no sketch.
use super::ApplicationWorld;
use cucumber::{given, then, when};
use oblikovati_viewport::ViewportPhysicalKey;
pub(super) mod acceptance;
#[cfg(target_os = "linux")]
mod boolean;
#[cfg(target_os = "linux")]
mod direction;
#[cfg(target_os = "linux")]
mod direction_selector;
#[cfg(target_os = "linux")]
mod distance_editor;
#[cfg(target_os = "linux")]
mod distance_handle;
#[cfg(target_os = "linux")]
mod driven_distance;
#[cfg(target_os = "linux")]
mod edit_properties;
#[cfg(target_os = "linux")]
mod reference_extent;
#[cfg(target_os = "linux")]
mod taper;
#[cfg(target_os = "linux")]
mod viewport_chamfer;
#[cfg(target_os = "linux")]
pub(super) mod viewport_fillet;
#[cfg(target_os = "linux")]
mod viewport_interaction;

#[given("a new Standard (mm) part with no sketch")]
fn new_metric_part(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::N, true);
    world.click_text("Standard metric part");
    world.expect_text("millimeters (mm)");
    world.click_text("Create");
    assert_no_sketches(world);
}

#[given("a copy of `fixtures/cad/revolve/revolve-blank.opd` is open with no sketch")]
fn published_blank_part(world: &mut ApplicationWorld) {
    let source = super::scenario_root().join("fixtures/cad/revolve/revolve-blank.opd");
    let copy = world.scratch_folder().join("revolve-blank.opd");
    std::fs::copy(source, &copy).unwrap();
    world.open_fixture(&copy);
    assert_no_sketches(world);
}

fn assert_no_sketches(world: &mut ApplicationWorld) {
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
    world.expect_text("10.00 mm");
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
    world.click(distance_a_input(world));
    world.key(ViewportPhysicalKey::A, true);
    world.type_text(value);
    world.pump(Vec::new());
    let label = world.run_in_window("Extrude", "DistanceA").clone();
    assert!(
        world.frame.runs.iter().any(|run| {
            run.text == value
                && (run.center()[1] - label.center()[1]).abs() < 4.
                && run.minimum[0] > label.maximum[0]
        }),
        "Distance A: expected {value:?} in the input field"
    );
}

fn distance_a_input(world: &ApplicationWorld) -> [i32; 2] {
    let (position, size) = super::diagnostics::window_geometry("Extrude").unwrap();
    let label = world.run_in_window("Extrude", "DistanceA");
    super::diagnostics::pointer_pixels([position[0] + size[0] * 0.55, label.center()[1]])
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

#[test]
fn invalid_distance_then_valid_distance_restores_the_preview() {
    let mut world = ApplicationWorld::new();
    selected_rectangle_preview(&mut world);
    replace_extrusion_distance(&mut world, "0");
    assert!(
        !world.state().extrude_preview_active,
        "invalid distance must clear preview"
    );
    replace_extrusion_distance(&mut world, "8");
    assert!(
        world.state().extrude_preview_active,
        "valid distance must restore preview"
    );
    let capture = super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::diagnostics::window_occlusions();
    assert!(super::solid_visibility::visible_preview_fill(
        &capture, &windows
    ));
    expect_unmodified_source(&mut world);
    world.close();
}

#[cfg(target_os = "linux")]
#[test]
fn preview_tracks_a_named_parameter_edit_and_cancel() {
    use oblikovati_api::{client::Client, wire::ParameterSetArgs};
    let mut world = ApplicationWorld::new();
    selected_rectangle_preview(&mut world);
    let parameter = |expression: &str| ParameterSetArgs {
        name: "depth".into(),
        expression: expression.into(),
        ..Default::default()
    };
    Client::new(world.app().owner.document_host())
        .parameters()
        .add(&parameter("10 mm"))
        .unwrap();
    world.pump(Vec::new());
    replace_extrusion_distance(&mut world, "depth");
    assert!(world.state().extrude_preview_active);
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        oblikovati_viewport::ViewportNavigation::Home { margin: 0.1 },
    )
    .unwrap();
    world.pump(Vec::new());
    let before_scene: Vec<_> = world
        .app()
        .window
        .scene_triangles
        .iter()
        .map(|triangle| triangle.positions)
        .collect();
    let before = super::capture_framebuffer(&mut world.app().window.native);
    Client::new(world.app().owner.document_host())
        .parameters()
        .set(&parameter("20 mm"))
        .unwrap();
    for _ in 0..4 {
        world.pump(Vec::new());
    }
    assert_ne!(
        world
            .app()
            .window
            .scene_triangles
            .iter()
            .map(|triangle| triangle.positions)
            .collect::<Vec<_>>(),
        before_scene,
        "parameter edit must update preview geometry"
    );
    let after = super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::diagnostics::window_occlusions();
    assert!(viewport_interaction::preview_changed_outside_windows(
        &before, &after, &windows
    ));
    cancel_extrusion(&mut world);
    cancelled_preview_keeps_sketch(&mut world);
    world.close();
}

#[cfg(target_os = "linux")]
#[test]
fn native_invalid_distance_recovery_and_cancel_preserve_the_sketch() {
    let mut world = ApplicationWorld::new();
    selected_rectangle_preview(&mut world);
    viewport_interaction::replace_distance_native(&mut world, "0");
    zero_is_rejected(&mut world);
    assert!(!world.state().extrude_preview_active);
    viewport_interaction::replace_distance_native(&mut world, "8");
    positive_distance_is_accepted(&mut world);
    assert!(world.state().extrude_preview_active);
    let capture = super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::diagnostics::window_occlusions();
    assert!(super::solid_visibility::visible_preview_fill(
        &capture, &windows
    ));
    viewport_interaction::click_text_native(&mut world, "Cancel");
    cancelled_preview_keeps_sketch(&mut world);
    world.close();
}
