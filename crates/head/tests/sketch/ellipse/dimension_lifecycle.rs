// SPDX-License-Identifier: GPL-2.0-only
//! The recorded ellipse minor-radius dimension and its cancelled value edit.
use super::super::super::{ApplicationWorld, diagnostics};
use super::choose_ellipse;
use cucumber::{given, then, when};
use oblikovati_api::{
    client::Client,
    types::{DIM_CONSTRAINT_ELLIPSE_MINOR_RADIUS, DIM_CONSTRAINT_ELLIPSE_RADIUS},
    wire::AddDimensionArgs,
};
use oblikovati_viewport::ViewportPhysicalKey;

const CENTER: [f64; 2] = [-6.213, 15.221];
const MINOR: f64 = 2.9788523;
#[given(
    "the recorded ellipse has major-radius annotation 8 and minor radius approximately 2.979 mm"
)]
fn recorded_ellipse(world: &mut ApplicationWorld) {
    world.start_sketch_on("YZ Plane");
    create_recorded_ellipse(world);
}
fn create_recorded_ellipse(world: &mut ApplicationWorld) {
    choose_ellipse(world);
    world.open_precise_input();
    for point in [
        CENTER,
        [CENTER[0] + 8., CENTER[1]],
        [CENTER[0], CENTER[1] + MINOR],
    ] {
        world.precise_point(point);
    }
    world.escape();
    let geometry = world.edited_geometry();
    let ellipse = geometry.ellipses()[0];
    Client::new(world.app().owner.document_host())
        .dimension()
        .add(&AddDimensionArgs {
            kind: DIM_CONSTRAINT_ELLIPSE_RADIUS.0.to_string(),
            entities: Some(vec![ellipse.id().0]),
            expression: "8 mm".into(),
            text_point: Some(vec![CENTER[0], CENTER[1] - 12.]),
            ..Default::default()
        })
        .unwrap();
    world.pump(Vec::new());
    world.expect_annotation("8");
}
#[when("General Dimension selects its curve and places the vertical center-to-axis dimension")]
fn place_minor(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::D, false);
    let geometry = world.edited_geometry();
    let ellipse = geometry.ellipses()[0];
    let center = geometry.point(ellipse.center()).unwrap().position;
    let hit = ellipse.evaluate(center, 0.7).unwrap().position;
    let hit = world.sketch_pixel_of(hit);
    world.sketch_click(hit);
    let label = world.sketch_pixel_of([center[0] + 1.2, center[1]]);
    world.sketch_click(label);
    world.pump(Vec::new());
}
#[then("the minor-radius editor opens with 2.9788523 mm selected")]
fn editor_shows_minor(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Edit Dimension").is_some());
    assert!(world.window_text("Edit Dimension").contains("2.9788523"));
    assert!(world.text_is_selected("2.9788523"));
}
#[when("4 is entered and Escape cancels the editor")]
fn cancel_value(world: &mut ApplicationWorld) {
    world.type_text("4");
    world.escape();
}
#[then("the placed dimension remains annotated 2.979")]
fn annotation_retained(world: &mut ApplicationWorld) {
    world.expect_annotation("2.979");
    assert!(world.state().dimension_tool_active);
}
#[when("its annotation is reopened")]
fn reopen(world: &mut ApplicationWorld) {
    world.click(world.exact_run_center("2.979"));
    world.pump(Vec::new());
}
#[then("the editor retains expression 2.9788523 mm")]
fn retained_expression(world: &mut ApplicationWorld) {
    assert!(world.window_text("Edit Dimension").contains("2.9788523"));
}
#[given("the reopened minor-radius expression is selected")]
fn reopened_minor(world: &mut ApplicationWorld) {
    recorded_ellipse(world);
    place_minor(world);
    editor_shows_minor(world);
    cancel_value(world);
    reopen(world);
    retained_expression(world);
    assert!(world.text_is_selected("2.9788523"));
}
#[when("4 is entered and Return accepts it")]
fn accept_minor(world: &mut ApplicationWorld) {
    world.type_text("4");
    world.key(ViewportPhysicalKey::Enter, false);
}
#[then("the ellipse grows vertically and its vertical annotation becomes 4")]
fn minor_changed(world: &mut ApplicationWorld) {
    world.expect_annotation("4");
    let ellipse = world.edited_geometry().ellipses()[0];
    assert!((ellipse.radii()[1] - 0.4).abs() < 1e-8);
}
#[then("its horizontal annotation remains 8")]
fn major_retained(world: &mut ApplicationWorld) {
    world.expect_annotation("8");
    let ellipse = world.edited_geometry().ellipses()[0];
    assert!((ellipse.radii()[0] - 0.8).abs() < 1e-8);
}
#[when("Escape exits General Dimension")]
fn exit_dimension(world: &mut ApplicationWorld) {
    world.escape();
    assert!(!world.state().dimension_tool_active);
}
#[then("the edited ellipse remains visible")]
fn ellipse_remains(world: &mut ApplicationWorld) {
    assert_eq!(world.edited_geometry().ellipses().len(), 1);
    world.expect_annotation("4");
    world.expect_annotation("8");
}

