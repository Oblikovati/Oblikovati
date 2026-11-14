// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::SketchConstraintId;

fn points(positions: &[[f64; 2]]) -> Sketch2D {
    let mut sketch = Sketch2D::default();
    for (index, position) in positions.iter().enumerate() {
        sketch
            .add_point(SketchEntityId(index as u64 + 1), *position)
            .unwrap();
    }
    sketch
}
fn coincide(sketch: &mut Sketch2D, constraint: u64, a: u64, b: u64) {
    sketch
        .add_constraint(
            SketchConstraintId(constraint),
            PlanarConstraint::Coincident {
                points: [SketchEntityId(a), SketchEntityId(b)],
            },
        )
        .unwrap();
}
#[test]
fn coincidence_groups_are_transitive_and_do_not_join_unrelated_equal_points() {
    let mut sketch = points(&[[0., 0.]; 4]);
    coincide(&mut sketch, 1, 2, 3);
    coincide(&mut sketch, 2, 3, 1);
    let groups = CoincidentPoints::new(&sketch);
    assert_eq!(
        groups.vertex(SketchEntityId(3)).unwrap().id,
        SketchEntityId(1)
    );
    assert_eq!(groups.sources(SketchEntityId(1)), vec![1, 2, 3]);
    assert_eq!(
        groups.vertex(SketchEntityId(4)).unwrap().id,
        SketchEntityId(4)
    );
}
#[test]
fn inconsistent_coincident_positions_are_not_moved_or_welded() {
    let mut sketch = points(&[[0., 0.], [1e-12, 0.]]);
    coincide(&mut sketch, 1, 1, 2);
    let before = sketch.recipe();
    let error = CoincidentPoints::new(&sketch)
        .vertex(SketchEntityId(2))
        .unwrap_err();
    assert_eq!(error.kind(), Kind::InvalidGeometry);
    assert!(error.to_string().contains("identical solved coordinates"));
    assert_eq!(sketch.recipe(), before);
}
