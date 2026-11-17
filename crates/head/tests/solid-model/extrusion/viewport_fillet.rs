// SPDX-License-Identifier: GPL-2.0-only
//! Native Fillet selection and creation for viewport scenarios (#4045).
use super::acceptance::feature_count;
use super::viewport_interaction::committed_extrusion;
use super::*;

#[given("a valid solid model is open and a feature tool has a viewport selection field active")]
#[given("a solid feature tool is requesting model geometry")]
fn fillet_waits_for_edge(world: &mut ApplicationWorld) {
    committed_extrusion(world);
    super::viewport_interaction::click_text_native(world, "Fillet");
    world.expect_text("Select Edges");
    assert_eq!(world.state().selected_edge_count, 0);
    assert!(!world.state().extrude_preview_active);
}

#[given("its required eligible model entities are available in the viewport")]
fn solid_is_available(world: &mut ApplicationWorld) {
    assert_eq!(world.state().browser.unwrap().bodies.len(), 1);
}

#[when("the user clicks an eligible model entity requested by that field")]
#[when("the user picks the required entities")]
pub(super) fn pick_fillet_edge(world: &mut ApplicationWorld) {
    let point = first_top_edge_midpoint(world);
    let pixel = world.app().window.native.view.model_pixel(point).unwrap();
    super::super::x11_input::click(pixel);
    world.pump(Vec::new());
    assert_eq!(
        world.state().selected_edge_count,
        1,
        "edge at {point:?}, pixel {pixel:?}"
    );
}

fn first_top_edge_midpoint(world: &mut ApplicationWorld) -> [f64; 3] {
    let document = world.active_id();
    let session = world
        .app()
        .owner
        .documents()
        .workspace()
        .session(document)
        .unwrap();
    let sketch = session.snapshot().sketches.sketches().first().unwrap();
    let geometry = sketch.geometry();
    let [a, b] = geometry.lines()[0]
        .endpoints
        .map(|id| geometry.point(id).unwrap().position);
    [(a[0] + b[0]) / 2., (a[1] + b[1]) / 2., 1.]
}

#[then("the field identifies the selected entity and its type")]
fn selected_edge_is_identified(world: &mut ApplicationWorld) {
    world.expect_text("1 Edge");
    assert_eq!(world.state().selected_edge_count, 1);
}

pub(in super::super) fn selected_edge_is_visible(world: &mut ApplicationWorld) {
    let point = first_top_edge_midpoint(world);
    let pixel = world.app().window.native.view.model_pixel(point).unwrap();
    let color = world.pixel_at(pixel.map(|value| usize::try_from(value).unwrap()));
    assert!(
        color[1] > color[0] && color[1] > color[2],
        "edge pixel {color:?}"
    );
}

#[then("the feature preview reflects the selected geometry and current parameters")]
fn selected_fillet_is_previewed(world: &mut ApplicationWorld) {
    assert!(world.state().extrude_preview_active);
    world.expect_text("2 mm");
    assert_eq!(feature_count(world), 1);
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::super::diagnostics::window_occlusions();
    assert!(super::super::solid_visibility::visible_preview_fill(
        &capture, &windows
    ));
    change_fillet_radius(world, "3");
    let updated = super::super::capture_framebuffer(&mut world.app().window.native);
    assert!(
        super::viewport_interaction::preview_changed_outside_windows(&capture, &updated, &windows)
    );
    assert!(world.state().extrude_preview_active);
}

fn change_fillet_radius(world: &mut ApplicationWorld, radius: &str) {
    let label = world.run_in_window("Fillet", "Radius").clone();
    let position = [label.minimum[0] as i32 - 30, label.center()[1] as i32];
    super::super::x11_input::click(position);
    world.pump(Vec::new());
    super::super::x11_input::press_control_a();
    super::super::x11_input::type_digits(radius);
    world.pump(Vec::new());
    world.expect_text(radius);
}

#[then("the tool's confirmation state reflects whether its required inputs are complete")]
fn fillet_can_commit(world: &mut ApplicationWorld) {
    let state = world.state();
    let (generation, edges) = state.fillet_selection.clone().unwrap();
    let action = oblikovati_gui::GuiShellAction::CreateFillet(oblikovati_gui::GuiFilletRequest {
        document: state.document.unwrap(),
        generation,
        edges,
        radius: "3 mm".into(),
    });
    assert!(action.enabled(&state));
}

#[given("a solid feature tool displays a valid preview after viewport selection")]
fn fillet_preview_is_ready(world: &mut ApplicationWorld) {
    fillet_waits_for_edge(world);
    pick_fillet_edge(world);
    selected_fillet_is_previewed(world);
}

#[when("the user confirms the feature")]
fn confirm_fillet(world: &mut ApplicationWorld) {
    super::viewport_interaction::click_text_native(world, "OK");
}

#[then("the preview is committed to the model")]
fn fillet_is_committed(world: &mut ApplicationWorld) {
    assert!(!world.state().extrude_preview_active);
    assert_eq!(feature_count(world), 2);
}

#[then("a feature entry appears in the model browser")]
fn fillet_is_in_browser(world: &mut ApplicationWorld) {
    let browser = world.state().browser.unwrap();
    assert!(
        browser
            .features
            .iter()
            .any(|feature| feature.name == "Fillet1" && feature.kind == "Fillet")
    );
    world.expect_text("Fillet1");
}
