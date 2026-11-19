// SPDX-License-Identifier: GPL-2.0-only
//! Native pointer coverage for a parallel constraint mark.
use super::super::ApplicationWorld;
use crate::HeadSketchWork;
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

#[test]
fn glyph_context_hides_one_and_f8_restores_all_without_changing_the_sketch() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    parallel_lines(&mut world);
    add_vertical_and_reference_dimension(&mut world);
    world.pump(Vec::new());
    world.expect_annotation("//");
    world.expect_annotation("V");
    let original = world.edited_geometry();
    let dimension_labels = world.app().owner.sketch_dimension_annotations().unwrap();
    assert!(!dimension_labels.labels.is_empty());
    world.right_click(world.exact_run_center("//"));
    world.click_text("Hide");
    world.expect_no_annotation("//");
    world.expect_annotation("V");
    world.key(ViewportPhysicalKey::Z, true);
    world.expect_annotation("//");
    world.key(ViewportPhysicalKey::Y, true);
    world.expect_no_annotation("//");
    assert_eq!(world.edited_geometry(), original);
    assert_eq!(
        world.app().owner.sketch_dimension_annotations().unwrap(),
        dimension_labels
    );
    world.key(ViewportPhysicalKey::F8, false);
    world.expect_annotation("//");
    world.right_click(world.exact_run_center("//"));
    world.click_text("Hide All Constraints");
    world.expect_no_annotation("//");
    world.expect_no_annotation("V");
    world.key(ViewportPhysicalKey::Z, true);
    world.expect_annotation("//");
    world.expect_annotation("V");
    world.key(ViewportPhysicalKey::Y, true);
    world.expect_no_annotation("//");
    world.expect_no_annotation("V");
    assert_eq!(world.edited_geometry(), original);
    assert_eq!(
        world.app().owner.sketch_dimension_annotations().unwrap(),
        dimension_labels
    );
    world.close();
}

fn add_vertical_and_reference_dimension(world: &mut ApplicationWorld) {
    use oblikovati_model::{
        history::SketchDimensionCreation,
        sketch::{DistanceMeasure, MeasuredDimensionOperands, SketchDimensionalConstraintCreation},
    };
    let edit = world.app().owner.active_sketch_edit().unwrap();
    let first_line = world.edited_geometry().lines()[0];
    let mut commands = world.app().owner.document_commands(edit.document);
    let vertical = commands
        .create_sketch_primitive(
            edit.sketch,
            &SketchPrimitiveCreation::Line {
                endpoints: [
                    SketchPointInput::New([5., -1.]),
                    SketchPointInput::New([5., 3.]),
                ],
                construction: false,
                centerline: false,
            },
        )
        .unwrap()
        .entity;
    commands
        .create_sketch_geometric_constraint(
            edit.sketch,
            &SketchGeometricConstraintCreation::Relation(PlanarConstraint::LineVertical {
                line: vertical,
            }),
        )
        .unwrap();
    commands
        .create_sketch_dimension(
            edit.sketch,
            &SketchDimensionCreation {
                operands: SketchDimensionalConstraintCreation::Measured(
                    MeasuredDimensionOperands::Distance {
                        points: first_line.endpoints,
                        measure: DistanceMeasure::Aligned,
                    },
                ),
                expression: "60 mm".into(),
                reference: true,
                text_point: Some([0., -2.]),
            },
        )
        .unwrap();
}

#[test]
fn stale_glyph_menu_cannot_hide_a_relation_after_the_sketch_changes() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    parallel_lines(&mut world);
    world.right_click(world.exact_run_center("//"));
    let context = world.state().sketch_context.unwrap();
    let glyph = context.glyph.expect("expected the captured relation glyph");
    let edit = world.app().owner.active_sketch_edit().unwrap();
    world
        .app()
        .owner
        .document_commands(edit.document)
        .create_sketch_primitive(
            edit.sketch,
            &SketchPrimitiveCreation::Point { position: [8., 8.] },
        )
        .unwrap();
    assert!(
        world
            .app()
            .owner
            .hide_context_constraint_glyph(context.token, glyph)
            .is_err()
    );
    let sketch = world
        .app()
        .owner
        .documents()
        .workspace()
        .session(edit.document)
        .unwrap()
        .snapshot()
        .sketches
        .get(edit.sketch)
        .unwrap();
    assert!(!sketch.hidden_constraint_glyphs().contains(&glyph));
    world.close();
}

#[cfg(target_os = "linux")]
#[test]
fn x11_glyph_context_hide_and_f8_restore() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    parallel_lines(&mut world);
    let original = world.edited_geometry();
    super::super::x11_input::focus_window();
    let glyph = world.exact_run_center("//");
    move_pointer(&mut world, glyph);
    super::super::x11_input::right_click(world.exact_run_center("//"));
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.expect_text("Hide");
    super::super::x11_input::click(world.exact_run_center("Hide"));
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.expect_no_annotation("//");
    super::super::x11_input::press_keysym(0xffc5); // X11 F8
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.expect_annotation("//");
    assert_eq!(world.edited_geometry(), original);
    world.close();
}
