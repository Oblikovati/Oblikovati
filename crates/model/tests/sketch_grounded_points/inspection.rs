// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::{
    GeometryMoveableStatus, MOVEABLE_BY_DIMENSION_CHANGE, MOVEABLE_FIXED, MOVEABLE_FREE,
};
use oblikovati_model::sketch::{SketchEntityInspection, SketchMovementOptions};
fn mixed_shapes() -> Sketch2D {
    let mut sketch = geometry(&[(1, [0.0; 2]), (2, [2.0, 0.0])]);
    sketch
        .add_line(SketchEntityId(5), [SketchEntityId(1), SketchEntityId(2)])
        .unwrap();
    sketch
        .add_circle(SketchEntityId(4), SketchEntityId(1), 2.0)
        .unwrap();
    sketch.add_point(SketchEntityId(3), [0.0, 2.0]).unwrap();
    sketch
        .add_arc(
            SketchEntityId(6),
            SketchEntityId(1),
            [SketchEntityId(2), SketchEntityId(3)],
            true,
        )
        .unwrap();
    sketch
}
fn inspected(sketch: &Sketch2D, anchors: &[SketchPoint2D]) -> Vec<SketchEntityInspection> {
    sketch
        .inspect_entities(&parameters(), anchors, Default::default())
        .unwrap()
}
fn statuses(sketch: &Sketch2D, anchors: &[SketchPoint2D]) -> Vec<GeometryMoveableStatus> {
    inspected(sketch, anchors)
        .into_iter()
        .map(|shape| shape.movement)
        .collect()
}
#[test]
fn analytic_observations_retain_authored_order_and_typed_kinds() {
    let sketch = mixed_shapes();
    let rows = inspected(&sketch, &[]);
    assert_eq!(
        rows.iter().map(|row| row.id.0).collect::<Vec<u64>>(),
        [1, 2, 5, 4, 3, 6]
    );
    assert_eq!(
        rows.iter()
            .map(|row| row.kind.0.as_ref())
            .collect::<Vec<&str>>(),
        ["point", "point", "line", "circle", "point", "arc"]
    );
}
#[test]
fn analytic_shapes_retain_shared_points_and_derived_circle_and_arc_radii() {
    let sketch = mixed_shapes();
    let rows = inspected(&sketch, &[]);
    assert_eq!(rows[2].points, [[0.0; 2], [2.0, 0.0]]);
    assert_eq!(rows[3].points, [[0.0; 2]]);
    assert_eq!(rows[5].points, [[0.0; 2], [2.0, 0.0], [0.0, 2.0]]);
    assert_eq!(
        rows.iter().map(|row| row.radius).collect::<Vec<f64>>(),
        [0.0, 0.0, 0.0, 2.0, 0.0, 2.0]
    );
    assert_eq!(statuses(&sketch, &[]), [MOVEABLE_FREE; 6]);
}
#[test]
fn grounded_and_active_coordinates_share_the_original_column_order() {
    let sketch = mixed_shapes();
    let original = sketch.clone();
    let rows = inspected(&sketch, &[point(1, [-0.0, 0.0])]);
    assert_eq!(rows[0].movement, MOVEABLE_FIXED);
    assert_eq!(rows[0].points[0][0].to_bits(), (-0.0_f64).to_bits());
    assert!(rows[1..].iter().all(|row| row.movement == MOVEABLE_FREE));
    assert!(sketch.same_recipe(&original));
}
#[test]
fn grounded_point_groups_hold_shared_curves_while_circle_radii_remain_free() {
    let sketch = mixed_shapes();
    let anchors = [
        point(1, [0.0; 2]),
        point(2, [2.0, 0.0]),
        point(3, [0.0, 2.0]),
    ];
    let frozen = inspected(&sketch, &anchors);
    let mut expected = [MOVEABLE_FIXED; 6];
    expected[3] = MOVEABLE_FREE;
    assert_eq!(movement_values(&frozen), expected);
}
#[test]
fn fixes_take_precedence_only_after_every_entity_variable_is_determined() {
    let mut sketch = mixed_shapes();
    install_fixed_points(&mut sketch);

    assert_eq!(
        statuses(&sketch, &[]),
        [
            MOVEABLE_FIXED,
            MOVEABLE_FIXED,
            MOVEABLE_FIXED,
            MOVEABLE_FREE,
            MOVEABLE_FIXED,
            MOVEABLE_FIXED
        ]
    );
}
#[test]
fn fixing_a_circle_center_and_driving_its_radius_holds_every_owned_variable() {
    let mut sketch = mixed_shapes();
    let anchors = [
        point(1, [0.0; 2]),
        point(2, [2.0, 0.0]),
        point(3, [0.0, 2.0]),
    ];
    relation(
        &mut sketch,
        4,
        PlanarConstraint::Radius {
            curve: SketchEntityId(4),
            parameter: 1,
        },
    );
    assert_eq!(statuses(&sketch, &anchors), [MOVEABLE_FIXED; 6]);
}
#[test]
fn dimensional_determination_is_distinct_from_fixed_ownership() {
    let mut sketch = driven_segment();
    sketch
        .add_line(SketchEntityId(3), [SketchEntityId(1), SketchEntityId(2)])
        .unwrap();
    let anchors = [point(1, [0.0; 2])];
    assert_eq!(
        statuses(&sketch, &anchors),
        [MOVEABLE_FIXED, MOVEABLE_BY_DIMENSION_CHANGE, MOVEABLE_FIXED]
    );
    let coupled = driven_segment();
    assert_eq!(statuses(&coupled, &[]), [MOVEABLE_FREE; 2]);
}
#[test]
fn current_anchor_positions_replace_stale_seeds_without_adopting_geometry() {
    let sketch = geometry(&[(1, [9.0, 8.0]), (2, [1.0, 2.0])]);
    let rows = inspected(&sketch, &[point(1, [3.0, 4.0])]);
    assert_eq!(rows[0].points, [[3.0, 4.0]]);
    assert_eq!(rows[1].points, [[1.0, 2.0]]);
    assert_eq!(
        sketch.point(SketchEntityId(1)).unwrap().position,
        [9.0, 8.0]
    );
    assert_eq!(rows[0].movement, MOVEABLE_FIXED);
    assert_eq!(rows[1].movement, MOVEABLE_FREE);
}
#[test]
fn invalid_policies_refuse_inspection() {
    let sketch = mixed_shapes();
    for options in [
        SketchMovementOptions {
            squared_freedom_tolerance: f64::NAN,
            ..Default::default()
        },
        SketchMovementOptions {
            rank_tolerance: 0.0,
            ..Default::default()
        },
    ] {
        assert!(
            sketch
                .inspect_entities(&parameters(), &[], options)
                .is_err()
        );
    }
}
#[test]
fn invalid_anchors_and_degenerate_arc_geometry_refuse_inspection() {
    let sketch = mixed_shapes();
    assert!(
        sketch
            .inspect_entities(&parameters(), &[point(99, [0.0; 2])], Default::default())
            .is_err()
    );
    assert!(
        sketch
            .inspect_entities(&parameters(), &[point(2, [0.0; 2])], Default::default())
            .is_err()
    );
}
#[test]
fn empty_and_failed_dimension_systems_are_explicit() {
    assert!(inspected(&Sketch2D::default(), &[]).is_empty());
    let sketch = driven_segment();
    let missing = ParameterCollection::new(Default::default());
    assert!(
        sketch
            .inspect_entities(&missing, &[], Default::default())
            .is_err()
    );
}

fn movement_values(rows: &[SketchEntityInspection]) -> Vec<GeometryMoveableStatus> {
    rows.iter().map(|row| row.movement).collect()
}
fn install_fixed_points(sketch: &mut Sketch2D) {
    for (id, position) in [(1, [0.0; 2]), (2, [2.0, 0.0]), (3, [0.0, 2.0])] {
        relation(
            sketch,
            id,
            PlanarConstraint::Fix {
                point: SketchEntityId(id),
                position,
            },
        );
    }
}
