// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::{AngleTurn, LineContactSide};
fn indexed_geometry() -> Sketch2D {
    let mut sketch = line_geometry([0.0; 2], [4.0, 1.0]);
    sketch.add_point(entity(4), [1.0, 4.0]).unwrap();
    sketch.add_line(entity(7), [entity(1), entity(4)]).unwrap();
    sketch.add_circle(entity(5), entity(4), 2.0).unwrap();
    sketch
        .add_arc(entity(8), entity(1), [entity(2), entity(4)], true)
        .unwrap();
    sketch
}
fn dimensional_relations() -> Vec<PlanarConstraint> {
    std::iter::once(distance_dimension())
        .chain(circular_dimensions())
        .chain(std::iter::once(offset_dimension()))
        .chain(angular_dimensions())
        .collect()
}
fn distance_dimension() -> PlanarConstraint {
    PlanarConstraint::Distance {
        points: [entity(1), entity(2)],
        measure: DistanceMeasure::Aligned,
        parameter: 11,
    }
}
fn circular_dimensions() -> [PlanarConstraint; 3] {
    [
        PlanarConstraint::Radius {
            curve: entity(5),
            parameter: 12,
        },
        PlanarConstraint::Diameter {
            curve: entity(5),
            parameter: 13,
        },
        PlanarConstraint::ArcLength {
            arc: entity(8),
            parameter: 14,
        },
    ]
}
fn offset_dimension() -> PlanarConstraint {
    PlanarConstraint::Offset {
        point: entity(4),
        line: entity(3),
        parameter: 15,
        side: LineContactSide::Left,
        linear_diameter: false,
    }
}
fn angular_dimensions() -> [PlanarConstraint; 2] {
    [
        PlanarConstraint::Angle {
            lines: [entity(3), entity(7)],
            parameter: 16,
            turn: AngleTurn::CounterClockwise,
        },
        PlanarConstraint::ThreePointAngle {
            vertex: entity(1),
            points: [entity(2), entity(4)],
            parameter: 17,
            turn: AngleTurn::CounterClockwise,
        },
    ]
}
#[test]
fn every_dimensional_family_is_excluded_from_geometric_indices_in_authored_order() {
    let mut sketch = indexed_geometry();
    append(&mut sketch, 30, line_relation(false));
    for (offset, relation) in dimensional_relations().into_iter().enumerate() {
        append(&mut sketch, offset as u64 + 1, relation);
    }
    append_tangent_dimension(&mut sketch);
    append(&mut sketch, 20, line_relation(true));
    let ids: Vec<u64> = sketch
        .geometric_constraints()
        .map(|constraint| constraint.id.0)
        .collect();
    assert_eq!(ids, [30, 20]);
    let index = sketch
        .geometric_constraint_index(SketchConstraintId(20))
        .unwrap();
    assert_eq!(index, 1);
    assert_excluded_ids(&sketch);
}
fn assert_excluded_ids(sketch: &Sketch2D) {
    for id in (1..=8).chain([0, 99]) {
        let cause = sketch
            .geometric_constraint_index(SketchConstraintId(id))
            .unwrap_err()
            .to_string();
        assert!(
            cause.contains(&id.to_string()) && cause.contains("owned geometric relation"),
            "{cause}"
        );
    }
    assert_eq!(
        sketch.parameter_dependencies(),
        (11..=18).collect::<Vec<_>>()
    );
}
#[test]
fn removing_geometric_records_reindexes_without_reusing_constraint_identities() {
    let mut sketch = indexed_geometry();
    append(&mut sketch, 9, line_relation(false));
    append_indexed_radius(&mut sketch);
    append(&mut sketch, 7, line_relation(true));
    let removed = sketch
        .remove_geometric_constraint(SketchConstraintId(9))
        .unwrap();
    assert_eq!(removed.id, SketchConstraintId(9));
    let index = sketch
        .geometric_constraint_index(SketchConstraintId(7))
        .unwrap();
    assert_eq!(index, 0);
    assert!(
        sketch
            .geometric_constraint_index(SketchConstraintId(9))
            .is_err()
    );
}
#[test]
fn geometric_line_pairs_and_point_alignments_remain_in_the_same_ordered_projection() {
    let mut sketch = indexed_geometry();
    append(&mut sketch, 4, point_relation(false));
    append(
        &mut sketch,
        1,
        PlanarConstraint::Lines {
            lines: [entity(3), entity(7)],
            relation: LineRelation::Parallel,
        },
    );
    append(&mut sketch, 9, line_relation(true));
    assert_eq!(
        sketch
            .geometric_constraints()
            .map(|constraint| constraint.id.0)
            .collect::<Vec<_>>(),
        [4, 1, 9]
    );
}
fn append_tangent_dimension(sketch: &mut Sketch2D) {
    sketch
        .add_tangent_distance(
            SketchConstraintId(8),
            entity(3),
            entity(5),
            18,
            Default::default(),
        )
        .unwrap();
}
fn append_indexed_radius(sketch: &mut Sketch2D) {
    let radius = PlanarConstraint::Radius {
        curve: entity(5),
        parameter: 1,
    };
    append(sketch, 2, radius);
}

#[test]
fn deletion_refuses_every_dimensional_family_and_missing_identity_without_mutation() {
    let mut sketch = indexed_geometry();
    for (offset, relation) in dimensional_relations().into_iter().enumerate() {
        append(&mut sketch, offset as u64 + 1, relation);
    }
    append_tangent_dimension(&mut sketch);
    let before = sketch.recipe();
    for id in (1..=8).chain([0, 99]) {
        let cause = sketch
            .remove_geometric_constraint(SketchConstraintId(id))
            .unwrap_err();
        assert!(cause.to_string().contains(&id.to_string()));
        assert_eq!(sketch.recipe(), before);
    }
}