#[given("the recorded ellipse has a driving minor-radius dimension storing 4 mm")]
fn dimensioned_ellipse(world: &mut ApplicationWorld) {
    if world.app().owner.active_sketch_edit().is_none() {
        recorded_ellipse(world);
        add_minor_dimension(world);
    }
    ellipse_remains(world);
}
fn add_minor_dimension(world: &mut ApplicationWorld) {
    let ellipse = world.edited_geometry().ellipses()[0];
    let label_position = world.sketch_point_of([950, 600]);
    Client::new(world.app().owner.document_host())
        .dimension()
        .add(&AddDimensionArgs {
            kind: DIM_CONSTRAINT_ELLIPSE_MINOR_RADIUS.0.to_string(),
            entities: Some(vec![ellipse.id().0]),
            expression: "4 mm".into(),
            text_point: Some(label_position.to_vec()),
            ..Default::default()
        })
        .unwrap();
    world.pump(Vec::new());
    world.expect_annotation("4");
}
#[when(
    "General Dimension opens the minor-radius dimension and 0 followed by Return attempts an edit"
)]
fn reject_zero(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::D, false);
    let label = world.exact_run_center("4");
    world.click(label);
    assert!(
        diagnostics::window_geometry("Edit Dimension").is_some(),
        "minor editor did not open at {label:?}: {:?}; annotations: {:?}",
        world.status_text(),
        crate::HeadSketchWork::sketch_dimension_annotations(&world.app().owner).unwrap()
    );
    world.type_text("0");
    world.key(ViewportPhysicalKey::Enter, false);
    let refusal = world.window_text("Edit Dimension Value");
    assert!(refusal.contains("positive"), "refusal text: {refusal:?}");
}
#[when("Cancel dismisses the failure window")]
fn dismiss_refusal(world: &mut ApplicationWorld) {
    world.click_text("Cancel");
}
#[then("General Dimension remains active with annotations 8 and 4 visible")]
fn dimensions_remain(world: &mut ApplicationWorld) {
    assert!(world.state().dimension_tool_active);
    world.expect_annotation("8");
    world.expect_annotation("4");
    let ellipse = world.edited_geometry().ellipses()[0];
    assert!((ellipse.radii()[1] - 0.4).abs() < 1e-8);
}
#[when("the minor-radius dimension is reopened")]
fn reopen_minor(world: &mut ApplicationWorld) {
    world.click(world.exact_run_center("4"));
}
#[then("its stored expression is 4 mm")]
fn retained_minor_value(world: &mut ApplicationWorld) {
    let text = world.window_text("Edit Dimension");
    assert!(text.contains("4mm"), "editor text: {text:?}");
}
#[when("Escape closes the editor and another Escape exits the command")]
fn close_dimension_command(world: &mut ApplicationWorld) {
    world.escape();
    world.escape();
}
#[then("Ready appears and the ellipse remains visible")]
fn ready_with_ellipse(world: &mut ApplicationWorld) {
    world.expect_ready();
    ellipse_remains(world);
}

