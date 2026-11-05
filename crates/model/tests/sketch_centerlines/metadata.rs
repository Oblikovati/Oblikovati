// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn malformed_centerline_order_and_owned_non_lines_are_refused() {
    for ids in [vec![9, 9], vec![3, 9], vec![1], vec![5], vec![99]] {
        let mut recipe = marked_centerlines().recipe();
        recipe.centerlines = Some(ids.into_iter().map(SketchEntityId).collect());
        assert!(Sketch2D::from_recipe(&recipe).is_err());
    }
}
#[test]
fn centerline_metadata_requires_its_complete_versioned_shape() {
    for version in [0, 1, 2, 3, 5] {
        let mut recipe = marked_centerlines().recipe();
        recipe.version = version;
        assert!(Sketch2D::from_recipe(&recipe).is_err());
    }
    for field in ["centerlines", "construction_curves", "entity_order"] {
        let mut recipe = marked_centerlines().recipe();
        match field {
            "centerlines" => recipe.centerlines = None,
            "construction_curves" => recipe.construction_curves = None,
            _ => recipe.entity_order = None,
        }
        assert!(Sketch2D::from_recipe(&recipe).is_err());
    }
}
#[test]
fn older_recipe_versions_restore_without_axis_roles_or_implicit_upgrades() {
    for version in [1, 2, 3] {
        let recipe = legacy_recipe(version);
        let restored = Sketch2D::from_recipe(&recipe).unwrap();
        assert!(restored.centerline_ids().is_empty());
        assert_eq!(restored.centerline_axis().unwrap(), None);
        assert!(restored.recipe().centerlines.is_none());
    }
}
fn legacy_recipe(version: u32) -> SketchRecipe {
    let mut recipe = centerline_fixture().recipe();
    recipe.version = version;
    recipe.entity_order = (version > 1).then(|| centerline_fixture().entity_ids().to_vec());
    recipe.construction_curves = (version == 3).then(|| vec![SketchEntityId(3)]);
    recipe
}
#[test]
fn explicit_empty_version_four_metadata_normalizes_after_valid_restore() {
    let mut recipe = marked_centerlines().recipe();
    recipe.centerlines = Some(Vec::new());
    let restored = Sketch2D::from_recipe(&recipe).unwrap();
    assert_eq!(restored.centerline_axis().unwrap(), None);
    assert_eq!(
        restored.profile_curve_ids(),
        [SketchEntityId(9), SketchEntityId(3), SketchEntityId(5)]
    );
}
