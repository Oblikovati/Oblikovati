// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::sketch::SketchConstraintId;
fn chain() -> Sketch2D {
    let mut sketch = Sketch2D::default();
    for (id, position) in [
        (1, [0., 0.]),
        (2, [0.1, 0.2]),
        (3, [0.3, 0.4]),
        (4, [4., 5.]),
    ] {
        sketch.add_point(SketchEntityId(id), position).unwrap();
    }
    for (id, points) in [(1, [1, 2]), (2, [2, 3])] {
        sketch
            .add_constraint(
                SketchConstraintId(id),
                PlanarConstraint::Coincident {
                    points: points.map(SketchEntityId),
                },
            )
            .unwrap();
    }
    sketch
}
#[test]
fn projected_anchor_freezes_transitive_coincident_coordinates_without_freezing_unrelated_points() {
    let sketch = chain();
    let anchors = [SketchPoint2D {
        id: SketchEntityId(1),
        position: [2., -3.],
    }];
    let constants =
        propagate(&sketch, &anchors, &BTreeMap::new(), &BTreeSet::from([0, 1])).unwrap();
    assert_eq!(constants, [(2, 2.), (3, -3.), (4, 2.), (5, -3.)]);
}
#[test]
fn conflicting_anchors_do_not_force_the_group_and_existing_fixed_values_remain() {
    let sketch = chain();
    let anchors = [
        SketchPoint2D {
            id: SketchEntityId(1),
            position: [0., 0.],
        },
        SketchPoint2D {
            id: SketchEntityId(3),
            position: [2., 0.],
        },
    ];
    let targets = BTreeMap::from([(2, Some(1.)), (3, Some(0.))]);
    let constants = propagate(&sketch, &anchors, &targets, &BTreeSet::from([0, 1, 4, 5])).unwrap();
    assert_eq!(constants, [(2, 1.), (3, 0.)]);
}
#[test]
fn authored_conflicts_do_not_impose_a_target_on_other_points() {
    let sketch = chain();
    let targets = BTreeMap::from([(0, None), (1, Some(3.)), (4, Some(2.))]);
    let constants = propagate(&sketch, &[], &targets, &BTreeSet::new()).unwrap();
    assert_eq!(constants, [(1, 3.), (3, 3.), (4, 2.), (5, 3.)]);
}
