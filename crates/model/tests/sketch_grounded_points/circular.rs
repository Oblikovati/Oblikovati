// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn grounded_circle_center_preserves_the_active_radius_and_circular_projection() {
    let mut sketch = geometry(&[(1, [0.0; 2])]);
    sketch
        .add_circle(SketchEntityId(3), SketchEntityId(1), 1.0)
        .unwrap();
    relation(
        &mut sketch,
        1,
        PlanarConstraint::Radius {
            curve: SketchEntityId(3),
            parameter: 1,
        },
    );
    let anchor = point(1, [7.0, -8.0]);
    let result = solve(&sketch, &[anchor]);
    assert_solved(&result, 1);
    assert_exact_anchor(&result, anchor);
    assert!((result.candidate().circles()[0].radius - 3.0).abs() < 1e-7);
}
#[test]
fn grounded_arc_points_compose_with_derived_radius_and_structural_circularity() {
    for (warm_start, expected) in [(8.0, 9.0), (4.0, 3.0)] {
        let sketch = arc_sketch([5.0, warm_start]);
        let anchors = [point(1, [5.0, 6.0]), point(2, [8.0, 6.0])];
        let result = solve(&sketch, &anchors);
        assert_solved(&result, 2);
        for anchor in anchors {
            assert_exact_anchor(&result, anchor);
        }
        let end = result
            .candidate()
            .point(SketchEntityId(3))
            .unwrap()
            .position;
        assert!((end[0] - 5.0).abs() < 1e-7);
        assert!((end[1] - expected).abs() < 1e-7);
    }
}
fn arc_sketch(end: [f64; 2]) -> Sketch2D {
    let mut sketch = geometry(&[(1, [0.0; 2]), (2, [1.0, 0.0]), (3, end)]);
    sketch
        .add_arc(
            SketchEntityId(4),
            SketchEntityId(1),
            [SketchEntityId(2), SketchEntityId(3)],
            true,
        )
        .unwrap();
    arc_constraints(&mut sketch);
    sketch
}
fn arc_constraints(sketch: &mut Sketch2D) {
    relation(
        sketch,
        1,
        PlanarConstraint::Vertical {
            points: [SketchEntityId(1), SketchEntityId(3)],
        },
    );
    relation(
        sketch,
        2,
        PlanarConstraint::Radius {
            curve: SketchEntityId(4),
            parameter: 1,
        },
    );
}
#[test]
fn repeated_reference_columns_are_substituted_without_creating_a_free_coordinate() {
    let mut sketch = geometry(&[(1, [1.0, 2.0])]);
    relation(
        &mut sketch,
        1,
        PlanarConstraint::Coincident {
            points: [SketchEntityId(1); 2],
        },
    );
    let anchor = point(1, [-0.0, 0.0]);
    let result = solve(&sketch, &[anchor]);
    assert_solved(&result, 0);
    assert_exact_anchor(&result, anchor);
    assert!(result.conflicting_constraints().is_empty());
}
