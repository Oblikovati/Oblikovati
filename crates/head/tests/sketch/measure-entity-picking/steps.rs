// SPDX-License-Identifier: GPL-2.0-only
//! Native Measure picks on a circular arc (#4034).
use super::super::ApplicationWorld;
use cucumber::{given, then, when};
use oblikovati_model::sketch::{SketchPointInput, SketchPrimitiveCreation};
use oblikovati_viewport::ViewportPhysicalKey;

#[given("an arc with a selectable midpoint is available in Measure")]
fn arc_in_measure(world: &mut ApplicationWorld) {
    let edit = world.app().owner.active_sketch_edit().unwrap();
    let arc = SketchPrimitiveCreation::Arc {
        center: SketchPointInput::New([0., 0.]),
        endpoints: [
            SketchPointInput::New([3., 0.]),
            SketchPointInput::New([0., 3.]),
        ],
        counter_clockwise: true,
        construction: false,
    };
    world
        .app()
        .owner
        .document_commands(edit.document)
        .create_sketch_primitive(edit.sketch, &arc)
        .unwrap();
    world.key(ViewportPhysicalKey::M, false);
    world.pump(Vec::new());
    assert!(super::super::diagnostics::window_geometry("Measure").is_some());
    world.record_initial_sketch();
}
#[when("the user selects the arc midpoint")]
fn select_midpoint(world: &mut ApplicationWorld) {
    let radius = 3. / 2.0_f64.sqrt();
    let position = world.sketch_pixel_of([radius, radius]);
    hover(world, position);
    assert!(matches!(
        world.app().window.native.tool_hover.as_ref(),
        Some(oblikovati_viewport::ViewportSketchFeedback::Point { .. })
    ));
    world.save_capture("measure-midpoint-hover-4034");
    click(world, position);
}
#[then("the selection is identified as a Middle Point")]
fn midpoint_identified(world: &mut ApplicationWorld) {
    world.expect_measure("Selection 1: Middle Point");
    world.save_capture("measure-arc-midpoint-4034");
    let radius = 3. / 2.0_f64.sqrt();
    let position = world.sketch_pixel_of([radius, radius]);
    hover(world, position);
    assert!(world.app().window.native.tool_hover.is_none());
}
#[when("the user clears the selection and selects the arc near its start")]
fn clear_and_select_curve(world: &mut ApplicationWorld) {
    click(world, [1100, 650]);
    let angle = std::f64::consts::PI / 12.;
    let position = world.sketch_pixel_of([3. * angle.cos(), 3. * angle.sin()]);
    hover(world, position);
    assert!(matches!(
        world.app().window.native.tool_hover.as_ref(),
        Some(oblikovati_viewport::ViewportSketchFeedback::Entities(_))
    ));
    world.save_capture("measure-curve-hover-4034");
    click(world, position);
}
#[then("the selection is identified as a Curve")]
fn curve_identified(world: &mut ApplicationWorld) {
    world.expect_measure("Selection 1: Curve");
    let current = world.edited_geometry();
    let before = &world.observed_sketch.as_ref().unwrap().geometry;
    assert_eq!(current.arcs(), before.arcs());
}
fn click(world: &mut ApplicationWorld, position: [i32; 2]) {
    #[cfg(target_os = "linux")]
    super::super::x11_input::click(position);
    #[cfg(not(target_os = "linux"))]
    world.click(position);
    world.pump(Vec::new());
}

fn hover(world: &mut ApplicationWorld, position: [i32; 2]) {
    #[cfg(target_os = "linux")]
    super::super::x11_input::move_pointer(position);
    #[cfg(not(target_os = "linux"))]
    world.pump(vec![super::super::super::pointer_motion(position)]);
    world.pump(Vec::new());
}
