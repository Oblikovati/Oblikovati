// SPDX-License-Identifier: GPL-2.0-only
//! Exercise midpoint-line selection and placement with native input.
use super::super::ApplicationWorld;
use oblikovati_model::sketch::{PlanarConstraint, Sketch2D};

#[cfg(target_os = "linux")]
fn start_midpoint_line(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    choose_midpoint_line(world);
}

#[cfg(target_os = "linux")]
fn choose_midpoint_line(world: &mut ApplicationWorld) {
    super::super::x11_input::focus_window();
    let caption = world.run("Line").clone();
    let arrow = world
        .frame
        .runs
        .iter()
        .find(|run| {
            run.text == "v"
                && run.minimum[0] > caption.maximum[0]
                && (run.center()[1] - caption.center()[1]).abs() < 4.
        })
        .expect("Line flyout arrow")
        .center();
    super::super::x11_input::click(super::super::diagnostics::pointer_pixels(arrow));
    world.pump(Vec::new());
    let choice = world.run("Midpoint Line").center();
    super::super::x11_input::click(super::super::diagnostics::pointer_pixels(choice));
    world.pump(Vec::new());
    world.expect_status("Select midpoint of line");
}

fn midpoint_positions(geometry: &Sketch2D) -> ([f64; 2], [[f64; 2]; 2]) {
    assert_eq!(geometry.lines().len(), 1);
    let (point, line) = geometry
        .constraints()
        .iter()
        .find_map(|item| match item.relation {
            PlanarConstraint::Midpoint { point, line } => Some((point, line)),
            _ => None,
        })
        .expect("authored midpoint relation");
    let center = geometry.point(point).unwrap().position;
    let endpoints = geometry
        .line(line)
        .unwrap()
        .endpoints
        .map(|id| geometry.point(id).unwrap().position);
    (center, endpoints)
}

#[cfg(target_os = "linux")]
#[test]
fn native_midpoint_line_flyout_places_and_retains_symmetric_line() {
    let mut world = ApplicationWorld::new();
    start_midpoint_line(&mut world);
    super::super::x11_input::click([620, 300]);
    world.pump(Vec::new());
    world.expect_status("Select end of line");
    super::super::x11_input::click([840, 420]);
    world.pump(Vec::new());
    world.expect_status("Select midpoint of line");
    let (center, endpoints) = midpoint_positions(&world.edited_geometry());
    if std::env::var_os("OBK_CAPTURE_MIDPOINT").is_some() {
        world.save_capture(&format!("midpoint-line-{}", std::process::id()));
    }
    for axis in 0..2 {
        assert!((endpoints[0][axis] + endpoints[1][axis] - 2. * center[axis]).abs() < 1e-9);
    }
    super::super::x11_input::press_escape();
    world.pump(Vec::new());
    assert!(world.state().sketch_at_ready);
    super::super::x11_input::press_control_letter(b'z');
    world.pump(Vec::new());
    assert!(world.edited_geometry().lines().is_empty());
    super::super::x11_input::press_control_letter(b'y');
    world.pump(Vec::new());
    let _ = midpoint_positions(&world.edited_geometry());
    world.close();
}

#[cfg(target_os = "linux")]
#[test]
fn native_midpoint_line_accepts_full_length_and_angle_fields() {
    let mut world = ApplicationWorld::new();
    start_midpoint_line(&mut world);
    super::super::x11_input::click([620, 300]);
    world.pump(Vec::new());
    super::super::x11_input::type_digits("10");
    world.pump(Vec::new());
    super::super::x11_input::press_keysym(0xff09);
    world.pump(Vec::new());
    super::super::x11_input::type_digits("30");
    world.pump(Vec::new());
    super::super::x11_input::press_keysym(0xff0d);
    world.pump(Vec::new());
    world.expect_status("Select midpoint of line");
    let (center, endpoints) = midpoint_positions(&world.edited_geometry());
    let delta = [
        endpoints[1][0] - endpoints[0][0],
        endpoints[1][1] - endpoints[0][1],
    ];
    let angle = delta[1].atan2(delta[0]);
    let expected_length = oblikovati_model::units::UnitsOfMeasure::default()
        .from_preferred(10., oblikovati_model::quantity::Dimension::Length)
        .value;
    assert!((delta[0].hypot(delta[1]) - expected_length).abs() < 1e-7);
    assert!((angle - 30_f64.to_radians()).abs() < 1e-7);
    for axis in 0..2 {
        assert!((endpoints[0][axis] + endpoints[1][axis] - 2. * center[axis]).abs() < 1e-7);
    }
    assert_eq!(world.edited_geometry().dimensional_constraints().count(), 2);
    super::super::x11_input::press_escape();
    world.pump(Vec::new());
    world.close();
}

#[cucumber::when("the user invoke Midpoint Line")]
fn invoke_midpoint_line(world: &mut ApplicationWorld) {
    world.choose_from_flyout("Line", "Midpoint Line");
    world.expect_status("Select midpoint of line");
}

