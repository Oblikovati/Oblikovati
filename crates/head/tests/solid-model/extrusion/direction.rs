// SPDX-License-Identifier: GPL-2.0-only
//! Compare native direction drafts and verify the accepted geometry.
use super::acceptance::feature_count;
use super::viewport_interaction::{click_text_native, preview_changed_outside_windows};
use super::*;

#[given("the rectangle extrusion preview is open at Default direction")]
fn default_direction(world: &mut ApplicationWorld) {
    super::selected_rectangle_preview(world);
    world.expect_text("Direction");
    let view = &mut world.app().window.native;
    crate::HeadWindow::navigate_view(
        view,
        oblikovati_viewport::ViewportNavigation::Home { margin: 0.2 },
    )
    .unwrap();
    world.pump(Vec::new());
    world.last_capture = Some(super::super::capture_framebuffer(
        &mut world.app().window.native,
    ));
}

#[when("Flipped is selected in Direction")]
fn choose_flipped(world: &mut ApplicationWorld) {
    click_text_native(world, "Flipped");
    world.pump(Vec::new());
}

#[when("Symmetric is selected in Direction")]
fn choose_symmetric(world: &mut ApplicationWorld) {
    choose_direction_icon(world, 2);
}

#[when("Asymmetric is selected in Direction")]
fn choose_asymmetric(world: &mut ApplicationWorld) {
    choose_direction_icon(world, 3);
}

fn choose_direction_icon(world: &mut ApplicationWorld, index: usize) {
    use super::super::{diagnostics, ribbon_appearance::icon_quads, x11_input};
    let ([left, _], [width, _]) = diagnostics::window_geometry("Extrude").unwrap();
    let y = world.run_in_window("Extrude", "Default").center()[1];
    let icons = icon_quads::icon_quads_inside(([left, y - 28.], [left + width, y - 4.]));
    assert_eq!(icons.len(), 4);
    let (low, high) = icons[index];
    let center = std::array::from_fn(|axis| (low[axis] + high[axis]) / 2.);
    x11_input::click(diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
}

#[then("Distance B and Taper B controls appear")]
fn asymmetric_controls_appear(world: &mut ApplicationWorld) {
    assert!(world.window_text("Extrude").contains("DistanceB"));
    world.run_in_window("Extrude", "TaperB");
    assert_eq!(preview_depth(world), [-0.5, 1.], "5 mm on B and 10 mm on A");
}

#[then("the preview extends on both sides of the source plane")]
fn symmetric_preview_spans_the_sketch_plane(world: &mut ApplicationWorld) {
    assert!(world.state().extrude_preview_active);
    assert_eq!(feature_count(world), 0);
    assert_eq!(
        preview_depth(world),
        [-0.5, 0.5],
        "10 mm total depth about the XY plane"
    );
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::super::diagnostics::window_occlusions();
    assert!(super::super::solid_visibility::visible_preview_fill(
        &capture, &windows
    ));
}

pub(super) fn preview_depth(world: &mut ApplicationWorld) -> [f32; 2] {
    world
        .app()
        .window
        .scene_triangles
        .iter()
        .flat_map(|triangle| triangle.positions)
        .map(|point| point[2])
        .fold([f32::INFINITY, f32::NEG_INFINITY], |[lo, hi], z| {
            [lo.min(z), hi.max(z)]
        })
}

#[test]
fn native_asymmetric_flip_changes_both_distances_and_one_history_step() {
    use oblikovati_api::{client::Client, wire::BodyPhysicalPropertiesArgs};
    let mut world = ApplicationWorld::new();
    default_direction(&mut world);
    choose_asymmetric(&mut world);
    asymmetric_controls_appear(&mut world);
    click_text_native(&mut world, "OK");
    assert_eq!(feature_count(&mut world), 1);
    super::acceptance::open_extrusion_editor(&mut world);
    replace_second_distance(&mut world, "0");
    assert!(!world.state().extrude_preview_active);
    assert!(!world.run_is_enabled("OK"));
    replace_second_distance(&mut world, "6");
    assert_eq!(preview_depth(&mut world), [-0.6, 1.]);
    let row = second_distance_row(&world);
    let flip = [row.maximum[0] - 14., row.center()[1]];
    super::super::x11_input::click(super::super::diagnostics::pointer_pixels(flip));
    world.pump(Vec::new());
    assert_eq!(preview_depth(&mut world), [-1., 0.6]);
    click_text_native(&mut world, "OK");
    let client = Client::new(world.app().owner.document_host());
    let request = BodyPhysicalPropertiesArgs {
        body_index: 0,
        density_g_cm3: 1.,
        accuracy: String::new(),
    };
    let body = client.body().physical_properties(&request).unwrap();
    assert!((body.centroid_z_mm + 2.).abs() < 1e-8);
    client.transactions().undo().unwrap();
    let body = client.body().physical_properties(&request).unwrap();
    assert!((body.centroid_z_mm - 2.5).abs() < 1e-8);
    drop(client);
    world.close();
}

fn replace_second_distance(world: &mut ApplicationWorld, value: &str) {
    use super::super::x11_input;
    let row = second_distance_row(world);
    let position = [row.minimum[0] as i32 - 30, row.center()[1] as i32];
    x11_input::click(position);
    world.pump(Vec::new());
    x11_input::press_control_a();
    x11_input::type_digits(value);
    world.pump(Vec::new());
}

fn second_distance_row(world: &ApplicationWorld) -> &super::super::diagnostics::RenderedRun {
    // The native text reader groups the label and the adjacent Flip A/B caption.
    world.run_in_window("Extrude", "DistanceBFlipA/B")
}

#[then("the preview changes its side or extent relative to Default direction")]
fn preview_is_flipped(world: &mut ApplicationWorld) {
    assert!(world.state().extrude_preview_active);
    assert_eq!(feature_count(world), 0);
    world.expect_text("10.00 mm");
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::super::diagnostics::window_occlusions();
    assert!(preview_changed_outside_windows(
        world.last_capture.as_ref().expect("Default preview"),
        &capture,
        &windows,
    ));
}

#[test]
fn flipped_literal_preview_and_commit_use_the_same_direction() {
    use oblikovati_api::{client::Client, wire::BodyPhysicalPropertiesArgs};
    let mut world = ApplicationWorld::new();
    default_direction(&mut world);
    choose_flipped(&mut world);
    preview_is_flipped(&mut world);
    click_text_native(&mut world, "OK");
    assert_eq!(feature_count(&mut world), 1);
    let body = Client::new(world.app().owner.document_host())
        .body()
        .physical_properties(&BodyPhysicalPropertiesArgs {
            body_index: 0,
            density_g_cm3: 1.,
            accuracy: String::new(),
        })
        .unwrap();
    assert!((body.centroid_z_mm + 5.).abs() < 1e-8);
    world.close();
}
