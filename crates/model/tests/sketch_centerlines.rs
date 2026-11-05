// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::{
    parameter::ParameterCollection,
    sketch::{PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId, SketchRecipe},
    units::UnitsOfMeasure,
};
use oblikovati_solve::nonlinear::SolveOptions;
fn centerline_fixture() -> Sketch2D {
    let mut sketch = Sketch2D::default();
    for (id, position) in [(1, [0.0, 0.0]), (2, [0.0, 4.0]), (6, [3.0, 1.0])] {
        sketch.add_point(SketchEntityId(id), position).unwrap();
    }
    sketch
        .add_line(SketchEntityId(9), [SketchEntityId(1), SketchEntityId(2)])
        .unwrap();
    sketch
        .add_line(SketchEntityId(3), [SketchEntityId(2), SketchEntityId(6)])
        .unwrap();
    sketch
        .add_circle(SketchEntityId(5), SketchEntityId(6), 1.0)
        .unwrap();
    sketch
}
fn marked_centerlines() -> Sketch2D {
    let mut sketch = centerline_fixture();
    sketch.set_line_centerline(SketchEntityId(3), true).unwrap();
    sketch.set_line_centerline(SketchEntityId(9), true).unwrap();
    sketch
}
#[test]
fn centerlines_imply_construction_and_are_excluded_from_profile_candidates() {
    let mut sketch = centerline_fixture();
    let axis = SketchEntityId(9);
    assert!(!sketch.is_centerline(axis).unwrap());
    assert!(sketch.set_line_centerline(axis, true).unwrap());
    assert!(!sketch.set_line_centerline(axis, true).unwrap());
    assert!(sketch.is_construction(axis).unwrap());
    assert_eq!(
        sketch.profile_curve_ids(),
        [SketchEntityId(3), SketchEntityId(5)]
    );
    assert!(sketch.set_line_centerline(axis, false).unwrap());
    assert!(!sketch.is_construction(axis).unwrap());
    assert!(!sketch.set_line_centerline(axis, false).unwrap());
}
#[test]
fn construction_and_centerline_are_independent_persistent_roles() {
    for construction in [false, true] {
        for centerline in [false, true] {
            assert_role_combination(construction, centerline);
        }
    }
}
fn assert_role_combination(construction: bool, centerline: bool) {
    let mut sketch = centerline_fixture();
    let axis = SketchEntityId(9);
    sketch
        .set_curve_construction(&[axis], construction)
        .unwrap();
    sketch.set_line_centerline(axis, centerline).unwrap();
    assert_eq!(
        sketch.is_construction(axis).unwrap(),
        construction || centerline
    );
    sketch.set_line_centerline(axis, false).unwrap();
    assert_eq!(sketch.is_construction(axis).unwrap(), construction);
}
#[test]
fn clearing_explicit_construction_does_not_clear_an_axis_role() {
    let mut sketch = marked_centerlines();
    sketch
        .set_curve_construction(&[SketchEntityId(9)], true)
        .unwrap();
    sketch
        .set_curve_construction(&[SketchEntityId(9)], false)
        .unwrap();
    assert!(sketch.is_construction(SketchEntityId(9)).unwrap());
    assert!(sketch.is_centerline(SketchEntityId(9)).unwrap());
}
#[test]
fn invalid_ownership_cannot_mutate_centerline_flags() {
    let mut sketch = marked_centerlines();
    let before = sketch.clone();
    for id in [1, 5, 99, 0] {
        let cause = sketch
            .set_line_centerline(SketchEntityId(id), true)
            .unwrap_err()
            .to_string();
        assert!(cause.contains(&id.to_string()), "{cause}");
        assert!(sketch.is_centerline(SketchEntityId(id)).is_err());
        assert!(before.same_recipe(&sketch));
    }
}
#[test]
fn axis_lookup_requires_one_unambiguous_nondegenerate_line() {
    let mut sketch = centerline_fixture();
    assert_eq!(sketch.centerline_axis().unwrap(), None);
    sketch.set_line_centerline(SketchEntityId(9), true).unwrap();
    assert_eq!(
        sketch.centerline_axis().unwrap(),
        Some(*sketch.line(SketchEntityId(9)).unwrap())
    );
    sketch.set_line_centerline(SketchEntityId(3), true).unwrap();
    let cause = sketch.centerline_axis().unwrap_err().to_string();
    assert!(
        cause.contains("9") && cause.contains("3") && cause.contains("at most one"),
        "{cause}"
    );
}
#[test]
fn coincident_axis_endpoints_are_a_named_decline() {
    let mut sketch = centerline_fixture();
    sketch
        .add_line(SketchEntityId(10), [SketchEntityId(1); 2])
        .unwrap();
    sketch
        .set_line_centerline(SketchEntityId(10), true)
        .unwrap();
    let cause = sketch.centerline_axis().unwrap_err().to_string();
    assert!(
        cause.contains("10") && cause.contains("distinct axis positions"),
        "{cause}"
    );
    assert!(sketch.is_centerline(SketchEntityId(10)).unwrap());
}
#[test]
fn centerline_metadata_round_trips_in_authored_order_and_preserves_recipe_bits() {
    let sketch = marked_centerlines();
    let recipe = sketch.recipe();
    assert_eq!(recipe.version, 4);
    assert_eq!(
        recipe.centerlines.as_deref().unwrap(),
        [SketchEntityId(9), SketchEntityId(3)]
    );
    assert_eq!(recipe.construction_curves.as_deref().unwrap(), []);
    assert_eq!(recipe.entity_order.as_deref().unwrap(), sketch.entity_ids());
    let encoded = serde_json::to_vec(&recipe).unwrap();
    let restored = Sketch2D::from_recipe(&serde_json::from_slice(&encoded).unwrap()).unwrap();
    assert!(sketch.same_recipe(&restored));
    assert_eq!(serde_json::to_vec(&restored.recipe()).unwrap(), encoded);
}
#[test]
fn clearing_centerlines_restores_the_unchanged_legacy_encoding() {
    let mut sketch = centerline_fixture();
    let encoded = serde_json::to_vec(&sketch.recipe()).unwrap();
    sketch.set_line_centerline(SketchEntityId(9), true).unwrap();
    assert!(!sketch.same_recipe(&centerline_fixture()));
    sketch
        .set_line_centerline(SketchEntityId(9), false)
        .unwrap();
    assert_eq!(sketch.recipe().version, 1);
    assert_eq!(serde_json::to_vec(&sketch.recipe()).unwrap(), encoded);
}
#[path = "sketch_centerlines/metadata.rs"]
mod metadata;
#[path = "sketch_centerlines/solve.rs"]
mod solve;
#[test]
fn different_point_identities_at_the_same_position_do_not_define_an_axis() {
    let mut sketch = Sketch2D::default();
    for (id, position) in [(1, [0.0, -0.0]), (2, [-0.0, 0.0])] {
        sketch.add_point(SketchEntityId(id), position).unwrap();
    }
    sketch
        .add_line(SketchEntityId(3), [SketchEntityId(1), SketchEntityId(2)])
        .unwrap();
    sketch.set_line_centerline(SketchEntityId(3), true).unwrap();
    assert!(
        sketch
            .centerline_axis()
            .unwrap_err()
            .to_string()
            .contains("distinct axis positions")
    );
}

#[test]
fn version_five_annotations_preserve_centerline_construction_and_entity_order() {
    let mut sketch = marked_centerlines();
    let circle = SketchEntityId(5);
    let dimension = SketchConstraintId(7);
    sketch.set_curve_construction(&[circle], true).unwrap();
    let relation = PlanarConstraint::Radius {
        curve: circle,
        parameter: 1,
    };
    sketch.add_constraint(dimension, relation).unwrap();
    let before = sketch.recipe();
    sketch
        .set_dimension_text_point(dimension, [2.0, 3.0])
        .unwrap();
    assert_annotation_keeps_curve_metadata(&sketch, &before);
}
fn assert_annotation_keeps_curve_metadata(sketch: &Sketch2D, before: &SketchRecipe) {
    let recipe = sketch.recipe();
    assert_eq!(recipe.version, 5);
    assert_eq!(recipe.entity_order, before.entity_order);
    assert_eq!(recipe.centerlines, before.centerlines);
    assert_eq!(recipe.construction_curves, before.construction_curves);
    let restored = Sketch2D::from_recipe(&recipe).unwrap();
    assert!(sketch.same_recipe(&restored));
}
