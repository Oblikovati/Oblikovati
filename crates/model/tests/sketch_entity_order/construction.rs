// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::document_sketch::PlacedSketchHealth;
fn curve_ids() -> [SketchEntityId; 3] {
    [SketchEntityId(11), SketchEntityId(7), SketchEntityId(15)]
}
#[test]
fn construction_is_curve_metadata_and_profile_selection_retains_authored_order() {
    let mut sketch = mixed_geometry();
    assert_eq!(sketch.profile_curve_ids(), curve_ids());
    let changed = sketch
        .set_curve_construction(&[SketchEntityId(15), SketchEntityId(11)], true)
        .unwrap();
    assert_eq!(changed, [SketchEntityId(11), SketchEntityId(15)]);
    assert_eq!(sketch.profile_curve_ids(), [SketchEntityId(7)]);
    assert!(sketch.is_construction(SketchEntityId(11)).unwrap());
    assert!(!sketch.is_construction(SketchEntityId(9)).unwrap());
    assert_eq!(identifiers(&sketch), [9, 3, 11, 7, 1, 15]);
}
#[test]
fn setting_every_supported_curve_and_clearing_flags_are_idempotent() {
    let mut sketch = mixed_geometry();
    assert_eq!(
        sketch.set_curve_construction(&curve_ids(), true).unwrap(),
        curve_ids()
    );
    assert!(sketch.profile_curve_ids().is_empty());
    assert!(
        sketch
            .set_curve_construction(&curve_ids(), true)
            .unwrap()
            .is_empty()
    );
}
#[test]
fn clearing_every_flag_restores_the_original_recipe_and_empty_edits_are_no_ops() {
    let original = mixed_geometry();
    let mut sketch = Sketch2D::from_recipe(&flagged_recipe()).unwrap();
    assert_eq!(
        sketch.set_curve_construction(&curve_ids(), false).unwrap(),
        curve_ids()
    );
    assert!(sketch.same_recipe(&original));
    assert_eq!(sketch.recipe().version, 2);
    assert!(sketch.set_curve_construction(&[], true).unwrap().is_empty());
    assert!(
        sketch
            .set_curve_construction(&curve_ids(), false)
            .unwrap()
            .is_empty()
    );
}
#[test]
fn invalid_batches_cannot_partially_flag_curves_or_claim_a_point_role() {
    let mut sketch = mixed_geometry();
    let original = sketch.clone();
    for ids in [
        vec![SketchEntityId(11), SketchEntityId(99)],
        vec![SketchEntityId(9)],
        vec![SketchEntityId(0)],
        vec![SketchEntityId(11); 2],
    ] {
        let cause = sketch
            .set_curve_construction(&ids, true)
            .unwrap_err()
            .to_string();
        assert!(cause.contains("expected unique owned line/circle/arc/ellipse IDs"));
        assert!(cause.contains(&format!("{ids:?}")));
        assert!(sketch.same_recipe(&original));
    }
}
#[test]
fn construction_inspection_refuses_unknown_entities() {
    let sketch = mixed_geometry();
    let cause = sketch
        .is_construction(SketchEntityId(99))
        .unwrap_err()
        .to_string();
    assert!(cause.contains("99") && cause.contains("owned entity"));
}
#[test]
fn version_three_records_construction_in_authored_order_with_exact_entity_order() {
    let mut sketch = mixed_geometry();
    sketch
        .set_curve_construction(&[SketchEntityId(15), SketchEntityId(11)], true)
        .unwrap();
    let recipe = sketch.recipe();
    assert_eq!(recipe.version, 3);
    assert_eq!(recipe.entity_order.as_deref().unwrap(), sketch.entity_ids());
    assert_eq!(
        recipe.construction_curves.unwrap(),
        [SketchEntityId(11), SketchEntityId(15)]
    );
    let encoded = serde_json::to_vec(&sketch.recipe()).unwrap();
    let recipe = serde_json::from_slice(&encoded).unwrap();
    let restored = Sketch2D::from_recipe(&recipe).unwrap();
    assert!(restored.same_recipe(&sketch));
    assert_eq!(restored.profile_curve_ids(), [SketchEntityId(7)]);
}
fn legacy_geometry() -> Sketch2D {
    let mut recipe = mixed_geometry().recipe();
    recipe.version = 1;
    recipe.entity_order = None;
    Sketch2D::from_recipe(&recipe).unwrap()
}
#[test]
fn unchanged_legacy_content_keeps_its_encoding_and_flags_upgrade_the_complete_order() {
    let mut legacy = legacy_geometry();
    let before = serde_json::to_vec(&legacy.recipe()).unwrap();
    legacy
        .set_curve_construction(&[SketchEntityId(11)], true)
        .unwrap();
    assert_eq!(legacy.recipe().version, 3);
    assert_eq!(
        legacy.recipe().entity_order.as_deref().unwrap(),
        legacy.entity_ids()
    );
    legacy
        .set_curve_construction(&[SketchEntityId(11)], false)
        .unwrap();
    assert_eq!(legacy.recipe().version, 1);
    assert_eq!(serde_json::to_vec(&legacy.recipe()).unwrap(), before);
}
fn flagged_recipe() -> SketchRecipe {
    let mut sketch = mixed_geometry();
    sketch.set_curve_construction(&curve_ids(), true).unwrap();
    sketch.recipe()
}
#[test]
fn invalid_metadata_permutations_and_cross_version_shapes_are_declined() {
    for ids in [
        vec![SketchEntityId(99)],
        vec![SketchEntityId(9)],
        vec![SketchEntityId(11); 2],
        vec![SketchEntityId(15), SketchEntityId(11)],
    ] {
        let mut recipe = flagged_recipe();
        recipe.construction_curves = Some(ids);
        assert!(Sketch2D::from_recipe(&recipe).is_err());
    }
    for version in [1, 2, 4] {
        let mut recipe = flagged_recipe();
        recipe.version = version;
        assert!(Sketch2D::from_recipe(&recipe).is_err());
    }
}
#[test]
fn version_three_requires_both_order_and_explicit_metadata_but_can_clear_every_flag() {
    let mut recipe = flagged_recipe();
    recipe.entity_order = None;
    assert!(Sketch2D::from_recipe(&recipe).is_err());
    recipe = flagged_recipe();
    recipe.construction_curves = None;
    assert!(
        Sketch2D::from_recipe(&recipe)
            .unwrap_err()
            .to_string()
            .contains("explicit curve metadata")
    );
    recipe.construction_curves = Some(vec![]);
    let cleared = Sketch2D::from_recipe(&recipe).unwrap();
    assert_eq!(cleared.recipe().version, 2);
    assert_eq!(cleared.profile_curve_ids(), curve_ids());
    recipe.version = 2;
    assert!(Sketch2D::from_recipe(&recipe).is_err());
}
#[test]
fn metadata_edits_retain_numerical_health_shared_geometry_constraints_and_names() {
    let parameters = ParameterCollection::new(Default::default());
    let mut sketches = collection()
        .solve(&parameters, Default::default(), || false)
        .candidate;
    let before = sketches.clone();
    let report = sketches
        .set_construction(DocumentSketchId(42), &curve_ids(), true)
        .unwrap();
    assert_eq!(report.changed, curve_ids());
    assert_unchanged_numerical_state(&sketches, &before, &parameters);
    let names = DocumentSketchReferenceIndex::new(&sketches, guid()).unwrap();
    assert_eq!(
        names
            .entity_key(DocumentSketchId(42), SketchEntityId(11))
            .unwrap(),
        guid().sketch_entity_key(11).to_string()
    );
}
fn assert_unchanged_numerical_state(
    sketches: &DocumentSketches,
    before: &DocumentSketches,
    parameters: &ParameterCollection<u64>,
) {
    let placed = sketches.get(DocumentSketchId(42)).unwrap();
    let original = before.get(DocumentSketchId(42)).unwrap();
    assert_eq!(placed.health(parameters), original.health(parameters));
    assert!(matches!(
        placed.health(parameters),
        PlacedSketchHealth::Current(_)
    ));
    assert_eq!(placed.geometry().points(), original.geometry().points());
    assert_eq!(
        placed.geometry().constraints(),
        original.geometry().constraints()
    );
}
#[test]
fn construction_curves_still_participate_in_the_shared_solver_and_inspection() {
    let parameters = ParameterCollection::new(Default::default());
    let original = mixed_geometry();
    let original_analysis = original.analyze(&parameters, 1e-9).unwrap();
    let mut construction = original.clone();
    construction
        .set_curve_construction(&curve_ids(), true)
        .unwrap();
    assert_eq!(
        construction.analyze(&parameters, 1e-9).unwrap(),
        original_analysis
    );
    let result = construction
        .solve(&parameters, Default::default(), || false)
        .unwrap();
    assert!(result.candidate().same_recipe(&construction));
}
#[test]
fn entity_inspection_reports_actual_construction_roles_for_curves_only() {
    let parameters = ParameterCollection::new(Default::default());
    let construction = Sketch2D::from_recipe(&flagged_recipe()).unwrap();
    let rows = construction
        .inspect_entities(&parameters, &[], Default::default())
        .unwrap();
    assert_eq!(
        rows.iter()
            .filter(|row| row.construction)
            .map(|row| row.id)
            .collect::<Vec<_>>(),
        curve_ids()
    );
}
