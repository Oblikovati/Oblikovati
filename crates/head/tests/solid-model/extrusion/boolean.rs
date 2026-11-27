// SPDX-License-Identifier: GPL-2.0-only
//! Native one-body Extrude Join, including its real replacement preview and history.
use super::super::{capture_framebuffer, diagnostics, solid_visibility};
use super::*;

const FIXTURE: &str = "fixtures/cad/extrusion/extrusion-base-circle.opd";
const BASE_TOP: f32 = 0.8;
const BOSS_TOP: f32 = 1.6;

#[given(
    "a copy of `fixtures/cad/extrusion/extrusion-base-circle.opd` is open with its diameter-3 mm circle sketch on the top face"
)]
#[given(
    "a copy of `fixtures/cad/extrusion/extrusion-base-circle.opd` is open with its circular sketch on the top plane and one existing solid"
)]
#[given(
    "a copy of `fixtures/cad/extrusion/extrusion-base-circle.opd` is open with its diameter-3 mm circle sketch on the base top face"
)]
#[given(
    "a copy of `fixtures/cad/extrusion/extrusion-base-circle.opd` is open with its base solid and circular sketch on the top plane"
)]
fn published_circle_part(world: &mut ApplicationWorld) {
    let source = super::super::scenario_root().join(FIXTURE);
    open_circle_part(world, &std::fs::read(source).unwrap());
}

#[given("a diameter-3 circular sketch on the top face of the existing solid")]
#[given("the restored circular Sketch2 and existing Solid1")]
#[given("the diameter-3 circle on the base top face")]
#[given("the base and restored circular Sketch2")]
fn pinned_circle_part(world: &mut ApplicationWorld) {
    open_circle_part(
        world,
        include_bytes!(
            "../../../../../test-utilities/fixtures/extrusion/extrusion-base-circle.opd"
        ),
    );
}

fn open_circle_part(world: &mut ApplicationWorld, package: &[u8]) {
    let path = world.scratch_folder().join("extrusion-base-circle.opd");
    std::fs::write(&path, package).unwrap();
    world.open_fixture(&path);
    assert_original_shape(world);
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        oblikovati_viewport::ViewportNavigation::Home { margin: 0.12 },
    )
    .unwrap();
    world.pump(Vec::new());
}

#[when("Extrude uses Default direction, Distance A 8 mm, zero taper and Join")]
fn preview_joined_boss(world: &mut ApplicationWorld) {
    let generation = world.state().document_generation;
    let original = world.state().browser.unwrap().bodies[0].key.clone();
    start_circle_extrude(world);
    click_panel(world, "NewSolid");
    world.click_text("Join");
    world.pump(Vec::new());
    let state = world.state();
    let draft = state.extrude_draft_preview.expect("certified Join preview");
    assert_eq!(
        draft.operation,
        Some(oblikovati_model::feature::sketch_prism::PrismBoolean::Join)
    );
    assert_eq!(
        draft.direction,
        oblikovati_gui::GuiExtrudeDirection::Positive
    );
    assert_eq!(state.document_generation, generation);
    assert_eq!(state.browser.unwrap().bodies[0].key, original);
    assert!(world.run_is_enabled("OK"));
    assert_circular_cap(world);
    assert_visible_replacement(world);
}

#[then("a cylindrical boss and Extrusion2 are visible")]
fn joined_result(world: &mut ApplicationWorld) {
    world.expect_text("Extrusion2");
    assert!(diagnostics::window_geometry("Extrude").is_none());
    let browser = world.state().browser.unwrap();
    let joined = browser
        .features
        .iter()
        .find(|row| row.name == "Extrusion2")
        .unwrap();
    assert_eq!(browser.bodies[0].feature, joined.id);
    assert_eq!(browser.bodies[0].faces, 8);
    assert!(browser.sketches.iter().any(|sketch| {
        sketch.name == "Sketch1" && !sketch.visible && sketch.consumed_by == Some(joined.id)
    }));
    assert_circular_cap(world);
    let frame = capture_framebuffer(&mut world.app().window.native);
    assert!(solid_visibility::visible_solid_fill(
        &frame,
        &diagnostics::window_occlusions()
    ));
}

#[then("the browser still shows Solid Bodies(1)")]
fn still_one_body(world: &mut ApplicationWorld) {
    world.expect_text("Solid Bodies");
    assert_eq!(world.state().browser.unwrap().bodies.len(), 1);
}

