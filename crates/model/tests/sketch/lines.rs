// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::LineRelation;
use std::error::Error;
fn line_pair(second_anchor: [f64; 2], second_end: [f64; 2]) -> Sketch2D {
    let mut sketch = Sketch2D::default();
    for (id, position) in [
        (1, [0.0, 0.0]),
        (2, [4.0, 0.0]),
        (3, second_anchor),
        (4, second_end),
    ] {
        sketch.add_point(entity(id), position).unwrap();
    }
    sketch.add_line(entity(5), [entity(1), entity(2)]).unwrap();
    sketch.add_line(entity(6), [entity(3), entity(4)]).unwrap();
    for (id, position) in [(1, [0.0, 0.0]), (2, [4.0, 0.0])] {
        relation(
            &mut sketch,
            id,
            PlanarConstraint::Fix {
                point: entity(id),
                position,
            },
        );
    }
    sketch
}
fn lines(sketch: &mut Sketch2D, id: u64, kind: LineRelation) {
    relation(
        sketch,
        id,
        PlanarConstraint::Lines {
            lines: [entity(5), entity(6)],
            relation: kind,
        },
    );
}
fn distance(
    sketch: &mut Sketch2D,
    id: u64,
    points: [u64; 2],
    parameter: u64,
    measure: DistanceMeasure,
) {
    relation(
        sketch,
        id,
        PlanarConstraint::Distance {
            points: points.map(entity),
            parameter,
            measure,
        },
    );
}
#[test]
fn parallel_and_perpendicular_lines_solve_with_length_parameters() {
    for (kind, initial, expected) in [
        (LineRelation::Parallel, [3.0, 3.0], [5.0, 2.0]),
        (LineRelation::Perpendicular, [1.0, 6.0], [0.0, 7.0]),
    ] {
        let mut sketch = line_pair([0.0, 2.0], initial);
        relation(
            &mut sketch,
            3,
            PlanarConstraint::Fix {
                point: entity(3),
                position: [0.0, 2.0],
            },
        );
        lines(&mut sketch, 4, kind);
        distance(&mut sketch, 5, [3, 4], 1, DistanceMeasure::Aligned);
        let original = sketch.clone();
        let result = sketch
            .solve(&parameters(), SolveOptions::default(), || false)
            .unwrap();
        assert_eq!(result.numerical().termination, SolveTermination::Converged);
        assert_eq!(
            result.numerical().structure.status,
            ConstraintStatus::WellConstrained
        );
        assert_eq!(result.numerical().structure.degrees_of_freedom, 0);
        assert_point(result.candidate(), 4, expected);
        assert_eq!(original, sketch);
        assert_eq!(result.candidate().lines(), original.lines());
    }
}
#[test]
fn collinearity_adds_two_independent_equations_and_retains_both_line_ids() {
    let mut sketch = line_pair([2.0, 1.0], [5.0, 2.0]);
    lines(&mut sketch, 3, LineRelation::Collinear);
    distance(&mut sketch, 4, [1, 3], 2, DistanceMeasure::Horizontal);
    distance(&mut sketch, 5, [3, 4], 1, DistanceMeasure::Aligned);
    let result = sketch
        .solve(&parameters(), SolveOptions::default(), || false)
        .unwrap();
    assert_eq!(result.numerical().termination, SolveTermination::Converged);
    assert_eq!(result.numerical().structure.rank, 8);
    assert_eq!(result.numerical().structure.redundant_equations, 0);
    assert_point(result.candidate(), 3, [2.5, 0.0]);
    assert_point(result.candidate(), 4, [7.5, 0.0]);
    assert_eq!(result.candidate().lines()[0].id, entity(5));
    assert_eq!(result.candidate().lines()[1].id, entity(6));
}
#[test]
fn equal_length_and_point_on_infinite_line_solve_without_segment_clamping() {
    let mut sketch = line_pair([0.0, 2.0], [1.0, 5.0]);
    relation(
        &mut sketch,
        3,
        PlanarConstraint::Fix {
            point: entity(3),
            position: [0.0, 2.0],
        },
    );
    lines(&mut sketch, 4, LineRelation::Perpendicular);
    lines(&mut sketch, 5, LineRelation::EqualLength);
    let result = sketch
        .solve(&parameters(), SolveOptions::default(), || false)
        .unwrap();
    assert_eq!(result.numerical().termination, SolveTermination::Converged);
    assert_point(result.candidate(), 4, [0.0, 6.0]);
    let mut sketch = line_pair([7.0, 2.0], [9.0, 1.0]);
    relation(
        &mut sketch,
        3,
        PlanarConstraint::PointOnLine {
            point: entity(3),
            line: entity(5),
        },
    );
    distance(&mut sketch, 4, [1, 3], 1, DistanceMeasure::Horizontal);
    let result = sketch
        .solve(&parameters(), SolveOptions::default(), || false)
        .unwrap();
    assert_eq!(result.numerical().termination, SolveTermination::Converged);
    assert_point(result.candidate(), 3, [5.0, 0.0]);
    assert_eq!(result.numerical().structure.degrees_of_freedom, 2);
}
#[test]
fn shared_endpoints_and_structural_line_relations_use_one_coordinate_universe() {
    let mut sketch = Sketch2D::default();
    for (id, position) in [(1, [0.0, 0.0]), (2, [3.0, 0.0]), (3, [4.0, 2.0])] {
        sketch.add_point(entity(id), position).unwrap();
    }
    sketch.add_line(entity(4), [entity(1), entity(2)]).unwrap();
    sketch.add_line(entity(5), [entity(2), entity(3)]).unwrap();
    relation(
        &mut sketch,
        1,
        PlanarConstraint::Lines {
            lines: [entity(4), entity(5)],
            relation: LineRelation::Parallel,
        },
    );
    let structure = sketch.analyze(&parameters(), 1e-9).unwrap();
    assert_eq!(structure.variables, 6);
    assert_eq!(structure.rank, 1);
    let result = sketch
        .solve(&parameters(), SolveOptions::default(), || false)
        .unwrap();
    assert_eq!(result.numerical().termination, SolveTermination::Converged);
    assert_eq!(result.numerical().structure.degrees_of_freedom, 5);
    assert_eq!(
        result.candidate().lines()[0].endpoints[1],
        result.candidate().lines()[1].endpoints[0]
    );
    let mut same = line_pair([0.0, 2.0], [3.0, 3.0]);
    relation(
        &mut same,
        3,
        PlanarConstraint::Lines {
            lines: [entity(5), entity(5)],
            relation: LineRelation::Parallel,
        },
    );
    assert_eq!(
        same.analyze(&parameters(), 1e-9)
            .unwrap()
            .redundant_equations,
        1
    );
}
#[test]
fn collapsed_orientation_errors_name_the_authored_constraint_and_leave_geometry_intact() {
    for kind in [
        LineRelation::Parallel,
        LineRelation::Perpendicular,
        LineRelation::Collinear,
    ] {
        let mut sketch = line_pair([1.0, 1.0], [1.0, 1.0]);
        lines(&mut sketch, 73, kind);
        let original = sketch.clone();
        let error = sketch
            .solve(&parameters(), SolveOptions::default(), || false)
            .unwrap_err();
        assert!(matches!(
            error,
            SketchError::Residual {
                constraint: SketchConstraintId(73),
                ..
            }
        ));
        assert!(error.to_string().contains("nonzero direction"));
        assert!(error.source().is_some());
        assert!(
            sketch
                .analyze(&parameters(), 1e-9)
                .unwrap_err()
                .to_string()
                .contains("73")
        );
        assert_eq!(sketch, original);
    }
    let mut sketch = line_pair([1.0, 1.0], [1.0, 1.0]);
    relation(
        &mut sketch,
        74,
        PlanarConstraint::PointOnLine {
            point: entity(1),
            line: entity(6),
        },
    );
    assert!(
        sketch
            .solve(&parameters(), SolveOptions::default(), || false)
            .unwrap_err()
            .to_string()
            .contains("74")
    );
}
#[test]
fn unknown_line_references_are_atomic_and_parallel_perpendicular_conflicts_are_named() {
    let mut sketch = line_pair([0.0, 2.0], [3.0, 2.0]);
    let original = sketch.clone();
    let invalid = PlanarConstraint::Lines {
        lines: [entity(5), entity(999)],
        relation: LineRelation::Parallel,
    };
    assert!(
        sketch
            .add_constraint(constraint(3), invalid)
            .unwrap_err()
            .to_string()
            .contains("999")
    );
    assert_eq!(sketch, original);
    for (id, point, position) in [(3, 3, [0.0, 2.0]), (4, 4, [3.0, 2.0])] {
        relation(
            &mut sketch,
            id,
            PlanarConstraint::Fix {
                point: entity(point),
                position,
            },
        );
    }
    lines(&mut sketch, 77, LineRelation::Parallel);
    lines(&mut sketch, 78, LineRelation::Perpendicular);
    let result = sketch
        .solve(&parameters(), SolveOptions::default(), || false)
        .unwrap();
    assert_eq!(result.numerical().termination, SolveTermination::Stalled);
    assert!(result.conflicting_constraints().contains(&constraint(78)));
}