#[cucumber::given("an empty sketch in RIGHT view in a metric part document")]
fn empty_right_sketch(world: &mut ApplicationWorld) {
    world.start_sketch_on("YZ Plane");
    super::super::viewport_orientation::expect_control_shows(world, "Right");
    assert!(world.edited_geometry().lines().is_empty());
}

#[cucumber::when("the user invoke Midpoint Line and choose the sketch origin as midpoint")]
fn midpoint_at_origin(world: &mut ApplicationWorld) {
    invoke_midpoint_line(world);
    let origin = world.sketch_pixel_of([0., 0.]);
    world.sketch_click(origin);
    world.expect_status("Select end of line");
}

#[cucumber::when("the user press 1 and then 0 in the length field")]
fn enter_length(world: &mut ApplicationWorld) {
    world.type_text("1");
    world.type_text("0");
}

#[cucumber::then("length 10.000 mm is locked and the angle field is selected")]
fn length_locked(world: &mut ApplicationWorld) {
    let hud = world.state().sketch_hud.expect("midpoint line fields");
    assert_eq!(hud.fields.len(), 2);
    assert!(hud.fields[0].locked);
    assert!(hud.fields[0].text.contains("10.000"));
    assert!(hud.fields[1].active);
}

#[cucumber::when("the user press 3 and then 0 in the angle field")]
fn enter_angle(world: &mut ApplicationWorld) {
    world.type_text("3");
    world.type_text("0");
}

#[cucumber::then("the command requests another midpoint")]
fn asks_another_midpoint(world: &mut ApplicationWorld) {
    world.expect_status("Select midpoint of line");
}

#[cucumber::then("the segment remains with 10 mm and 30.00-degree annotations")]
fn retained_dimensions(world: &mut ApplicationWorld) {
    assert_eq!(world.edited_geometry().lines().len(), 1);
    world.expect_annotation("10");
    world.expect_annotation("30.00");
}

#[cucumber::then("the sketch reports Fully Constrained in this fixture")]
fn fully_constrained(world: &mut ApplicationWorld) {
    assert_eq!(world.dimensions_needed(), 0);
    world.expect_status("Fully Constrained");
}

#[cfg(target_os = "linux")]
#[test]
fn native_midpoint_at_origin_with_typed_dimensions_has_no_freedom() {
    let mut world = ApplicationWorld::new();
    world.start_sketch_on("YZ Plane");
    let origin_id = world
        .edited_geometry()
        .points()
        .iter()
        .find(|point| point.position == [0., 0.])
        .unwrap()
        .id;
    choose_midpoint_line(&mut world);
    let origin = world.sketch_pixel_of([0., 0.]);
    super::super::x11_input::click(origin);
    world.pump(Vec::new());
    super::super::x11_input::type_digits("10");
    world.pump(Vec::new());
    super::super::x11_input::press_keysym(0xff09);
    world.pump(Vec::new());
    super::super::x11_input::type_digits("30");
    world.pump(Vec::new());
    super::super::x11_input::press_keysym(0xff0d);
    world.pump(Vec::new());
    super::super::x11_input::press_escape();
    world.pump(Vec::new());
    assert_eq!(world.dimensions_needed(), 0);
    world.expect_status("Fully Constrained");
    let geometry = world.edited_geometry();
    assert!(geometry.constraints().iter().any(|item| matches!(item.relation, PlanarConstraint::Midpoint { point, .. } if point == origin_id)));
    super::super::x11_input::press_control_letter(b'z');
    world.pump(Vec::new());
    assert!(world.edited_geometry().lines().is_empty());
    super::super::x11_input::press_control_letter(b'y');
    world.pump(Vec::new());
    assert_eq!(world.dimensions_needed(), 0);
    world.close();
}

#[cucumber::given("the origin-centered midpoint line has a driving length dimension of 10 mm")]
#[cucumber::given("the same midpoint line has length 10 mm and angle 30 degrees")]
fn origin_line_with_dimensions(world: &mut ApplicationWorld) {
    empty_right_sketch(world);
    midpoint_at_origin(world);
    enter_length(world);
    world.key(oblikovati_viewport::ViewportPhysicalKey::Tab, false);
    enter_angle(world);
    world.key(oblikovati_viewport::ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
    world.escape();
    world.pump(Vec::new());
    assert_eq!(
        world.dimensions_needed(),
        0,
        "expected origin-anchored fixture"
    );
}

#[cucumber::when("the user open the length dimension and enter 12 using separate key presses")]
fn enter_proposed_length(world: &mut ApplicationWorld) {
    world.open_dimension_label("10");
    world.type_text("1");
    world.type_text("2");
}

#[cucumber::when("the user press Escape and reopen the length dimension")]
fn cancel_and_reopen_length(world: &mut ApplicationWorld) {
    world.escape();
    world.pump(Vec::new());
    world.open_dimension_label("10");
}

#[cucumber::then("its stored expression remains 10 mm")]
fn original_length_remains(world: &mut ApplicationWorld) {
    let text = world.window_text("Edit Dimension");
    assert!(
        text.contains("10mm"),
        "expected 10 mm in the editor: {text:?}"
    );
}
