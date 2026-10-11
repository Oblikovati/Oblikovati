// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::USER_PARAM;
use oblikovati_model::sketch::DistanceMeasure;
fn driven_owner() -> DocumentEditHistory {
    let snapshot = driven_snapshot();
    DocumentEditHistory::new(
        snapshot,
        RecipeIdAllocator::restore(Some(2), &[1, 2]).unwrap(),
    )
    .unwrap()
}
fn driven_geometry() -> Sketch2D {
    let mut sketch = reference();
    sketch
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::Fix {
                point: SketchEntityId(1),
                position: [0.0, 0.0],
            },
        )
        .unwrap();
    append_dimensions(&mut sketch);
    sketch
}
#[test]
fn parameter_change_in_the_same_edit_is_resolved_before_inference() {
    let mut history = driven_owner();
    let token = history.begin("Resize and author").unwrap();
    let report = history.edit(&token, resize_and_create).unwrap();
    history.commit(&token, false).unwrap();
    assert_eq!(report.inference.points[0].target, SketchEntityId(2));
    assert_eq!(report.point_constraints[0], SketchConstraintId(4));
    assert_eq!(report.direction.unwrap().geometric_index, 2);
    let position = geometry(&history)
        .point(SketchEntityId(2))
        .unwrap()
        .position;
    assert!((position[0] - 6.0).abs() < 1e-10);
}
fn policy() -> SketchInferencePolicy {
    SketchInferencePolicy::for_units(&UnitsOfMeasure::default())
}
#[test]
fn failed_parameter_source_does_not_reserve_ids_or_publish_coordinates() {
    let mut history = driven_owner();
    let before = history.sketch_reservations();
    let token = history.begin("Broken source").unwrap();
    history.edit(&token, refuse_failed_source).unwrap();
    history.abort(&token).unwrap();
    assert_eq!(history.sketch_reservations(), before);
}

fn driven_snapshot() -> oblikovati_model::history::DocumentSnapshot {
    let mut snapshot = document_snapshot::empty_snapshot();
    snapshot
        .parameters
        .insert_expression(1, "span", USER_PARAM, "3 cm")
        .unwrap();
    snapshot
        .parameters
        .insert_expression(2, "rise", USER_PARAM, "2 cm")
        .unwrap();
    let geometry = driven_geometry();
    snapshot
        .sketches
        .add(DocumentSketchId(1), "Driven", SketchPlane::xy(), geometry)
        .unwrap();
    snapshot
}

fn append_dimensions(sketch: &mut Sketch2D) {
    for (id, measure) in [
        (1, DistanceMeasure::Horizontal),
        (2, DistanceMeasure::Vertical),
    ] {
        sketch
            .add_constraint(
                SketchConstraintId(id + 1),
                PlanarConstraint::Distance {
                    points: [SketchEntityId(1), SketchEntityId(2)],
                    parameter: id,
                    measure,
                },
            )
            .unwrap();
    }
}

fn resize_and_create(
    snapshot: &mut oblikovati_model::history::DocumentSnapshot,
    ids: &mut oblikovati_model::history::RecipeReservations<'_>,
) -> Result<SketchLineCreationReport, String> {
    snapshot.parameters.set_expression(1, "6 cm").unwrap();
    let request = input([[6.0001, 2.0001], [10.0, 2.0001]]);
    snapshot
        .create_inferred_sketch_line(
            DocumentSketchId(1),
            &request,
            policy(),
            Default::default(),
            ids,
        )
        .map_err(|cause| cause.to_string())
}

fn refuse_failed_source(
    snapshot: &mut oblikovati_model::history::DocumentSnapshot,
    ids: &mut oblikovati_model::history::RecipeReservations<'_>,
) -> Result<(), String> {
    snapshot
        .parameters
        .set_expression(1, "missing_name")
        .unwrap();
    let geometry_before = snapshot.sketches.clone();
    let cause = snapshot
        .create_inferred_sketch_line(
            DocumentSketchId(1),
            &input([[0.0; 2], [5.0, 0.0]]),
            policy(),
            Default::default(),
            ids,
        )
        .unwrap_err();
    assert!(cause.to_string().contains("1"));
    assert_eq!(snapshot.sketches, geometry_before);
    Ok(())
}
