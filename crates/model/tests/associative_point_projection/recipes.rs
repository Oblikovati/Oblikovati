// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn version_four_round_trip_preserves_sources_without_saving_verification() {
    let mut snapshot = snapshot();
    recompute(&mut snapshot);
    let recipe = snapshot.sketches.recipe();
    assert_eq!(recipe.version, 4);
    let bytes = serde_json::to_vec(&recipe).unwrap();
    let restored = DocumentSketches::from_recipe(&serde_json::from_slice(&bytes).unwrap()).unwrap();
    assert!(restored.same_recipe(&snapshot.sketches));
    assert!(!restored.same_evaluated_state(&snapshot.sketches));
    snapshot.sketches = restored;
    assert!(snapshot.require_resolved_sketch(SKETCH).is_err());
    recompute(&mut snapshot);
    assert_ready(&snapshot, [2.0, 4.0]);
}
#[test]
fn old_versions_cannot_accept_bindings_and_invalid_anchors_are_refused_on_restore() {
    let snapshot = snapshot();
    let recipe = snapshot.sketches.recipe();
    for version in [1, 2, 3, 5] {
        let mut invalid = recipe.clone();
        invalid.version = version;
        assert!(DocumentSketches::from_recipe(&invalid).is_err());
    }
    let mut invalid = recipe;
    invalid.sketches[0].point_projections[0].anchor = SketchEntityId(99);
    assert!(DocumentSketches::from_recipe(&invalid).is_err());
}
#[test]
fn removing_bindings_restores_the_original_version_and_free_point_dof() {
    let mut snapshot = snapshot();
    snapshot
        .sketches
        .set_point_projections(SKETCH, Vec::new())
        .unwrap();
    assert_eq!(snapshot.sketches.recipe().version, 1);
    assert!(
        !serde_json::to_string(&snapshot.sketches.recipe())
            .unwrap()
            .contains("point_projections")
    );
    assert_eq!(
        snapshot
            .analyze_document_sketch(SKETCH, 1e-9)
            .unwrap()
            .degrees_of_freedom,
        2
    );
}
#[test]
fn a_saved_warm_seed_cannot_override_current_projection_sources() {
    let mut snapshot = snapshot();
    let mut recipe = snapshot.sketches.recipe();
    recipe.sketches[0].geometry.points[0].position = [f64::MAX, -f64::MAX];
    snapshot.sketches = DocumentSketches::from_recipe(&recipe).unwrap();
    recompute(&mut snapshot);
    assert_ready(&snapshot, [2.0, 4.0]);
}