#[then("the circular sketch returns and the boss disappears")]
#[then("Sketch2 returns and the boss disappears")]
fn assert_original_shape(world: &mut ApplicationWorld) {
    still_one_body(world);
    let browser = world.state().browser.unwrap();
    assert_eq!(browser.bodies[0].faces, 6);
    assert!(browser.features.iter().all(|row| row.name != "Extrusion2"));
    // The published native package calls the second sketch Sketch1.
    assert!(browser.sketches.iter().any(|sketch| {
        sketch.name == "Sketch1" && sketch.visible && sketch.consumed_by.is_none()
    }));
    let top = world
        .app()
        .window
        .scene_triangles
        .iter()
        .flat_map(|triangle| triangle.positions.map(|point| point[2]))
        .fold(f32::NEG_INFINITY, f32::max);
    assert!((top - BASE_TOP).abs() < 1e-5, "base cap is at {top} cm");
}

fn click_panel(world: &mut ApplicationWorld, label: &str) {
    let point = diagnostics::pointer_pixels(world.run_in_window("Extrude", label).center());
    world.click(point);
}

fn assert_circular_cap(world: &mut ApplicationWorld) {
    let mut min = [f32::INFINITY; 2];
    let mut max = [f32::NEG_INFINITY; 2];
    for point in world
        .app()
        .window
        .scene_triangles
        .iter()
        .flat_map(|triangle| triangle.positions)
        .filter(|point| (point[2] - BOSS_TOP).abs() < 1e-5)
    {
        for axis in 0..2 {
            min[axis] = min[axis].min(point[axis]);
            max[axis] = max[axis].max(point[axis]);
        }
    }
    for axis in 0..2 {
        let span = max[axis] - min[axis];
        assert!(
            (span - 0.3).abs() < 0.01,
            "boss cap {axis} span is {span} cm at {BOSS_TOP} cm"
        );
    }
}

#[when("Extrude uses New Solid with Distance A 8 mm and zero taper")]
fn preview_separate_boss(world: &mut ApplicationWorld) {
    start_circle_extrude(world);
    let state = world.state();
    assert!(state.extrude_draft_preview.unwrap().operation.is_none());
    assert_eq!(state.browser.unwrap().bodies.len(), 1);
    assert_circular_cap(world);
}

fn start_circle_extrude(world: &mut ApplicationWorld) {
    world.click_text("Sketch1");
    world.click_text("Extrude");
    super::replace_extrusion_distance(world, "8");
    world.pump(Vec::new());
}

#[when("Body Name is replaced with b and OK is clicked")]
fn accept_named_solid(world: &mut ApplicationWorld) {
    let original = world.state().browser.unwrap().bodies[0].key.clone();
    click_panel(world, "Solid2");
    world.type_text("b");
    world.pump(Vec::new());
    world.run_in_window("Extrude", "b");
    assert!(world.run_is_enabled("OK"));
    click_panel(world, "OK");
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry("Extrude").is_none());
    let browser = world.state().browser.unwrap();
    assert_eq!(browser.bodies[0].key, original);
    assert_eq!(browser.bodies[0].faces, 6);
    assert_eq!(browser.bodies[1].faces, 3);
    assert_circular_cap(world);
}

#[then("the browser shows Solid Bodies(2)")]
fn two_solid_bodies(world: &mut ApplicationWorld) {
    world.expect_text("Solid Bodies");
    assert_eq!(world.state().browser.unwrap().bodies.len(), 2);
}

#[then("expanding it shows the original body and b")]
#[then("expanding it shows Solid1 and b")]
fn solid_names_and_history(world: &mut ApplicationWorld) {
    if !world.text().contains("Solid1") {
        world.click_text("Solid Bodies");
    }
    world.expect_text("Solid1");
    world.expect_text("b");
    assert_solid_names(world);
    world.key(ViewportPhysicalKey::Z, true);
    world.pump(Vec::new());
    assert_original_shape(world);
    world.key(ViewportPhysicalKey::Y, true);
    world.pump(Vec::new());
    assert_solid_names(world);
    assert_circular_cap(world);
}

fn assert_solid_names(world: &mut ApplicationWorld) {
    let browser = world.state().browser.unwrap();
    let names: Vec<_> = browser
        .bodies
        .iter()
        .map(|body| body.name.as_str())
        .collect();
    assert_eq!(names, ["Solid1", "b"]);
}

