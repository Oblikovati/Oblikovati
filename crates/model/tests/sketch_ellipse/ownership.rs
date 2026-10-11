// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::{MOVEABLE_FREE, SKETCH_ENTITY_ELLIPSE};
use oblikovati_model::{
    parameter::ParameterCollection,
    sketch::{Sketch2D, SketchMovementOptions},
};
fn owned_ellipse() -> Sketch2D {
    let mut sketch = Sketch2D::default();
    sketch.add_point(SketchEntityId(1), [2.0, -3.0]).unwrap();
    sketch.add_ellipse(ellipse([3.0, 4.0], [3.0, 1.0])).unwrap();
    sketch
}
#[test]
fn shared_center_owner_and_three_shape_variables_produce_five_freedoms() {
    let sketch = owned_ellipse();
    let parameters = ParameterCollection::new(Default::default());
    let structure = sketch.analyze(&parameters, 1e-9).unwrap();
    assert_eq!(structure.degrees_of_freedom, 5);
    assert_eq!(sketch.ellipses().len(), 1);
    assert_eq!(
        sketch.ellipse(SketchEntityId(2)).unwrap().center(),
        SketchEntityId(1)
    );
    assert_eq!(
        sketch.entity_kind(SketchEntityId(2)).unwrap(),
        SKETCH_ENTITY_ELLIPSE
    );
    assert_ellipse_inspection(&sketch, &parameters);
}
fn assert_ellipse_inspection(sketch: &Sketch2D, parameters: &ParameterCollection<u64>) {
    let rows = sketch
        .inspect_entities(parameters, &[], SketchMovementOptions::default())
        .unwrap();
    assert_eq!(rows[1].kind, SKETCH_ENTITY_ELLIPSE);
    assert_eq!(rows[1].radius, 0.0);
    assert_eq!(rows[1].movement, MOVEABLE_FREE);
    assert_eq!(sketch.profile_curve_ids(), [SketchEntityId(2)]);
}

#[test]
fn failed_adoption_and_cross_kind_collisions_do_not_change_owner_order() {
    let mut sketch = Sketch2D::default();
    assert!(sketch.add_ellipse(ellipse([1.0, 0.0], [3.0, 1.0])).is_err());
    assert!(sketch.entity_ids().is_empty());
    sketch = owned_ellipse();
    let before = sketch.clone();
    assert!(sketch.add_ellipse(ellipse([1.0, 0.0], [3.0, 1.0])).is_err());
    assert!(
        sketch
            .add_circle(SketchEntityId(2), SketchEntityId(1), 1.0)
            .is_err()
    );
    assert!(sketch.add_point(SketchEntityId(2), [0.0; 2]).is_err());
    assert!(sketch.ellipse(SketchEntityId(99)).is_err());
    assert!(sketch.same_recipe(&before));
}
#[test]
fn translation_uses_shared_center_and_construction_excludes_only_the_curve() {
    let mut sketch = owned_ellipse();
    let shape = *sketch.ellipse(SketchEntityId(2)).unwrap();
    sketch
        .translate_entities(&[SketchEntityId(2)], [5.0, 7.0])
        .unwrap();
    assert_eq!(
        sketch.point(SketchEntityId(1)).unwrap().position,
        [7.0, 4.0]
    );
    assert_eq!(sketch.ellipse(SketchEntityId(2)).unwrap(), &shape);
    sketch
        .set_curve_construction(&[SketchEntityId(2)], true)
        .unwrap();
    assert!(sketch.profile_curve_ids().is_empty());
    assert!(sketch.is_construction(SketchEntityId(2)).unwrap());
    assert!(!sketch.is_construction(SketchEntityId(1)).unwrap());
}
#[test]
fn schema_seven_preserves_axis_bits_and_order_with_strict_saved_value_validation() {
    let mut sketch = owned_ellipse();
    sketch.add_point(SketchEntityId(3), [8.0, 9.0]).unwrap();
    sketch
        .set_curve_construction(&[SketchEntityId(2)], true)
        .unwrap();
    let recipe = sketch.recipe();
    assert_eq!(recipe.version, 7);
    let bytes = serde_json::to_vec(&recipe).unwrap();
    let restored_recipe = serde_json::from_slice(&bytes).unwrap();
    let restored = Sketch2D::from_recipe(&restored_recipe).unwrap();
    assert!(sketch.same_recipe(&restored));
    assert_eq!(
        restored.entity_ids(),
        [SketchEntityId(1), SketchEntityId(2), SketchEntityId(3)]
    );
    assert_bad_saved_ellipses();
}
fn assert_bad_saved_ellipses() {
    for (axis, radii) in [("[2,0]", "[3,1]"), ("[0,0]", "[3,1]"), ("[1,0]", "[1,3]")] {
        let record = format!(r#"{{"id":2,"center":1,"major_axis":{axis},"radii":{radii}}}"#);
        assert!(serde_json::from_str::<SketchEllipse2D>(&record).is_err());
    }
    let mut recipe = owned_ellipse().recipe();
    recipe.version = 6;
    assert!(Sketch2D::from_recipe(&recipe).is_err());
    recipe.version = 7;
    recipe.ellipses = None;
    assert!(Sketch2D::from_recipe(&recipe).is_err());
}

#[test]
fn history_comparison_observes_signed_axis_zero_instead_of_numeric_equality() {
    let mut positive = Sketch2D::default();
    positive.add_point(SketchEntityId(1), [0.0; 2]).unwrap();
    positive
        .add_ellipse(ellipse([1.0, 0.0], [3.0, 1.0]))
        .unwrap();
    let mut negative = Sketch2D::default();
    negative.add_point(SketchEntityId(1), [0.0; 2]).unwrap();
    negative
        .add_ellipse(ellipse([1.0, -0.0], [3.0, 1.0]))
        .unwrap();
    assert!(!positive.same_recipe(&negative));
    let encoded = serde_json::to_vec(&negative.recipe()).unwrap();
    let restored = Sketch2D::from_recipe(&serde_json::from_slice(&encoded).unwrap()).unwrap();
    assert!(negative.same_recipe(&restored));
}

#[test]
fn grounding_shared_center_removes_only_coordinate_freedoms_and_solve_keeps_frame_bits() {
    let sketch = owned_ellipse();
    let parameters = ParameterCollection::new(Default::default());
    let anchors = [oblikovati_model::sketch::SketchPoint2D {
        id: SketchEntityId(1),
        position: [2.0, -3.0],
    }];
    let structure = sketch
        .analyze_with_grounded_points(&parameters, &anchors, 1e-9)
        .unwrap();
    assert_eq!(structure.degrees_of_freedom, 3);
    let outcome = sketch
        .solve_with_grounded_points(&parameters, &anchors, Default::default(), || false)
        .unwrap();
    assert!(sketch.same_recipe(outcome.candidate()));
}

#[path = "dimension.rs"]
mod dimension;

#[path = "axis.rs"]
mod axis;
