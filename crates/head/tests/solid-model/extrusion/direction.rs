// SPDX-License-Identifier: GPL-2.0-only
//! Compare native direction drafts and verify the accepted geometry.
use super::acceptance::feature_count;
use super::viewport_interaction::click_text_native;
use super::*;

#[given("the rectangle extrusion preview is open at Default direction")]
fn default_direction(world: &mut ApplicationWorld) {
    super::selected_rectangle_preview(world);
    default_preview_points_forward(world);
}

#[given(
    "a copy of `fixtures/cad/revolve/revolve-profile.opd` is open with an active rectangular Extrude preview at Default direction"
)]
fn published_default_direction(world: &mut ApplicationWorld) {
    super::reference_extent::native_fixture_preview(world);
    default_preview_points_forward(world);
}

fn default_preview_points_forward(world: &mut ApplicationWorld) {
    world.expect_text("Direction");
    assert_eq!(preview_depth(world), [0., 1.]);
}

#[when("Flipped is selected in Direction")]
fn choose_flipped(world: &mut ApplicationWorld) {
    choose_direction_icon(world, 1);
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
    let icons = direction_icons(world);
    assert_eq!(icons.len(), 4);
    let (low, high) = icons[index];
    let center = std::array::from_fn(|axis| (low[axis] + high[axis]) / 2.);
    super::viewport_interaction::click_at(world, super::super::diagnostics::pointer_pixels(center));
}

pub(super) fn direction_icons(world: &ApplicationWorld) -> Vec<([f32; 2], [f32; 2])> {
    use super::super::{diagnostics, ribbon_appearance::icon_quads};
    let ([left, _], [width, _]) = diagnostics::window_geometry("Extrude").unwrap();
    if world.window_text("Extrude").contains("Default") {
        let y = world.run_in_window("Extrude", "Default").center()[1];
        return icon_quads::icon_quads_inside(([left, y - 28.], [left + width, y - 4.]));
    }
    let y = world.run_in_window("Extrude", "Direction").center()[1];
    let mut icons = icon_quads::icon_quads_inside(([left, y + 7.], [left + width, y + 47.]));
    if let Some(top) = icons.iter().map(|(low, _)| low[1]).min_by(f32::total_cmp) {
        icons.retain(|(low, _)| (low[1] - top).abs() < 1.);
    }
    icons
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
    let row = second_distance_row(world);
    let position =
        super::super::diagnostics::pointer_pixels([row.minimum[0] - 30., row.center()[1]]);
    world.click(position);
    world.key(ViewportPhysicalKey::A, true);
    world.type_text(value);
    world.pump(Vec::new());
}

fn second_distance_row(world: &ApplicationWorld) -> &super::super::diagnostics::RenderedRun {
    // Depending on scale, the text reader can group the label with the adjacent button.
    let label = if world.frame.runs.iter().any(|run| run.text == "DistanceB") {
        "DistanceB"
    } else {
        "DistanceBFlipA/B"
    };
    world.run_in_window("Extrude", label)
}

#[then("the preview changes its side or extent relative to Default direction")]
fn preview_reaches_the_opposite_side(world: &mut ApplicationWorld) {
    assert!(world.state().extrude_preview_active);
    assert_eq!(feature_count(world), 0);
    world.expect_text("10.00 mm");
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::super::diagnostics::window_occlusions();
    assert!(matches!(preview_depth(world), [-1., 0.] | [-0.5, 1.]));
    assert!(super::super::solid_visibility::visible_preview_fill(
        &capture, &windows
    ));
}

#[test]
fn flipped_literal_preview_and_commit_use_the_same_direction() {
    use oblikovati_api::{client::Client, wire::BodyPhysicalPropertiesArgs};
    let mut world = ApplicationWorld::new();
    default_direction(&mut world);
    choose_flipped(&mut world);
    preview_reaches_the_opposite_side(&mut world);
    assert_eq!(preview_depth(&mut world), [-1., 0.]);
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

// Use the published rectangular part for every row of the direction outline.
#[given(
    "a fresh copy of `fixtures/cad/revolve/revolve-profile.opd` is open in Extrude with its rectangle selected"
)]
fn open_for_measured_direction(world: &mut ApplicationWorld) {
    super::reference_extent::native_fixture_preview(world);
}

#[given("the start reference is the sketch plane with both tapers zero")]
fn untapered_sketch_plane_start(world: &mut ApplicationWorld) {
    world.run_in_window("Extrude", "SketchPlane");
    assert_eq!(preview_depth(world), [0., 1.]);
    assert_matching_caps(world, [0., 1.]);
}

#[when(regex = r"^the user chooses (Default|Flipped|Symmetric) and sets Distance A to (\d+) mm$")]
fn set_measured_direction(world: &mut ApplicationWorld, direction: String, a: String) {
    let index = ["Default", "Flipped", "Symmetric"]
        .iter()
        .position(|label| *label == direction)
        .unwrap();
    choose_direction_icon(world, index);
    super::replace_extrusion_distance(world, &a);
}

#[when("the user chooses Asymmetric and sets Distance A to 8 mm and B to 4 mm")]
fn set_measured_asymmetric(world: &mut ApplicationWorld) {
    choose_asymmetric(world);
    super::replace_extrusion_distance(world, "8");
    replace_second_distance(world, "4");
}

#[then(
    regex = r"^the extrusion preview extends (8 mm along the default direction|8 mm opposite the default direction|4 mm on each side of the sketch plane|8 mm on the default side and 4 mm on the opposite side)$"
)]
fn check_measured_extent(world: &mut ApplicationWorld, extent: String) {
    let expected = match extent.as_str() {
        "8 mm along the default direction" => [0., 0.8],
        "8 mm opposite the default direction" => [-0.8, 0.],
        "4 mm on each side of the sketch plane" => [-0.4, 0.4],
        "8 mm on the default side and 4 mm on the opposite side" => [-0.4, 0.8],
        _ => unreachable!("unrecognized extrusion extent {extent:?}"),
    };
    assert_eq!(preview_depth(world), expected);
    assert_matching_caps(world, expected);
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::super::diagnostics::window_occlusions();
    assert!(super::super::solid_visibility::visible_preview_fill(
        &capture, &windows
    ));
}

#[then("the command is ready to accept")]
fn measured_draft_is_ready(world: &mut ApplicationWorld) {
    assert!(world.state().extrude_preview_active);
    assert!(world.run_is_enabled("OK"));
    assert_eq!(feature_count(world), 0);
}

fn assert_matching_caps(world: &mut ApplicationWorld, limits: [f32; 2]) {
    let scene = &world.app().window.scene_triangles;
    let bounds = |depth| {
        scene
            .iter()
            .flat_map(|triangle| triangle.positions)
            .filter(|point| point[2] == depth)
            .fold(
                [
                    f32::INFINITY,
                    f32::INFINITY,
                    f32::NEG_INFINITY,
                    f32::NEG_INFINITY,
                ],
                |[min_x, min_y, max_x, max_y], [x, y, _]| {
                    [min_x.min(x), min_y.min(y), max_x.max(x), max_y.max(y)]
                },
            )
    };
    let first = bounds(limits[0]);
    assert!(first[0] < first[2] && first[1] < first[3]);
    assert_eq!(first, bounds(limits[1]), "expected parallel walls");
}