#[given("the fixture sketch contains the ellipse with annotations 8 and 4")]
fn visible_dimensioned_ellipse(world: &mut ApplicationWorld) {
    if world.app().owner.active_sketch_edit().is_some() {
        ellipse_remains(world);
        return;
    }
    world.add_part();
    super::fixture::show_home_view(world);
    world.click_text("Start 2D Sketch");
    world.pump(Vec::new());
    world.click_text("YZ Plane");
    world.pump(Vec::new());
    create_recorded_ellipse(world);
    add_minor_dimension(world);
}
#[when("Finish Sketch returns to 3D Model")]
fn finish_dimensioned_sketch(world: &mut ApplicationWorld) {
    world.finish_sketch();
    world.pump(Vec::new());
}
#[then("the ellipse and its dimension annotations remain visible in the oblique view")]
fn dimensions_after_finish(world: &mut ApplicationWorld) {
    world.expect_annotation("8");
    world.expect_annotation("4");
    let labels = crate::HeadSketchWork::sketch_dimension_annotations(&world.app().owner).unwrap();
    assert_eq!(labels.labels.len(), 2);
    let scene = world.app().window.native.sketch_scene.as_ref().unwrap();
    assert_eq!(scene.sketches.len(), 1);
    assert_eq!(scene.sketches[0].geometry.ellipses().len(), 1);
    let camera = crate::HeadWindow::view_camera(&world.app().window).unwrap();
    let direction = std::array::from_fn::<_, 3, _>(|axis| camera.eye[axis] - camera.target[axis]);
    assert!(
        direction.iter().all(|value| value.abs() > 1e-8),
        "{direction:?}"
    );
    assert_rendered_ellipse(world);
}
fn assert_rendered_ellipse(world: &mut ApplicationWorld) {
    let sketches = world
        .app()
        .window
        .native
        .sketch_scene
        .as_ref()
        .unwrap()
        .sketches
        .clone();
    let points = ellipse_pixels(world, &sketches[0]);
    let native = &mut world.app().window.native;
    let visible = super::super::super::capture_framebuffer(native);
    let hidden = world.canvas_without_sketches();
    let changed = points
        .iter()
        .filter(|point| pixels_differ_near(&visible, &hidden, **point))
        .count();
    assert!(
        changed >= 4,
        "ellipse samples with rendered pixels: {changed} of {}",
        points.len()
    );
}
fn ellipse_pixels(
    world: &mut ApplicationWorld,
    sketch: &oblikovati_viewport::ViewportSketchDisplay,
) -> Vec<[i32; 2]> {
    let ellipse = sketch.geometry.ellipses()[0];
    let center = sketch.geometry.point(ellipse.center()).unwrap().position;
    (0..32)
        .map(|index| {
            let parameter = std::f64::consts::TAU * f64::from(index) / 32.;
            let point = ellipse.evaluate(center, parameter).unwrap().position;
            crate::HeadWindow::sketch_pixel(&world.app().window, sketch.plane, point).unwrap()
        })
        .collect()
}
fn pixels_differ_near(
    visible: &oblikovati_viewport::ViewportCapture,
    hidden: &oblikovati_viewport::ViewportCapture,
    point: [i32; 2],
) -> bool {
    let [width, height] = visible.extent.map(|side| side as i32);
    (-3..=3).any(|dy| {
        (-3..=3).any(|dx| {
            let [x, y] = [point[0] + dx, point[1] + dy];
            if x < 0 || y < 0 || x >= width || y >= height {
                return false;
            }
            let offset = ((y * width + x) * 4) as usize;
            visible.rgba[offset..offset + 3] != hidden.rgba[offset..offset + 3]
        })
    })
}
#[when("the sketch is right-clicked in the browser")]
fn open_sketch_menu(world: &mut ApplicationWorld) {
    let row = world
        .frame
        .runs
        .iter()
        .find(|run| run.text.starts_with("[DOF"))
        .expect("expected the finished sketch row")
        .center()
        .map(|coordinate| coordinate as i32);
    world.right_click(row);
    world.pump(Vec::new());
}
#[then("Edit Sketch is available and Visibility and Dimension Visibility are checked")]
fn sketch_menu_state(world: &mut ApplicationWorld) {
    assert!(world.shows("Edit Sketch"));
    assert!(world.checked("Visibility"));
    assert!(world.checked("Dimension Visibility"));
}
#[when("Edit Sketch is selected")]
fn reopen_dimensioned_sketch(world: &mut ApplicationWorld) {
    world.click_text("Edit Sketch");
    world.pump(Vec::new());
}
#[then("the editable sketch returns in Right view with the ellipse and annotations 8 and 4")]
fn ellipse_after_reopen(world: &mut ApplicationWorld) {
    ellipse_remains(world);
    assert!(world.app().owner.active_sketch_edit().is_some());
    let camera = crate::HeadWindow::view_camera(&world.app().window).unwrap();
    let direction = std::array::from_fn::<_, 3, _>(|axis| camera.eye[axis] - camera.target[axis]);
    assert!(
        direction[0] > 0. && direction[1].abs() < 1e-8 && direction[2].abs() < 1e-8,
        "{direction:?}"
    );
}
