// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    history::SketchPolygonCreation,
    sketch::{LineRelation, PlanarConstraint, SketchPolygonSeed},
};
fn polygon_history() -> DocumentEditHistory {
    let mut snapshot = document_snapshot::empty_snapshot();
    snapshot
        .sketches
        .add(
            DocumentSketchId(1),
            "Polygon",
            SketchPlane::xy(),
            Sketch2D::default(),
        )
        .unwrap();
    DocumentEditHistory::new(snapshot, RecipeIdAllocator::default()).unwrap()
}
fn polygon_input(sides: usize, inscribed: bool) -> SketchPolygonCreation {
    SketchPolygonCreation {
        seed: SketchPolygonSeed {
            center: [1.0, 2.0],
            through: [4.0, 2.0],
            sides,
            inscribed,
        },
        construction: false,
    }
}
#[test]
fn polygons_own_ordered_vertices_circle_and_all_regularity_constraints() {
    for sides in [3, 4, 5, 8] {
        for inscribed in [true, false] {
            let mut history = polygon_history();
            let token = history.begin("Polygon").unwrap();
            let report = history
                .edit(&token, |snapshot, ids| {
                    snapshot.create_sketch_polygon(
                        DocumentSketchId(1),
                        &polygon_input(sides, inscribed),
                        ids,
                    )
                })
                .unwrap();
            history.commit(&token, false).unwrap();
            let geometry = history
                .snapshot()
                .sketches
                .get(DocumentSketchId(1))
                .unwrap()
                .geometry();
            assert_eq!(
                report.chain.points,
                (1..=sides as u64).map(SketchEntityId).collect::<Vec<_>>()
            );
            assert_eq!(report.center.0, 2 * sides as u64 + 1);
            assert_eq!(report.circle.0, 2 * sides as u64 + 2);
            assert_eq!(report.constraints.len(), 2 * sides - 1);
            assert!(geometry.is_construction(report.circle).unwrap());
            verify_polygon_relations(geometry, &report);
            assert!(geometry.same_recipe(&Sketch2D::from_recipe(&geometry.recipe()).unwrap()));
        }
    }
}
fn verify_polygon_relations(
    geometry: &Sketch2D,
    report: &oblikovati_model::history::SketchPolygonCreationReport,
) {
    for (index, constraint) in geometry.constraints().iter().enumerate() {
        let expected = if index < report.chain.points.len() {
            PlanarConstraint::PointOnCircularCurve {
                point: report.chain.points[index],
                curve: report.circle,
            }
        } else {
            let start = index - report.chain.points.len();
            PlanarConstraint::Lines {
                lines: [report.chain.lines[start], report.chain.lines[start + 1]],
                relation: LineRelation::EqualLength,
            }
        };
        assert_eq!(constraint.relation, expected);
    }
}
#[test]
fn polygon_entity_and_constraint_exhaustion_never_adopts_a_partial_chain() {
    for entity in [u64::MAX - 3, 0] {
        let mut history = polygon_history()
            .with_sketch_reservations(SketchReservationMarks {
                sketch: 1,
                entity,
                constraints: [(1, u64::MAX - 2)].into(),
                ..Default::default()
            })
            .unwrap();
        let before = history.snapshot().sketches.clone();
        let token = history.begin("Fail polygon").unwrap();
        let error = history
            .edit(&token, |snapshot, ids| {
                snapshot.create_sketch_polygon(DocumentSketchId(1), &polygon_input(4, true), ids)
            })
            .unwrap_err();
        let EditError::Mutation(cause) = error else {
            panic!("expected polygon mutation error")
        };
        assert!(cause.to_string().contains("regular polygon creation"));
        assert!(std::error::Error::source(&cause).is_some());
        assert!(history.snapshot().sketches.same_recipe(&before));
        assert!(history.sketch_reservations().entity > entity);
    }
}
#[test]
fn invalid_polygon_seeds_consume_no_identities() {
    for seed in [
        SketchPolygonSeed {
            sides: 2,
            ..polygon_input(4, true).seed
        },
        SketchPolygonSeed {
            through: [1.0, 2.0],
            ..polygon_input(4, true).seed
        },
        SketchPolygonSeed {
            center: [f64::NAN, 0.0],
            ..polygon_input(4, true).seed
        },
        SketchPolygonSeed {
            through: [1e200, 0.0],
            ..polygon_input(4, true).seed
        },
        SketchPolygonSeed {
            sides: usize::MAX,
            ..polygon_input(4, true).seed
        },
    ] {
        let mut history = polygon_history();
        let token = history.begin("Invalid").unwrap();
        let input = SketchPolygonCreation {
            seed,
            construction: false,
        };
        assert!(
            history
                .edit(&token, |snapshot, ids| snapshot.create_sketch_polygon(
                    DocumentSketchId(1),
                    &input,
                    ids
                ))
                .is_err()
        );
        assert_eq!(history.sketch_reservations().entity, 0);
        assert!(
            history
                .snapshot()
                .sketches
                .get(DocumentSketchId(1))
                .unwrap()
                .geometry()
                .points()
                .is_empty()
        );
    }
}
#[test]
fn polygon_constraints_allow_zero_dof_and_parameter_driven_radius_edits() {
    use oblikovati_model::{
        parameter::ParameterCollection, sketch::SketchConstraintId, units::UnitsOfMeasure,
    };
    use oblikovati_solve::nonlinear::{SolveOptions, SolveTermination};
    let mut history = polygon_history();
    let token = history.begin("Polygon").unwrap();
    let report = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_sketch_polygon(DocumentSketchId(1), &polygon_input(5, true), ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let mut geometry = history
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry()
        .clone();
    let mut parameters = ParameterCollection::<u64>::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "radius", oblikovati_api::types::USER_PARAM, "3 cm")
        .unwrap();
    for (index, relation) in [
        PlanarConstraint::Fix {
            point: report.center,
            position: [1.0, 2.0],
        },
        PlanarConstraint::Horizontal {
            points: [report.center, report.chain.points[0]],
        },
        PlanarConstraint::Radius {
            curve: report.circle,
            parameter: 1,
        },
    ]
    .into_iter()
    .enumerate()
    {
        geometry
            .add_constraint(SketchConstraintId(100 + index as u64), relation)
            .unwrap();
    }
    for radius in [3.0, 6.0] {
        parameters
            .set_expression(1, &format!("{radius} cm"))
            .unwrap();
        let outcome = geometry
            .solve(&parameters, SolveOptions::default(), || false)
            .unwrap();
        assert_eq!(outcome.numerical().termination, SolveTermination::Converged);
        assert_eq!(outcome.numerical().structure.degrees_of_freedom, 0);
        geometry = outcome.into_candidate();
        for point in &report.chain.points {
            let p = geometry.point(*point).unwrap().position;
            assert!(((p[0] - 1.0).hypot(p[1] - 2.0) - radius).abs() < 1e-7);
        }
    }
}
