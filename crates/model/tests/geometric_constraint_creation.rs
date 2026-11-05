// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/document_snapshot.rs"]
mod document_snapshot;
use oblikovati_model::{
    document_sketch::{DocumentSketchId, PlacedSketchHealth, SketchEvaluation},
    history::{DocumentEditHistory, SketchGeometricCreationReport},
    identity::RecipeIdAllocator,
    sketch::{
        LineRelation, PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId,
        SketchGeometricConstraintCreation, SketchPlane,
    },
};
use oblikovati_solve::nonlinear::SolveOptions;
use std::error::Error;
fn entity(id: u64) -> SketchEntityId {
    SketchEntityId(id)
}
fn owner(geometry: Sketch2D) -> DocumentEditHistory {
    let mut snapshot = document_snapshot::empty_snapshot();
    snapshot
        .sketches
        .add(
            DocumentSketchId(1),
            "Relations",
            SketchPlane::xy(),
            geometry,
        )
        .unwrap();
    DocumentEditHistory::new(snapshot, RecipeIdAllocator::default()).unwrap()
}
fn fixture() -> Sketch2D {
    let mut geometry = Sketch2D::default();
    let points = [
        [0.0, 0.0],
        [1.0, 0.0],
        [0.0, 1.0],
        [3.0, 0.0],
        [4.0, 0.0],
        [3.0, 1.0],
    ];
    for (index, point) in points.into_iter().enumerate() {
        geometry.add_point(entity(index as u64 + 1), point).unwrap();
    }
    append_curves(&mut geometry);
    geometry
}
fn create(
    history: &mut DocumentEditHistory,
    input: &SketchGeometricConstraintCreation,
) -> SketchGeometricCreationReport {
    let token = history.begin("Create geometric relation").unwrap();
    let report = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_sketch_geometric_constraint(
                DocumentSketchId(1),
                input,
                Default::default(),
                ids,
            )
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    report
}
fn direct(relation: PlanarConstraint) -> SketchGeometricConstraintCreation {
    SketchGeometricConstraintCreation::Relation(relation)
}
fn point_inputs() -> Vec<SketchGeometricConstraintCreation> {
    alignment_inputs()
        .into_iter()
        .chain(incidence_inputs())
        .collect()
}
fn incidence_inputs() -> Vec<SketchGeometricConstraintCreation> {
    use PlanarConstraint::*;
    let mut inputs = vec![
        SketchGeometricConstraintCreation::Fix { point: entity(2) },
        direct(Midpoint {
            point: entity(6),
            line: entity(7),
        }),
        direct(ArcMidpoint {
            point: entity(6),
            arc: entity(11),
        }),
        direct(PointOnLine {
            point: entity(6),
            line: entity(7),
        }),
    ];
    inputs.extend(circular_incidence_input());
    inputs
}
fn line_inputs() -> Vec<SketchGeometricConstraintCreation> {
    let mut inputs = vec![
        direct(PlanarConstraint::LineHorizontal { line: entity(7) }),
        direct(PlanarConstraint::LineVertical { line: entity(7) }),
    ];
    for relation in [
        LineRelation::Parallel,
        LineRelation::Perpendicular,
        LineRelation::Collinear,
        LineRelation::EqualLength,
    ] {
        inputs.push(direct(PlanarConstraint::Lines {
            lines: [entity(7), entity(8)],
            relation,
        }));
    }
    inputs
}
fn circular_inputs() -> Vec<SketchGeometricConstraintCreation> {
    vec![
        direct(PlanarConstraint::Concentric {
            curves: [entity(9), entity(10)],
        }),
        direct(PlanarConstraint::EqualRadius {
            curves: [entity(9), entity(10)],
        }),
        SketchGeometricConstraintCreation::LineTangent {
            line: entity(8),
            curve: entity(9),
        },
        SketchGeometricConstraintCreation::CircularTangent {
            curves: [entity(9), entity(10)],
        },
    ]
}
#[test]
fn every_geometric_form_authors_a_real_relation_with_current_rank_and_health() {
    for input in point_inputs()
        .into_iter()
        .chain(line_inputs())
        .chain(circular_inputs())
    {
        let mut history = owner(fixture());
        let report = create(&mut history, &input);
        assert_created(&history, &report);
    }
}
fn assert_created(history: &DocumentEditHistory, report: &SketchGeometricCreationReport) {
    let snapshot = history.snapshot();
    let sketch = snapshot.sketches.get(DocumentSketchId(1)).unwrap();
    assert_eq!(report.constraint.id, SketchConstraintId(1));
    assert_eq!(report.geometric_index, 0);
    assert_eq!(
        sketch.geometry().constraints(),
        std::slice::from_ref(&report.constraint)
    );
    let rank = snapshot
        .analyze_document_sketch(DocumentSketchId(1), SolveOptions::default().rank_tolerance)
        .unwrap();
    assert_eq!(report.degrees_of_freedom, rank.degrees_of_freedom);
    assert!(matches!(
        sketch.health(&snapshot.parameters),
        PlacedSketchHealth::Current(SketchEvaluation::Numerical(_))
    ));
}
#[test]
fn invalid_dimensions_and_owned_operands_are_declined_before_identity_reservation() {
    for input in invalid_inputs() {
        let mut history = owner(fixture());
        let marks = history.sketch_reservations();
        assert!(!caught_failure(&mut history, &input).is_empty());
        assert_eq!(history.sketch_reservations(), marks);
    }
}
fn caught_failure(
    history: &mut DocumentEditHistory,
    input: &SketchGeometricConstraintCreation,
) -> String {
    let token = history.begin("Catch decline").unwrap();
    let cause = history
        .edit(&token, |snapshot, ids| -> Result<String, String> {
            caught_candidate(snapshot, ids, input)
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    cause
}
#[test]
fn contact_selection_failure_reserves_its_identity_without_publishing_geometry() {
    let mut history = owner(fixture());
    let input = SketchGeometricConstraintCreation::LineTangent {
        line: entity(7),
        curve: entity(9),
    };
    assert!(caught_failure(&mut history, &input).contains("nonzero seed side"));
    assert_eq!(history.sketch_reservations().constraints[&1], 1);
    let report = create(&mut history, &circular_inputs()[2]);
    assert_eq!(report.constraint.id, SketchConstraintId(2));
    assert_eq!(report.geometric_index, 0);
}
#[test]
fn source_compile_failure_keeps_the_attempted_identity_and_existing_dimension() {
    let mut history = missing_dimension_owner();
    let input = direct(PlanarConstraint::LineHorizontal { line: entity(7) });
    assert!(caught_failure(&mut history, &input).contains("99"));
    assert_eq!(history.sketch_reservations().constraints[&1], 2);
    assert_eq!(
        history.snapshot().sketches.sketches()[0]
            .geometry()
            .constraints()
            .len(),
        1
    );
}

fn append_curves(geometry: &mut Sketch2D) {
    geometry
        .add_line(entity(7), [entity(1), entity(2)])
        .unwrap();
    geometry
        .add_line(entity(8), [entity(3), entity(6)])
        .unwrap();
    geometry.add_circle(entity(9), entity(1), 1.0).unwrap();
    geometry.add_circle(entity(10), entity(4), 2.0).unwrap();
    geometry
        .add_arc(entity(11), entity(1), [entity(2), entity(3)], true)
        .unwrap();
}

fn alignment_inputs() -> Vec<SketchGeometricConstraintCreation> {
    use PlanarConstraint::*;
    vec![
        direct(Coincident {
            points: [entity(2), entity(4)],
        }),
        direct(Horizontal {
            points: [entity(2), entity(6)],
        }),
        direct(Vertical {
            points: [entity(3), entity(6)],
        }),
    ]
}

fn invalid_inputs() -> Vec<SketchGeometricConstraintCreation> {
    vec![
        direct(PlanarConstraint::Radius {
            curve: entity(9),
            parameter: 99,
        }),
        direct(PlanarConstraint::Coincident {
            points: [entity(2), entity(99)],
        }),
        SketchGeometricConstraintCreation::Fix { point: entity(99) },
        SketchGeometricConstraintCreation::LineTangent {
            line: entity(99),
            curve: entity(9),
        },
        SketchGeometricConstraintCreation::CircularTangent {
            curves: [entity(9), entity(99)],
        },
    ]
}

fn caught_candidate(
    snapshot: &mut oblikovati_model::history::DocumentSnapshot,
    ids: &mut oblikovati_model::history::RecipeReservations<'_>,
    input: &SketchGeometricConstraintCreation,
) -> Result<String, String> {
    let before = snapshot.sketches.clone();
    let cause = snapshot
        .create_sketch_geometric_constraint(DocumentSketchId(1), input, Default::default(), ids)
        .unwrap_err();
    assert!(cause.source().is_some());
    assert_eq!(snapshot.sketches, before);
    Ok(cause.to_string())
}

fn missing_dimension_owner() -> DocumentEditHistory {
    let mut geometry = fixture();
    geometry
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::Radius {
                curve: entity(9),
                parameter: 99,
            },
        )
        .unwrap();
    owner(geometry)
}

#[test]
fn fix_captures_current_driven_point_and_filters_the_existing_dimension_index() {
    let mut history = driven_owner();
    let token = history.begin("Drive then fix").unwrap();
    let report = history
        .edit(&token, |snapshot, ids| {
            snapshot.parameters.set_expression(1, "3 cm").unwrap();
            let input = SketchGeometricConstraintCreation::Fix { point: entity(2) };
            snapshot.create_sketch_geometric_constraint(
                DocumentSketchId(1),
                &input,
                Default::default(),
                ids,
            )
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    assert_driven_report(&report);
}
fn assert_current_fix(report: &SketchGeometricCreationReport) {
    let PlanarConstraint::Fix { point, position } = report.constraint.relation else {
        panic!("expected fix: {:?}", report.constraint);
    };
    assert_eq!(point, entity(2));
    let tolerance = 2.0 * SolveOptions::default().relative_tolerance;
    assert!((position[0] - 3.0).abs() <= tolerance, "{position:?}");
    assert!(position[1].abs() <= tolerance, "{position:?}");
}
fn driven_owner() -> DocumentEditHistory {
    let mut snapshot = document_snapshot::empty_snapshot();
    snapshot
        .parameters
        .insert_expression(1, "width", oblikovati_api::types::USER_PARAM, "1 cm")
        .unwrap();
    let mut geometry = Sketch2D::default();
    geometry.add_point(entity(1), [0.0; 2]).unwrap();
    geometry.add_point(entity(2), [1.0, 0.0]).unwrap();
    for (index, relation) in driven_relations().into_iter().enumerate() {
        geometry
            .add_constraint(SketchConstraintId(index as u64 + 1), relation)
            .unwrap();
    }
    snapshot
        .sketches
        .add(DocumentSketchId(1), "Driven", SketchPlane::xy(), geometry)
        .unwrap();
    DocumentEditHistory::new(snapshot, RecipeIdAllocator::restore(Some(1), &[1]).unwrap()).unwrap()
}
fn driven_relations() -> [PlanarConstraint; 3] {
    [
        PlanarConstraint::Fix {
            point: entity(1),
            position: [0.0; 2],
        },
        PlanarConstraint::Horizontal {
            points: [entity(1), entity(2)],
        },
        PlanarConstraint::Distance {
            points: [entity(1), entity(2)],
            measure: oblikovati_model::sketch::DistanceMeasure::Aligned,
            parameter: 1,
        },
    ]
}
#[test]
fn exhausted_constraint_namespace_does_not_publish_a_relation() {
    let marks = oblikovati_model::history::SketchReservationMarks {
        sketch: 1,
        entity: 11,
        constraints: std::collections::BTreeMap::from([(1, u64::MAX)]),
        dimension_names: Default::default(),
    };
    let mut history = owner(fixture())
        .with_sketch_reservations(marks.clone())
        .unwrap();
    let cause = caught_failure(&mut history, &line_inputs()[0]);
    assert!(
        cause.contains(&u64::MAX.to_string()) && cause.contains("below u64::MAX"),
        "{cause}"
    );
    assert_eq!(history.sketch_reservations(), marks);
}
#[test]
fn numerical_nonconvergence_retains_the_authored_relation_and_explicit_health() {
    let mut history = contradictory_owner();
    let report = create(
        &mut history,
        &direct(PlanarConstraint::Horizontal {
            points: [entity(1), entity(3)],
        }),
    );
    assert_eq!(report.constraint.id, SketchConstraintId(3));
    let snapshot = history.snapshot();
    let health = snapshot.sketches.sketches()[0].health(&snapshot.parameters);
    let PlacedSketchHealth::Current(SketchEvaluation::Numerical(numerical)) = health else {
        panic!("{health:?}");
    };
    assert_ne!(
        numerical.termination,
        oblikovati_solve::nonlinear::SolveTermination::Converged
    );
}

fn assert_driven_report(report: &SketchGeometricCreationReport) {
    assert_eq!(report.constraint.id, SketchConstraintId(4));
    assert_eq!(report.geometric_index, 2);
    assert_eq!(report.degrees_of_freedom, 0);
    assert_current_fix(report);
}

fn contradictory_owner() -> DocumentEditHistory {
    let mut geometry = fixture();
    for (offset, point) in [entity(1), entity(3)].into_iter().enumerate() {
        let position = geometry.point(point).unwrap().position;
        geometry
            .add_constraint(
                SketchConstraintId(offset as u64 + 1),
                PlanarConstraint::Fix { point, position },
            )
            .unwrap();
    }
    owner(geometry)
}

fn circular_incidence_input() -> Vec<SketchGeometricConstraintCreation> {
    vec![direct(PlanarConstraint::PointOnCircularCurve {
        point: entity(3),
        curve: entity(9),
    })]
}

#[path = "geometric_constraint_creation/generated.rs"]
mod generated;
