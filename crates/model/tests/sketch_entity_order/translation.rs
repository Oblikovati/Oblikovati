// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn mixed_curve_selection_moves_shared_points_once_in_authored_order() {
    let mut sketch = mixed_geometry();
    sketch
        .set_curve_construction(&[SketchEntityId(7), SketchEntityId(15)], true)
        .unwrap();
    let before = sketch.recipe();
    let moved = sketch
        .translate_entities(&mixed_selection(), [10.0, 5.0])
        .unwrap();
    assert_eq!(
        moved,
        [SketchEntityId(9), SketchEntityId(3), SketchEntityId(1)]
    );
    assert_moved_points(&sketch);
    assert_same_curve_metadata(&before, &sketch.recipe());
    assert_eq!(
        sketch.arc_radius(SketchEntityId(15)).unwrap().to_bits(),
        2.0_f64.to_bits()
    );
}
fn assert_same_curve_metadata(before: &SketchRecipe, after: &SketchRecipe) {
    assert_eq!(before.lines, after.lines);
    assert_eq!(before.circles, after.circles);
    assert_eq!(before.arcs, after.arcs);
    assert_eq!(before.constraints, after.constraints);
    assert_eq!(before.entity_order, after.entity_order);
    assert_eq!(before.construction_curves, after.construction_curves);
}
#[test]
fn moving_one_point_does_not_move_other_defining_points_or_unrelated_geometry() {
    let mut sketch = mixed_geometry();
    let before = sketch.recipe();
    assert_eq!(
        sketch
            .translate_entities(&[SketchEntityId(3)], [1.0, 2.0])
            .unwrap(),
        [SketchEntityId(3)]
    );
    assert_eq!(
        sketch.point(SketchEntityId(3)).unwrap().position,
        [3.0, 2.0]
    );
    assert_eq!(sketch.point(SketchEntityId(9)).unwrap().position, [0.0; 2]);
    assert_eq!(
        sketch.point(SketchEntityId(1)).unwrap().position,
        [0.0, 2.0]
    );
    assert_same_curve_metadata(&before, &sketch.recipe());
}
#[test]
fn an_invalid_selection_never_moves_a_valid_prefix() {
    let mut sketch = mixed_geometry();
    let original = sketch.clone();
    let cause = sketch
        .translate_entities(&[SketchEntityId(11), SketchEntityId(99)], [2.0; 2])
        .unwrap_err()
        .to_string();
    assert!(cause.contains("99") && cause.contains("expected an owned"));
    assert!(sketch.same_recipe(&original));
}
#[test]
fn nonfinite_vectors_decline_even_for_empty_selections() {
    let mut sketch = mixed_geometry();
    let original = sketch.clone();
    for invalid in [f64::NAN, f64::INFINITY, f64::NEG_INFINITY] {
        let cause = sketch
            .translate_entities(&[], [invalid, 0.0])
            .unwrap_err()
            .to_string();
        assert!(cause.contains("expected two finite"));
        assert!(sketch.same_recipe(&original));
    }
}
#[test]
fn output_overflow_is_atomic_across_all_selected_points() {
    let mut sketch = Sketch2D::default();
    sketch.add_point(SketchEntityId(1), [0.0; 2]).unwrap();
    sketch
        .add_point(SketchEntityId(2), [f64::MAX, 0.0])
        .unwrap();
    let original = sketch.clone();
    let cause = sketch
        .translate_entities(&[SketchEntityId(1), SketchEntityId(2)], [f64::MAX, 0.0])
        .unwrap_err()
        .to_string();
    assert!(cause.contains("SketchEntityId(2)") && cause.contains("expected two finite"));
    assert!(sketch.same_recipe(&original));
}
#[test]
fn empty_and_zero_moves_keep_existing_recipe_bits_for_ordinary_coordinates() {
    let mut sketch = mixed_geometry();
    let original = sketch.clone();
    assert!(sketch.translate_entities(&[], [2.0; 2]).unwrap().is_empty());
    sketch
        .translate_entities(&[SketchEntityId(15)], [0.0; 2])
        .unwrap();
    assert!(sketch.same_recipe(&original));
}
#[test]
fn signed_zero_translation_uses_the_original_go_affine_expression() {
    let mut sketch = Sketch2D::default();
    sketch.add_point(SketchEntityId(1), [-0.0, 0.0]).unwrap();
    sketch
        .translate_entities(&[SketchEntityId(1)], [-0.0; 2])
        .unwrap();
    assert_eq!(sketch.points()[0].position.map(f64::to_bits), [0, 0]);
}
#[test]
fn translated_recipes_keep_roles_order_keys_and_shared_incidence() {
    let mut sketch = mixed_geometry();
    sketch
        .set_curve_construction(&[SketchEntityId(7)], true)
        .unwrap();
    sketch
        .translate_entities(&[SketchEntityId(15)], [3.0, 4.0])
        .unwrap();
    let encoded = serde_json::to_vec(&sketch.recipe()).unwrap();
    let restored = Sketch2D::from_recipe(&serde_json::from_slice(&encoded).unwrap()).unwrap();
    assert!(restored.same_recipe(&sketch));
    assert_roundtrip_key(restored);
}

fn assert_roundtrip_key(restored: Sketch2D) {
    let mut sketches = DocumentSketches::default();
    sketches
        .add(DocumentSketchId(1), "Moved", SketchPlane::xy(), restored)
        .unwrap();
    let names = DocumentSketchReferenceIndex::new(&sketches, guid()).unwrap();
    assert_eq!(
        names
            .entity_key(DocumentSketchId(1), SketchEntityId(7))
            .unwrap(),
        guid().sketch_entity_key(7).to_string()
    );
}

fn mixed_selection() -> [SketchEntityId; 5] {
    [
        SketchEntityId(15),
        SketchEntityId(11),
        SketchEntityId(7),
        SketchEntityId(9),
        SketchEntityId(11),
    ]
}
fn assert_moved_points(sketch: &Sketch2D) {
    assert_eq!(
        sketch.point(SketchEntityId(9)).unwrap().position,
        [10.0, 5.0]
    );
    assert_eq!(
        sketch.point(SketchEntityId(3)).unwrap().position,
        [12.0, 5.0]
    );
    assert_eq!(
        sketch.point(SketchEntityId(1)).unwrap().position,
        [10.0, 7.0]
    );
}
