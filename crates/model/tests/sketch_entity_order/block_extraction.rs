// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::{PlanarConstraint, SketchConstraintId};

fn shared_lines() -> Sketch2D {
    let mut sketch = Sketch2D::default();
    for (id, position) in [(1, [0., 0.]), (2, [1., 0.]), (3, [2., 0.])] {
        sketch.add_point(SketchEntityId(id), position).unwrap();
    }
    sketch
        .add_line(SketchEntityId(4), [1, 2].map(SketchEntityId))
        .unwrap();
    sketch
        .add_line(SketchEntityId(5), [2, 3].map(SketchEntityId))
        .unwrap();
    sketch
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::Horizontal {
                points: [1, 2].map(SketchEntityId),
            },
        )
        .unwrap();
    sketch
        .add_constraint(
            SketchConstraintId(2),
            PlanarConstraint::Horizontal {
                points: [2, 3].map(SketchEntityId),
            },
        )
        .unwrap();
    sketch
}

#[test]
fn extraction_keeps_unselected_curve_and_retires_relations_on_shared_points() {
    let mut sketch = shared_lines();
    let (captured, retired) = sketch
        .extract_block_geometry(&[SketchEntityId(4), SketchEntityId(2)])
        .unwrap();
    assert_eq!(
        captured.line(SketchEntityId(4)).unwrap().endpoints,
        [1, 2].map(SketchEntityId)
    );
    assert!(captured.entity_ids().contains(&SketchEntityId(2)));
    assert_eq!(
        sketch.line(SketchEntityId(5)).unwrap().endpoints,
        [2, 3].map(SketchEntityId)
    );
    assert!(!sketch.entity_ids().contains(&SketchEntityId(2)));
    assert_eq!(retired.constraints.len(), 2);
    assert!(sketch.constraints().is_empty());
    assert!(
        Sketch2D::from_recipe(&sketch.recipe())
            .unwrap()
            .same_recipe(&sketch)
    );
}

#[test]
fn invalid_extraction_leaves_the_source_unchanged() {
    let mut sketch = shared_lines();
    let before = sketch.clone();
    assert!(
        sketch
            .extract_block_geometry(&[SketchEntityId(4), SketchEntityId(99)])
            .is_err()
    );
    assert!(sketch.same_recipe(&before));
}

#[test]
fn extracting_a_parent_with_an_unselected_offset_is_rejected_atomically() {
    use oblikovati_model::sketch::SketchOffsetSpline2D;
    let mut sketch = super::defining_transforms::hidden_geometry();
    sketch
        .add_offset_spline(
            SketchOffsetSpline2D::new(SketchEntityId(23), SketchEntityId(21), 1.).unwrap(),
        )
        .unwrap();
    let before = sketch.clone();
    let error = sketch
        .extract_block_geometry(&[SketchEntityId(21)])
        .unwrap_err();
    assert!(error.to_string().contains("dependent entities"));
    assert!(sketch.same_recipe(&before));
}