#[when("Extrude opens in Default direction and Cut is selected")]
fn choose_cut(world: &mut ApplicationWorld) {
    world.click_text("Sketch1");
    world.click_text("Extrude");
    world.pump(Vec::new());
    assert_eq!(
        world.state().extrude_draft_preview.unwrap().direction,
        oblikovati_gui::GuiExtrudeDirection::Positive
    );
    let generation = world.state().document_generation;
    click_panel(world, "NewSolid");
    world.click_text("Cut");
    world.pump(Vec::new());
    assert_eq!(world.state().document_generation, generation);
}

#[then("Direction changes to Flipped and the preview shows an opening")]
fn flipped_cut_preview(world: &mut ApplicationWorld) {
    let draft = world
        .state()
        .extrude_draft_preview
        .expect("certified Cut preview");
    assert_eq!(
        draft.operation,
        Some(oblikovati_model::feature::sketch_prism::PrismBoolean::Cut)
    );
    assert_eq!(
        draft.direction,
        oblikovati_gui::GuiExtrudeDirection::Negative
    );
    assert!(world.run_is_enabled("OK"));
    assert_visible_replacement(world);
    assert_top_opening(world, true);
}

#[when("OK is clicked with Distance A 8 mm and zero taper")]
fn accept_eight_millimeter_draft(world: &mut ApplicationWorld) {
    super::replace_extrusion_distance(world, "8");
    world.pump(Vec::new());
    assert!(world.run_is_enabled("OK"));
    click_panel(world, "OK");
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry("Extrude").is_none());
}

#[then("Extrusion2 and the opening are retained in one solid body")]
#[then("the opening disappears then returns")]
fn cut_result(world: &mut ApplicationWorld) {
    still_one_body(world);
    world.expect_text("Extrusion2");
    assert_eq!(world.state().browser.unwrap().bodies[0].faces, 7);
    assert_top_opening(world, true);
}

#[when("Undo then Redo are invoked")]
fn undo_redo_boolean(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::Z, true);
    world.pump(Vec::new());
    assert_original_shape(world);
    assert_top_opening(world, false);
    world.key(ViewportPhysicalKey::Y, true);
    world.pump(Vec::new());
}

fn assert_top_opening(world: &mut ApplicationWorld, opening: bool) {
    let triangles = &world.app().window.scene_triangles;
    let top_faces = triangles.iter().filter(|triangle| {
        triangle
            .positions
            .iter()
            .all(|point| (point[2] - BASE_TOP).abs() < 1e-5)
    });
    let covered = top_faces
        .clone()
        .any(|triangle| contains_xy(triangle.positions, [2., 1.5]));
    assert_eq!(
        !covered, opening,
        "expected top opening {opening} along the circle axis"
    );
    assert!(
        top_faces
            .clone()
            .any(|triangle| contains_xy(triangle.positions, [2.3, 1.5]))
    );
}

fn contains_xy(triangle: [[f32; 3]; 3], point: [f32; 2]) -> bool {
    let signs: [f32; 3] = std::array::from_fn(|index| {
        let a = triangle[index];
        let b = triangle[(index + 1) % 3];
        (b[0] - a[0]) * (point[1] - a[1]) - (b[1] - a[1]) * (point[0] - a[0])
    });
    signs.iter().all(|sign| *sign >= -1e-6) || signs.iter().all(|sign| *sign <= 1e-6)
}

#[when("Intersect is selected in Extrude")]
fn choose_intersect(world: &mut ApplicationWorld) {
    world.click_text("Sketch1");
    world.click_text("Extrude");
    world.pump(Vec::new());
    let generation = world.state().document_generation;
    click_panel(world, "NewSolid");
    world.click_text("Intersect");
    world.pump(Vec::new());
    assert_eq!(world.state().document_generation, generation);
}

#[then("Direction changes to Flipped and a cylindrical region is previewed")]
fn intersect_preview(world: &mut ApplicationWorld) {
    let draft = world
        .state()
        .extrude_draft_preview
        .expect("certified Intersect preview");
    assert_eq!(
        draft.operation,
        Some(oblikovati_model::feature::sketch_prism::PrismBoolean::Intersect)
    );
    assert_eq!(
        draft.direction,
        oblikovati_gui::GuiExtrudeDirection::Negative
    );
    assert!(world.run_is_enabled("OK"));
    assert_retained_cylinder_mesh(world);
    assert_visible_replacement(world);
}

#[then("the browser retains one solid body")]
fn one_intersected_body(world: &mut ApplicationWorld) {
    still_one_body(world);
}

