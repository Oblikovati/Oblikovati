// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::{
    SketchEllipse2D, SketchError, SketchOffsetSpline2D, SketchSelectionTransform, SketchSpline2D,
    SketchSplineKind,
};
fn turn(center: [f64; 2], radians: f64) -> SketchSelectionTransform {
    SketchSelectionTransform::Rotate { center, radians }
}
fn near(actual: [f64; 2], expected: [f64; 2]) {
    for (found, want) in actual.into_iter().zip(expected) {
        assert!((found - want).abs() < 1e-14, "{actual:?} != {expected:?}");
    }
}
#[test]
fn quarter_turn_rotates_shared_points_once_and_keeps_curve_identity() {
    let mut sketch = mixed_geometry();
    let before = sketch.recipe();
    let selected = [9, 3, 11, 7, 1, 15, 11].map(SketchEntityId);
    let changed = sketch
        .transform_entities(&selected, turn([1.0, 1.0], std::f64::consts::FRAC_PI_2))
        .unwrap();
    assert_eq!(changed, [9, 3, 1].map(SketchEntityId));
    for (id, expected) in [(9, [2.0, 0.0]), (3, [2.0, 2.0]), (1, [0.0, 0.0])] {
        near(sketch.point(SketchEntityId(id)).unwrap().position, expected);
    }
    let after = sketch.recipe();
    assert_eq!(before.lines, after.lines);
    assert_eq!(before.circles, after.circles);
    assert_eq!(before.arcs, after.arcs);
    assert_eq!(before.entity_order, after.entity_order);
    assert_eq!(before.constraints, after.constraints);
}
#[test]
fn selected_ellipse_axis_turns_once_while_a_selected_center_leaves_it_alone() {
    let mut sketch = ellipse_sketch();
    let rotation = turn([0.0; 2], std::f64::consts::FRAC_PI_2);
    sketch
        .transform_entities(&[SketchEntityId(9)], rotation)
        .unwrap();
    assert_eq!(
        sketch.ellipse(SketchEntityId(20)).unwrap().major_axis(),
        [1.0, 0.0]
    );
    sketch
        .transform_entities(&[SketchEntityId(20); 2], rotation)
        .unwrap();
    let ellipse = sketch.ellipse(SketchEntityId(20)).unwrap();
    near(ellipse.major_axis(), [0.0, 1.0]);
    assert_eq!(ellipse.radii(), [3.0, 1.0]);
}
#[test]
fn invalid_rotation_never_publishes_a_valid_prefix() {
    let mut sketch = mixed_geometry();
    let line = [SketchEntityId(11)];
    for input in [
        turn([f64::NAN, 0.0], 1.0),
        turn([0.0; 2], f64::INFINITY),
        turn([f64::MAX; 2], 3.0),
    ] {
        refused_rotation(&mut sketch, &line, input);
    }
    let unknown = [SketchEntityId(11), SketchEntityId(999)];
    refused_rotation(&mut sketch, &unknown, turn([0.0; 2], 1.0));
}
fn refused_rotation(
    sketch: &mut Sketch2D,
    ids: &[SketchEntityId],
    transform: SketchSelectionTransform,
) -> SketchError {
    let before = sketch.clone();
    let error = sketch.transform_entities(ids, transform).unwrap_err();
    assert!(sketch.same_recipe(&before));
    error
}
#[test]
fn empty_selection_is_checked_and_zero_rotation_preserves_ordinary_coordinates() {
    let mut sketch = ellipse_sketch();
    assert!(
        sketch
            .transform_entities(&[], turn([0.0; 2], f64::NAN))
            .is_err()
    );
    assert!(
        sketch
            .transform_entities(&[], turn([0.0; 2], 1.0))
            .unwrap()
            .is_empty()
    );
    let before = sketch.clone();
    let selected = [SketchEntityId(11), SketchEntityId(20)];
    sketch
        .transform_entities(&selected, turn([0.0; 2], 0.0))
        .unwrap();
    assert!(sketch.same_recipe(&before));
}
#[test]
fn selected_point_deforms_incident_curves_without_turning_other_points() {
    let mut sketch = mixed_geometry();
    let rotation = turn([0.0; 2], std::f64::consts::FRAC_PI_2);
    sketch
        .transform_entities(&[SketchEntityId(3)], rotation)
        .unwrap();
    near_point(&sketch, 3, [0.0, 2.0]);
    assert_eq!(
        sketch.point(SketchEntityId(1)).unwrap().position,
        [0.0, 2.0]
    );
    assert_eq!(
        sketch.lines()[0].endpoints,
        [SketchEntityId(9), SketchEntityId(3)]
    );
}
#[test]
fn spline_control_and_fit_points_rotate_but_offsets_require_the_parent_selection() {
    for kind in [
        SketchSplineKind::Control,
        SketchSplineKind::Fit {
            method: oblikovati_api::types::SPLINE_FIT_SMOOTH,
        },
    ] {
        verify_rotated_spline(kind);
    }
}
fn verify_rotated_spline(kind: SketchSplineKind) {
    let mut sketch = spline_sketch(kind);
    let rotation = turn([0.0; 2], std::f64::consts::FRAC_PI_2);
    let selection = [SketchEntityId(20), SketchEntityId(21)];
    assert!(
        refused_rotation(&mut sketch, &selection, rotation)
            .to_string()
            .contains("parent")
    );
    sketch
        .transform_entities(&selection[..1], rotation)
        .unwrap();
    near_point(&sketch, 3, [0.0, 2.0]);
    near_point(&sketch, 1, [-2.0, 0.0]);
    assert_eq!(
        sketch.offset_spline(SketchEntityId(21)).unwrap().distance(),
        2.0
    );
}
fn spline_sketch(kind: SketchSplineKind) -> Sketch2D {
    let mut sketch = mixed_geometry();
    let points = [9, 3, 1].map(SketchEntityId).to_vec();
    let spline = SketchSpline2D::new(SketchEntityId(20), points, false, kind).unwrap();
    let offset = SketchOffsetSpline2D::new(SketchEntityId(21), SketchEntityId(20), 2.0).unwrap();
    sketch.add_spline(spline).unwrap();
    sketch.add_offset_spline(offset).unwrap();
    sketch
}
#[test]
fn finite_rotation_with_overflowing_output_is_atomic() {
    let mut sketch = Sketch2D::default();
    sketch.add_point(SketchEntityId(1), [1.0; 2]).unwrap();
    sketch.add_point(SketchEntityId(2), [f64::MAX; 2]).unwrap();
    let before = sketch.clone();
    let error = sketch
        .transform_entities(
            &[SketchEntityId(1), SketchEntityId(2)],
            turn([0.0; 2], std::f64::consts::FRAC_PI_4),
        )
        .unwrap_err();
    assert!(error.to_string().contains("2"));
    assert!(sketch.same_recipe(&before));
}
#[test]
fn repeated_ellipse_rotations_retain_a_reloadable_unit_axis() {
    let mut sketch = ellipse_sketch();
    for _ in 0..256 {
        sketch
            .transform_entities(&[SketchEntityId(20)], turn([0.0; 2], 0.37))
            .unwrap();
        let json = serde_json::to_vec(&sketch.recipe()).unwrap();
        let recipe: SketchRecipe = serde_json::from_slice(&json).unwrap();
        let restored = Sketch2D::from_recipe(&recipe).unwrap();
        assert!(sketch.same_recipe(&restored));
    }
}
fn ellipse_sketch() -> Sketch2D {
    let mut sketch = mixed_geometry();
    let ellipse = SketchEllipse2D::new(
        SketchEntityId(20),
        SketchEntityId(9),
        [1.0, 0.0],
        [3.0, 1.0],
    )
    .unwrap();
    sketch.add_ellipse(ellipse).unwrap();
    sketch
}

fn near_point(sketch: &Sketch2D, id: u64, expected: [f64; 2]) {
    let actual = sketch.point(SketchEntityId(id)).unwrap().position;
    near(actual, expected);
}
