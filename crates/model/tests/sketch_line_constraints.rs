// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::{
    parameter::ParameterCollection,
    sketch::{
        DistanceMeasure, LineRelation, PlanarConstraint, Sketch2D, SketchConstraintId,
        SketchEntityId,
    },
    units::UnitsOfMeasure,
};
use oblikovati_solve::nonlinear::{SolveOptions, SolveTermination};
fn entity(id: u64) -> SketchEntityId {
    SketchEntityId(id)
}
fn line_geometry(start: [f64; 2], end: [f64; 2]) -> Sketch2D {
    let mut sketch = Sketch2D::default();
    sketch.add_point(entity(1), start).unwrap();
    sketch.add_point(entity(2), end).unwrap();
    sketch.add_line(entity(3), [entity(1), entity(2)]).unwrap();
    sketch
}
fn line_relation(vertical: bool) -> PlanarConstraint {
    if vertical {
        return PlanarConstraint::LineVertical { line: entity(3) };
    }
    PlanarConstraint::LineHorizontal { line: entity(3) }
}
fn point_relation(vertical: bool) -> PlanarConstraint {
    if vertical {
        return PlanarConstraint::Vertical {
            points: [entity(1), entity(2)],
        };
    }
    PlanarConstraint::Horizontal {
        points: [entity(1), entity(2)],
    }
}
fn append(sketch: &mut Sketch2D, id: u64, relation: PlanarConstraint) {
    sketch
        .add_constraint(SketchConstraintId(id), relation)
        .unwrap();
}
fn parameters() -> ParameterCollection<u64> {
    ParameterCollection::new(UnitsOfMeasure::default())
}
#[test]
fn single_line_relations_remain_distinct_from_point_alignment_in_saved_recipes() {
    for vertical in [false, true] {
        let mut sketch = line_geometry([0.0; 2], [4.0, 1.0]);
        append(&mut sketch, 1, line_relation(vertical));
        append(&mut sketch, 2, point_relation(vertical));
        assert_ne!(
            sketch.constraints()[0].relation,
            sketch.constraints()[1].relation
        );
        assert_restored_indices(&sketch);
    }
}
#[test]
fn missing_or_non_line_operands_are_refused_without_appending_a_constraint() {
    let mut sketch = line_geometry([0.0; 2], [4.0, 1.0]);
    sketch.add_circle(entity(4), entity(1), 1.0).unwrap();
    for line in [entity(0), entity(1), entity(4), entity(99)] {
        for relation in [
            PlanarConstraint::LineHorizontal { line },
            PlanarConstraint::LineVertical { line },
        ] {
            let cause = sketch
                .add_constraint(SketchConstraintId(1), relation)
                .unwrap_err()
                .to_string();
            assert!(cause.contains(&line.0.to_string()), "{cause}");
            assert!(sketch.constraints().is_empty());
        }
    }
}
#[test]
fn line_and_point_alignment_use_bit_identical_numerical_equations() {
    for vertical in [false, true] {
        let start = [2.0, 3.0];
        let end = if vertical { [3.0, 7.0] } else { [6.0, 4.0] };
        assert_shared_solution(line_geometry(start, end), vertical);
    }
}
fn assert_shared_solution(mut geometry: Sketch2D, vertical: bool) {
    let mut point_aligned = geometry.clone();
    append(&mut geometry, 1, line_relation(vertical));
    append(&mut point_aligned, 1, point_relation(vertical));
    let actual = geometry
        .solve(&parameters(), SolveOptions::default(), || false)
        .unwrap();
    let expected = point_aligned
        .solve(&parameters(), SolveOptions::default(), || false)
        .unwrap();
    assert_eq!(actual.numerical().termination, SolveTermination::Converged);
    assert_eq!(actual.numerical().values, expected.numerical().values);
    assert_eq!(actual.numerical().structure.rank, 1);
    assert_eq!(actual.numerical().structure.degrees_of_freedom, 3);
    assert_eq!(
        actual.candidate().constraints()[0].relation,
        line_relation(vertical)
    );
}
#[test]
fn grounded_single_line_relations_solve_with_one_remaining_length_freedom() {
    for vertical in [false, true] {
        let end = if vertical { [1.0, 4.0] } else { [4.0, 1.0] };
        let mut geometry = line_geometry([0.0; 2], end);
        append(&mut geometry, 1, line_relation(vertical));
        fix_start(&mut geometry, 2);
        let solved = geometry
            .solve(&parameters(), SolveOptions::default(), || false)
            .unwrap();
        assert_eq!(solved.numerical().termination, SolveTermination::Converged);
        assert_eq!(solved.numerical().structure.degrees_of_freedom, 1);
        let coordinate = usize::from(!vertical);
        let points = solved.candidate().points();
        assert!((points[1].position[coordinate] - points[0].position[coordinate]).abs() < 1e-10);
    }
}
#[test]
fn contradictory_fixed_endpoints_report_the_owned_line_constraint_as_conflicting() {
    let mut geometry = line_geometry([0.0; 2], [4.0, 1.0]);
    fix_all_points(&mut geometry);
    append(&mut geometry, 9, line_relation(false));
    let solved = geometry
        .solve(&parameters(), SolveOptions::default(), || false)
        .unwrap();
    assert_ne!(solved.numerical().termination, SolveTermination::Converged);
    assert!(
        solved
            .conflicting_constraints()
            .contains(&SketchConstraintId(9))
    );
}
#[test]
fn aligned_degenerate_endpoints_retain_the_existing_coordinate_equation_semantics() {
    for vertical in [false, true] {
        assert_shared_solution(line_geometry([1.0, 2.0], [1.0, 2.0]), vertical);
    }
}
#[path = "sketch_line_constraints/generated.rs"]
mod generated;
#[path = "sketch_line_constraints/indices.rs"]
mod indices;
fn assert_restored_indices(sketch: &Sketch2D) {
    let encoded = serde_json::to_vec(&sketch.recipe()).unwrap();
    let restored = Sketch2D::from_recipe(&serde_json::from_slice(&encoded).unwrap()).unwrap();
    assert!(sketch.same_recipe(&restored));
    for (index, id) in [1, 2].into_iter().enumerate() {
        assert_eq!(
            restored
                .geometric_constraint_index(SketchConstraintId(id))
                .unwrap(),
            index
        );
    }
}
fn fix_start(sketch: &mut Sketch2D, id: u64) {
    let first = sketch.points()[0];
    append(
        sketch,
        id,
        PlanarConstraint::Fix {
            point: first.id,
            position: first.position,
        },
    );
}
fn fix_all_points(sketch: &mut Sketch2D) {
    for point in sketch.points().to_vec() {
        append(
            sketch,
            point.id.0,
            PlanarConstraint::Fix {
                point: point.id,
                position: point.position,
            },
        );
    }
}
