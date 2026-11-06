// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::{
    SketchEllipse2D, SketchSelectionTransform, SketchSpline2D, SketchSplineKind,
};
pub(super) fn hidden_geometry() -> Sketch2D {
    let mut sketch = Sketch2D::default();
    for (id, position) in [(9, [0.0; 2]), (3, [2.0, 0.0]), (1, [0.0, 2.0])] {
        sketch
            .add_defining_point(SketchEntityId(id), position)
            .unwrap();
    }
    sketch
        .add_line(SketchEntityId(11), [SketchEntityId(9), SketchEntityId(3)])
        .unwrap();
    sketch
        .add_circle(SketchEntityId(7), SketchEntityId(9), 2.0)
        .unwrap();
    add_quarter_arc(&mut sketch, 1).unwrap();
    add_hidden_curves(&mut sketch);
    sketch.add_point(SketchEntityId(99), [10.0; 2]).unwrap();
    sketch
}
fn add_hidden_curves(sketch: &mut Sketch2D) {
    let ellipse = SketchEllipse2D::new(
        SketchEntityId(20),
        SketchEntityId(9),
        [1.0, 0.0],
        [3.0, 1.0],
    )
    .unwrap();
    sketch.add_ellipse(ellipse).unwrap();
    add_hidden_splines(sketch);
}
fn add_hidden_splines(sketch: &mut Sketch2D) {
    let kinds = [
        SketchSplineKind::Control,
        SketchSplineKind::Fit {
            method: oblikovati_api::types::SPLINE_FIT_SMOOTH,
        },
    ];
    for (index, kind) in kinds.into_iter().enumerate() {
        let points = [9, 3, 1].map(SketchEntityId).to_vec();
        let spline =
            SketchSpline2D::new(SketchEntityId(21 + index as u64), points, false, kind).unwrap();
        sketch.add_spline(spline).unwrap();
    }
}
#[test]
fn moving_curves_transforms_hidden_shared_defining_points_once() {
    assert_hidden_transform(
        SketchSelectionTransform::Move([3.0, 4.0]),
        [[3.0, 4.0], [5.0, 4.0], [3.0, 6.0]],
    );
}
#[test]
fn rotating_curves_transforms_hidden_centers_endpoints_and_spline_points() {
    let rotation = SketchSelectionTransform::Rotate {
        center: [1.0; 2],
        radians: std::f64::consts::FRAC_PI_2,
    };
    assert_hidden_transform(rotation, [[2.0, 0.0], [2.0, 2.0], [0.0, 0.0]]);
}
fn assert_hidden_transform(transform: SketchSelectionTransform, expected: [[f64; 2]; 3]) {
    let mut sketch = hidden_geometry();
    let visible = sketch.entity_ids().to_vec();
    let selected = [11, 7, 15, 20, 21, 22, 11].map(SketchEntityId);
    let points = sketch.transform_entities(&selected, transform).unwrap();
    assert_eq!(points, [9, 3, 1].map(SketchEntityId));
    for (id, expected) in points.into_iter().zip(expected) {
        near_coordinates(sketch.point(id).unwrap().position, expected);
    }
    assert_eq!(sketch.entity_ids(), visible);
    assert_eq!(
        sketch.point(SketchEntityId(99)).unwrap().position,
        [10.0; 2]
    );
    let recipe: SketchRecipe =
        serde_json::from_slice(&serde_json::to_vec(&sketch.recipe()).unwrap()).unwrap();
    assert!(sketch.same_recipe(&Sketch2D::from_recipe(&recipe).unwrap()));
}
fn near_coordinates(actual: [f64; 2], expected: [f64; 2]) {
    for (found, want) in actual.into_iter().zip(expected) {
        assert!((found - want).abs() < 1e-14, "{actual:?} != {expected:?}");
    }
}
#[test]
fn mixed_visible_and_hidden_selection_keeps_owned_order_and_unselected_points() {
    let mut sketch = hidden_geometry();
    let selected = [11, 99, 11].map(SketchEntityId);
    let changed = sketch.translate_entities(&selected, [3.0, 4.0]).unwrap();
    assert_eq!(changed, [99, 9, 3].map(SketchEntityId));
    assert_eq!(
        sketch.point(SketchEntityId(99)).unwrap().position,
        [13.0, 14.0]
    );
    assert_eq!(
        sketch.point(SketchEntityId(1)).unwrap().position,
        [0.0, 2.0]
    );
    assert_eq!(
        sketch.recipe().defining_points.unwrap(),
        [9, 3, 1].map(SketchEntityId)
    );
}
#[test]
fn hidden_output_overflow_refuses_the_entire_transform() {
    let mut sketch = hidden_geometry();
    sketch
        .set_position(SketchEntityId(3), [f64::MAX, 0.0])
        .unwrap();
    let before = sketch.clone();
    let error = sketch
        .translate_entities(&[SketchEntityId(11)], [f64::MAX, 0.0])
        .unwrap_err();
    assert!(error.to_string().contains("SketchEntityId(3)"));
    assert!(sketch.same_recipe(&before));
}