#[then("the cylinder replaces the visible rectangular base result")]
fn retained_cylinder(world: &mut ApplicationWorld) {
    world.expect_text("Extrusion2");
    let browser = world.state().browser.unwrap();
    assert_eq!(browser.bodies[0].faces, 3);
    assert_retained_cylinder_mesh(world);
}

#[then("the base returns then the cylinder returns")]
fn intersect_after_history_and_reopen(world: &mut ApplicationWorld) {
    retained_cylinder(world);
    let key = world.state().browser.unwrap().bodies[0].key.clone();
    let path = world
        .scratch_folder()
        .join("extrusion-intersection-accepted.opd");
    super::acceptance::save_and_reopen(world, &path);
    world.pump(Vec::new());
    retained_cylinder(world);
    assert_eq!(world.state().browser.unwrap().bodies[0].key, key);
}

fn assert_retained_cylinder_mesh(world: &mut ApplicationWorld) {
    let mut min = [f32::INFINITY; 3];
    let mut max = [f32::NEG_INFINITY; 3];
    for point in world
        .app()
        .window
        .scene_triangles
        .iter()
        .flat_map(|triangle| triangle.positions)
    {
        for axis in 0..3 {
            min[axis] = min[axis].min(point[axis]);
            max[axis] = max[axis].max(point[axis]);
        }
    }
    for (actual, expected) in min
        .into_iter()
        .chain(max)
        .zip([1.85, 1.35, 0., 2.15, 1.65, 0.8])
    {
        assert!(
            (actual - expected).abs() < 0.01,
            "intersection bound {actual} instead of {expected} cm"
        );
    }
}

fn assert_visible_replacement(world: &mut ApplicationWorld) {
    let frame = capture_framebuffer(&mut world.app().window.native);
    assert!(solid_visibility::visible_replacement_preview_fill(
        &frame,
        &diagnostics::window_occlusions()
    ));
}

#[given(
    "a copy of `fixtures/cad/extrusion/extrusion-base-circle.opd` is open and the circle has been intersected with the base solid, then restored by Redo"
)]
fn published_intersection_for_measure(world: &mut ApplicationWorld) {
    published_circle_part(world);
    intersect_for_measure(world);
}

#[given("the accepted Intersect result after Redo")]
fn pinned_intersection_for_measure(world: &mut ApplicationWorld) {
    pinned_circle_part(world);
    intersect_for_measure(world);
}

fn intersect_for_measure(world: &mut ApplicationWorld) {
    choose_intersect(world);
    accept_eight_millimeter_draft(world);
    undo_redo_boolean(world);
    retained_cylinder(world);
}

#[when("its top face and cylindrical face are selected in Measure")]
fn measure_intersection_faces(world: &mut ApplicationWorld) {
    world.click_text("Tools");
    world.click_text("Measure");
    let mut camera = oblikovati_viewport::ViewportCamera {
        eye: [2., 1.5, 20.4],
        target: [2., 1.5, 0.4],
        scale: 1.1,
        ..Default::default()
    };
    pick_face_in_measure(world, camera, [2., 1.5, 0.8]);
    world.observed_rows = Some(world.window_text("Measure"));
    camera.eye = [2., -18.5, 0.4];
    camera.up = [0., 0., 1.];
    pick_face_in_measure(world, camera, [2., 1.35, 0.4]);
}

fn pick_face_in_measure(
    world: &mut ApplicationWorld,
    camera: oblikovati_viewport::ViewportCamera,
    point: [f64; 3],
) {
    crate::HeadWindow::set_view_camera(&mut world.app().window, camera).unwrap();
    world.pump(Vec::new());
    let pixel = world.app().window.native.view.model_pixel(point).unwrap();
    super::viewport_interaction::click_at(world, pixel);
    world.pump(Vec::new());
    world.expect_text("Selection 1: Face");
}

#[then("the top face area is 7.069 square mm")]
fn circular_cap_area(world: &mut ApplicationWorld) {
    let text = world.observed_rows.as_deref().unwrap();
    assert!(
        text.contains("Area7.069mm²"),
        "expected the measured cap area, found {text:?}"
    );
}

#[then("the cylindrical face diameter is 3.000 mm and radius 1.500 mm")]
fn cylindrical_face_dimensions(world: &mut ApplicationWorld) {
    let text = world.window_text("Measure");
    assert!(
        text.contains("Diameter3.000mm") && text.contains("Radius1.500mm"),
        "expected both cylindrical face dimensions, found {text:?}"
    );
}
