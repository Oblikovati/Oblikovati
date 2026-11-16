// SPDX-License-Identifier: GPL-2.0-only
//! Native pointer coverage for a parallel constraint mark.
use super::super::ApplicationWorld;
use cucumber::{given, then, when};
use oblikovati_model::sketch::{
    LineRelation, PlanarConstraint, SketchGeometricConstraintCreation, SketchPointInput,
    SketchPrimitiveCreation,
};
use oblikovati_viewport::ViewportPhysicalKey;

#[given("the MCAD Software is displaying an editable sketch")]
fn editable_sketch(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
}
#[given("the viewport pointer is idle")]
fn idle_pointer(world: &mut ApplicationWorld) {
    move_pointer(world, [1100, 650]);
}
#[given("two lines have a visible parallel constraint glyph")]
fn parallel_lines(world: &mut ApplicationWorld) {
    let edit = world.app().owner.active_sketch_edit().unwrap();
    let mut commands = world.app().owner.document_commands(edit.document);
    let lines = [-1.0, 1.0].map(|y| {
        commands
            .create_sketch_primitive(edit.sketch, &line(y))
            .unwrap()
            .entity
    });
    let relation = PlanarConstraint::Lines {
        lines,
        relation: LineRelation::Parallel,
    };
    commands
        .create_sketch_geometric_constraint(
            edit.sketch,
            &SketchGeometricConstraintCreation::Relation(relation),
        )
        .unwrap();
    world.key(ViewportPhysicalKey::F8, false);
    world.pump(Vec::new());
    world.expect_annotation("//");
    world.record_initial_sketch();
    world.last_capture = Some(super::super::super::capture_framebuffer(
        &mut world.app().window.native,
    ));
}
fn line(y: f64) -> SketchPrimitiveCreation {
    SketchPrimitiveCreation::Line {
        endpoints: [
            SketchPointInput::New([-3., y]),
            SketchPointInput::New([3., y]),
        ],
        construction: false,
        centerline: false,
    }
}
#[when("the user hovers over the parallel constraint glyph")]
fn hover(world: &mut ApplicationWorld) {
    let position = world.exact_run_center("//");
    move_pointer(world, position);
}
fn move_pointer(world: &mut ApplicationWorld, position: [i32; 2]) {
    #[cfg(target_os = "linux")]
    super::super::x11_input::move_pointer(position);
    #[cfg(not(target_os = "linux"))]
    world.pump(vec![super::super::super::pointer_motion(position)]);
    world.pump(Vec::new());
}
#[then("both constrained lines highlight")]
fn lines_highlight(world: &mut ApplicationWorld) {
    let witnesses = [world.line_point(0, 0.25), world.line_point(1, 0.25)];
    let capture = super::super::super::capture_framebuffer(&mut world.app().window.native);
    let before = world.last_capture.as_ref().unwrap();
    for witness in witnesses {
        assert!(
            changed_near(&capture, before, witness),
            "expected highlight near {witness:?}"
        );
    }
}
fn changed_near(
    after: &oblikovati_viewport::ViewportCapture,
    before: &oblikovati_viewport::ViewportCapture,
    at: [i32; 2],
) -> bool {
    let width = after.extent[0] as i32;
    (-3..=3).any(|y| {
        (-3..=3).any(|x| {
            let offset = (((at[1] + y) * width + at[0] + x) * 4) as usize;
            after.rgba[offset..offset + 3] != before.rgba[offset..offset + 3]
        })
    })
}
#[then("the constraint remains in place")]
fn constraint_retained(world: &mut ApplicationWorld) {
    let current = world.edited_geometry();
    let previous = &world.observed_sketch.as_ref().unwrap().geometry;
    assert_eq!(current.constraints(), previous.constraints());
    assert_eq!(current.lines(), previous.lines());
    world.expect_annotation("//");
}
