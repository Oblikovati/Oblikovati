// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn hidden_sketch_recipe_round_trips_in_version_five() {
    let mut encoded = serde_json::to_value(document().recipe()).unwrap();
    encoded["version"] = 5.into();
    encoded["sketches"][0]["visible"] = false.into();
    let recipe: DocumentSketchRecipe = serde_json::from_value(encoded.clone()).unwrap();
    let restored = DocumentSketches::from_recipe(&recipe).unwrap();
    assert_eq!(serde_json::to_value(restored.recipe()).unwrap(), encoded);
}

#[test]
fn legacy_recipes_keep_every_sketch_visible_without_new_fields() {
    let mut legacy = document().recipe();
    for version in 1..=4 {
        legacy.version = version;
        let encoded = serde_json::to_value(&legacy).unwrap();
        assert!(
            encoded["sketches"]
                .as_array()
                .unwrap()
                .iter()
                .all(|row| row.get("visible").is_none())
        );
        let decoded: DocumentSketchRecipe = serde_json::from_value(encoded).unwrap();
        let restored = DocumentSketches::from_recipe(&decoded).unwrap();
        assert!(restored.sketches().iter().all(|sketch| sketch.visible()));
    }
}

#[test]
fn visibility_edits_are_authored_content_and_repeated_values_are_noops() {
    let mut sketches = document();
    let original = sketches.clone();
    sketches.set_visible(DocumentSketchId(1), false).unwrap();
    assert!(!sketches.get(DocumentSketchId(1)).unwrap().visible());
    assert!(sketches.get(DocumentSketchId(2)).unwrap().visible());
    assert!(!original.same_recipe(&sketches));
    assert_ne!(original, sketches);
    let hidden = sketches.clone();
    sketches.set_visible(DocumentSketchId(1), false).unwrap();
    assert!(hidden.same_evaluated_state(&sketches));
    sketches.set_visible(DocumentSketchId(1), true).unwrap();
    assert!(original.same_evaluated_state(&sketches));
    assert_eq!(
        serde_json::to_vec(&sketches.recipe()).unwrap(),
        serde_json::to_vec(&original.recipe()).unwrap()
    );
}

#[test]
fn visibility_preserves_geometry_identity_and_read_footprints() {
    let mut sketches = document();
    let original = sketches.clone();
    sketches.set_visible(DocumentSketchId(1), false).unwrap();
    let before = original.get(DocumentSketchId(1)).unwrap();
    let after = sketches.get(DocumentSketchId(1)).unwrap();
    assert!(after.geometry().same_recipe(before.geometry()));
    assert_eq!(after.placement(), before.placement());
    assert_eq!(after.read_footprint(), before.read_footprint());
    assert_eq!(sketches.sketch_ids(), original.sketch_ids());
    assert_eq!(sketches.entity_ids(), original.entity_ids());
}

fn solved_visibility_fixture() -> (DocumentSketches, ParameterCollection<u64>) {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "5 cm")
        .unwrap();
    let solved = document().solve(&parameters, SolveOptions::default(), || false);
    assert!(solved.report.reports.iter().all(|row| row.result.is_ok()));
    (solved.candidate, parameters)
}

#[test]
fn repeated_visibility_changes_preserve_current_solver_verification() {
    let (mut sketches, parameters) = solved_visibility_fixture();
    let id = DocumentSketchId(1);
    let baseline = sketches
        .get(id)
        .unwrap()
        .require_fully_constrained(&parameters)
        .unwrap()
        .clone();
    let geometry = sketches.get(id).unwrap().geometry().clone();
    for visible in [false, false, true] {
        sketches.set_visible(id, visible).unwrap();
        let placed = sketches.get(id).unwrap();
        assert_eq!(placed.visible(), visible);
        let report = placed.require_fully_constrained(&parameters).unwrap();
        assert_eq!(report, &baseline);
        assert_eq!(
            report.relative_residual.to_bits(),
            baseline.relative_residual.to_bits()
        );
        assert!(placed.geometry().same_recipe(&geometry));
    }
}

#[test]
fn unknown_visibility_targets_leave_the_verified_document_unchanged() {
    let (mut sketches, _) = solved_visibility_fixture();
    let original = sketches.clone();
    for id in [0, 999] {
        let cause = sketches
            .set_visible(DocumentSketchId(id), false)
            .unwrap_err();
        assert!(cause.to_string().contains(&id.to_string()), "{cause}");
        assert!(original.same_evaluated_state(&sketches));
    }
}

#[test]
fn visibility_fields_require_version_five_even_for_explicit_true() {
    for version in 1..=4 {
        for visible in [false, true] {
            let mut recipe = document().recipe();
            recipe.version = version;
            recipe.sketches[0].visible = Some(visible);
            let cause = DocumentSketches::from_recipe(&recipe).unwrap_err();
            assert!(cause.to_string().contains("expected version 5"), "{cause}");
            assert!(
                cause.to_string().contains(&format!("Some({visible})")),
                "{cause}"
            );
        }
    }
}

#[test]
fn hidden_work_plane_sketch_keeps_its_reference_without_resolving_it() {
    use oblikovati_model::feature::FeatureId;
    let mut sketches = document();
    sketches
        .attach_work_plane(DocumentSketchId(1), FeatureId(7))
        .unwrap();
    let placement = sketches.get(DocumentSketchId(1)).unwrap().placement();
    sketches.set_visible(DocumentSketchId(1), false).unwrap();
    let restored = DocumentSketches::from_recipe(&sketches.recipe()).unwrap();
    let sketch = restored.get(DocumentSketchId(1)).unwrap();
    assert!(!sketch.visible());
    assert_eq!(sketch.placement(), placement);
    assert!(sketch.plane(None).is_err());
    assert_eq!(restored.recipe().version, 5);
}

#[test]
fn explicit_true_and_missing_overrides_restore_the_legacy_default() {
    let mut recipe = document().recipe();
    recipe.version = 5;
    recipe.sketches[0].visible = Some(true);
    let restored = DocumentSketches::from_recipe(&recipe).unwrap();
    assert!(restored.sketches().iter().all(|sketch| sketch.visible()));
    assert_eq!(restored.recipe(), document().recipe());
}