fn scaled_parallel(scale: f64) -> Sketch2D {
    let mut sketch = Sketch2D::default();
    for (id, position) in [
        (1, [0.0, 0.0]),
        (2, [4.0 * scale, 0.0]),
        (3, [0.0, 2.0 * scale]),
        (4, [3.0 * scale, 3.0 * scale]),
    ] {
        sketch.add_point(entity(id), position).unwrap();
        if id < 4 {
            relation(
                &mut sketch,
                id,
                PlanarConstraint::Fix {
                    point: entity(id),
                    position,
                },
            );
        }
    }
    sketch.add_line(entity(5), [entity(1), entity(2)]).unwrap();
    sketch.add_line(entity(6), [entity(3), entity(4)]).unwrap();
    lines(&mut sketch, 4, LineRelation::Parallel);
    distance(&mut sketch, 5, [3, 4], 1, DistanceMeasure::Aligned);
    sketch
}
#[test]
fn line_constraint_solve_and_rank_are_invariant_over_coordinate_scale_sweeps() {
    for scale in [1e-100, 1e-6, 1.0, 1e6, 1e100] {
        let sketch = scaled_parallel(scale);
        let mut parameters = parameters();
        parameters
            .set_expression(1, &format!("{} cm", 5.0 * scale))
            .unwrap();
        let structure = sketch.analyze(&parameters, 1e-9).unwrap();
        assert_eq!(structure.rank, 8, "scale {scale}");
        let result = sketch
            .solve(&parameters, SolveOptions::default(), || false)
            .unwrap();
        assert_eq!(
            result.numerical().termination,
            SolveTermination::Converged,
            "scale {scale}"
        );
        let point = result.candidate().point(entity(4)).unwrap().position;
        assert!((point[0] / scale - 5.0).abs() < 1e-8);
        assert!((point[1] / scale - 2.0).abs() < 1e-8);
    }
}
