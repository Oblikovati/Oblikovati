// SPDX-License-Identifier: GPL-2.0-only
use super::recompute::saved_driven_sketch;
use super::*;
use oblikovati_model::sketch::{
    DistanceMeasure, SketchConstraintId, SketchEntityId, SketchPointInput, SketchPrimitiveCreation,
};
fn new_line(host: &mut Host, document: u64, sketch: DocumentSketchId) -> SketchEntityId {
    let input = SketchPrimitiveCreation::Line {
        endpoints: [
            SketchPointInput::Owned(SketchEntityId(1)),
            SketchPointInput::New([4.0, 1.0]),
        ],
        construction: false,
        centerline: false,
    };
    host.commands(document)
        .unwrap()
        .create_sketch_primitive(sketch, &input)
        .unwrap()
        .entity
}
fn constrain_line(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
    sketch: DocumentSketchId,
    line: SketchEntityId,
) -> Result<(), String> {
    let mut geometry = snapshot.sketches.get(sketch).unwrap().geometry().clone();
    append_owned_line_constraints(&mut geometry, ids, sketch, line);
    snapshot
        .sketches
        .replace_geometry(sketch, geometry)
        .map_err(|cause| cause.to_string())
}
fn constrained_line_owner() -> (Host, SequenceGuids, u64, DocumentSketchId, SketchEntityId) {
    let (mut host, source, document, sketch) = saved_driven_sketch();
    let line = new_line(&mut host, document, sketch);
    host.commands(document)
        .unwrap()
        .run("Constrain line", |snapshot, ids| {
            constrain_line(snapshot, ids, sketch, line)
        })
        .unwrap();
    (host, source, document, sketch, line)
}
fn assert_owned_line_relation(
    host: &Host,
    document: u64,
    sketch: DocumentSketchId,
    line: SketchEntityId,
) {
    let snapshot = host.workspace().session(document).unwrap().snapshot();
    let view = snapshot.require_resolved_sketch(sketch).unwrap();
    let geometry = view.sketch.geometry();
    assert_eq!(view.verification.structure.degrees_of_freedom, 0);
    assert_eq!(
        geometry.constraints()[4].relation,
        PlanarConstraint::LineHorizontal { line }
    );
    assert_eq!(
        geometry
            .geometric_constraint_index(SketchConstraintId(5))
            .unwrap(),
        3
    );
    assert_unit_endpoint(geometry, line);
}
#[test]
fn source_current_line_constraints_keep_the_line_operand_and_filtered_index_through_history() {
    let (mut host, _, document, sketch, line) = constrained_line_owner();
    assert_owned_line_relation(&host, document, sketch, line);
    host.commands(document).unwrap().undo().unwrap();
    let snapshot = host.workspace().session(document).unwrap().snapshot();
    assert!(
        snapshot
            .sketches
            .get(sketch)
            .unwrap()
            .geometry()
            .geometric_constraint_index(SketchConstraintId(5))
            .is_err()
    );
    host.commands(document).unwrap().redo().unwrap();
    assert_owned_line_relation(&host, document, sketch, line);
}
#[test]
fn saved_line_constraint_operands_and_indices_survive_reopen_and_copy() {
    let (mut host, mut source, document, sketch, line) = constrained_line_owner();
    host.save(document, "alpha", &mut source).unwrap();
    host.save_copy(document, &path("line-copy.obk"), None, "alpha", &mut source)
        .unwrap();
    let copied = open_file(
        &mut host,
        "line-copy.obk",
        &mut source,
        OpenActivation::Background,
    );
    assert_owned_line_relation(&host, copied, sketch, line);
    host.close_discarding_changes(document).unwrap();
    let reopened = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    assert_owned_line_relation(&host, reopened, sketch, line);
}
fn append_owned_line_constraints(
    geometry: &mut Sketch2D,
    ids: &mut RecipeReservations<'_>,
    sketch: DocumentSketchId,
    line: SketchEntityId,
) {
    let endpoints = geometry.line(line).unwrap().endpoints;
    let relations = [
        PlanarConstraint::LineHorizontal { line },
        PlanarConstraint::Distance {
            points: endpoints,
            measure: DistanceMeasure::Horizontal,
            parameter: 1,
        },
    ];
    for relation in relations {
        geometry
            .add_constraint(ids.allocate_sketch_constraint(sketch).unwrap(), relation)
            .unwrap();
    }
}
fn assert_unit_endpoint(geometry: &Sketch2D, line: SketchEntityId) {
    let end = geometry.line(line).unwrap().endpoints[1];
    let position = geometry.point(end).unwrap().position;
    let tolerance = SolveOptions::default().relative_tolerance;
    // This unit-length fixture uses the solver's convergence policy as its metric bound.
    assert!(
        (position[0] - 1.0).abs() < tolerance && position[1].abs() < tolerance,
        "{position:?}"
    );
}
