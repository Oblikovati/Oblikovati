// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn failed_body_creation_burns_its_shared_feature_id_without_adopting_inputs() {
    let mut owner = part_owner();
    let mut recipe = prism_source().recipe();
    recipe.sketch.0 = 99;
    let invalid = SketchPrismDefinition::from_recipe(&recipe).unwrap();
    let token = owner.begin("Missing sketch").unwrap();
    assert!(
        owner
            .create_sketch_prism(&token, "Rejected", invalid)
            .is_err()
    );
    assert_eq!(owner.snapshot().work_geometry.features().len(), 1);
    assert_eq!(owner.feature_high_water(), 2);
    owner.abort(&token).unwrap();
    assert_eq!(append_prism(&mut owner), FeatureId(3));
}
#[test]
fn construction_and_plane_only_metadata_cannot_label_a_body_input() {
    let mut owner = part_owner();
    append_prism(&mut owner);
    let mut recipe = owner.snapshot().work_geometry.part_recipe().unwrap();
    recipe.features[1].construction = true;
    assert!(FeatureSequence::from_part_recipe(&recipe).is_err());
    recipe.features[1].construction = false;
    recipe.features[1].normal_flipped = true;
    assert!(FeatureSequence::from_part_recipe(&recipe).is_err());
}
