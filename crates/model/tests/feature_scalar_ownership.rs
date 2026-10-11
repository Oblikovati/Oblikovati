// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::feature::{
    AuthoredFeature, FeatureId, FeatureReadFootprint, FeatureReadKey, FeatureSequence,
    work_geometry::{DatumLength, WorkGeometryDefinition},
    work_plane::{WorkPlaneOrigin, WorkPlaneReference},
};

fn offset(parameter: u64) -> WorkGeometryDefinition {
    WorkGeometryDefinition::Offset {
        base: WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
        distance: DatumLength::Parameter(parameter),
    }
}
fn owner(id: u64, parameter: u64) -> AuthoredFeature<WorkGeometryDefinition> {
    AuthoredFeature::datum(FeatureId(id), "Offset", offset(parameter))
        .with_owned_scalar("offset", parameter)
        .unwrap()
}
fn program() -> FeatureSequence<WorkGeometryDefinition> {
    let mut program = FeatureSequence::default();
    program.append(owner(1, 9)).unwrap();
    program
}

#[test]
fn scalar_roles_are_stable_unique_attributed_identities() {
    let feature = owner(1, 9);
    assert_eq!(feature.owned_scalars()[0].role, "offset");
    assert_eq!(feature.owned_scalars()[0].parameter, 9);
    assert!(feature.clone().with_owned_scalar("offset", 9).is_err());
    assert!(feature.with_owned_scalar("angle", 9).is_err());
    for role in ["", "Offset", "2offset", "off set", "距離"] {
        assert!(
            AuthoredFeature::datum(FeatureId(1), "Plane", offset(9))
                .with_owned_scalar(role, 9)
                .is_err()
        );
    }
}
#[test]
fn bindings_require_positive_ids_and_exact_definition_reads() {
    assert!(
        AuthoredFeature::datum(FeatureId(1), "Plane", offset(0))
            .with_owned_scalar("offset", 0)
            .is_err()
    );
    let feature = AuthoredFeature::datum(FeatureId(1), "Plane", offset(9));
    let error = feature.with_owned_scalar("offset", 10).unwrap_err();
    assert!(error.to_string().contains("parameter: 10"));
    assert!(error.to_string().contains("attributed scalar read"));
}
#[test]
fn duplicate_roles_cannot_alias_distinct_parameters() {
    let reads = FeatureReadFootprint::new(
        [FeatureReadKey::Parameter(9), FeatureReadKey::Parameter(10)],
        false,
    );
    let feature = AuthoredFeature::new(FeatureId(1), "Plane", offset(9), reads);
    assert!(
        feature
            .with_owned_scalar("offset", 9)
            .unwrap()
            .with_owned_scalar("offset", 10)
            .is_err()
    );
}
#[test]
fn another_feature_can_read_but_cannot_claim_the_same_parameter() {
    let mut sequence = program();
    let error = sequence.append(owner(2, 9)).unwrap_err();
    assert!(error.to_string().contains("one owning feature"));
    assert_eq!(sequence.features().len(), 1);
    sequence
        .append(AuthoredFeature::datum(FeatureId(2), "Reader", offset(9)))
        .unwrap();
    sequence.append(owner(3, 10)).unwrap();
    assert_eq!(sequence.features().len(), 3);
}
#[test]
fn replacement_preserves_ownership_and_refuses_detached_scalar_reads() {
    let mut sequence = program();
    sequence
        .replace(FeatureId(1), offset(9), offset(9).reads())
        .unwrap();
    assert_eq!(
        sequence.get(FeatureId(1)).unwrap().owned_scalars(),
        owner(1, 9).owned_scalars()
    );
}
#[test]
fn replacement_cannot_detach_an_owned_parameter() {
    let mut sequence = program();
    let baseline = sequence.clone();
    assert!(
        sequence
            .replace(FeatureId(1), offset(10), offset(10).reads())
            .is_err()
    );
    assert!(baseline.same_work_geometry_recipe(&sequence));
}
#[test]
fn rename_preserves_parameter_ownership() {
    let mut sequence = program();
    sequence.rename(FeatureId(1), "Renamed").unwrap();
    assert_eq!(
        sequence.get(FeatureId(1)).unwrap().owned_scalars()[0].parameter,
        9
    );
}
#[test]
fn ownership_round_trip_requires_version_seven_and_changes_authored_equality() {
    let sequence = program();
    let recipe = sequence.work_geometry_recipe().unwrap();
    assert_eq!(recipe.version, 7);
    let json = serde_json::to_string(&recipe).unwrap();
    let restored =
        FeatureSequence::from_work_geometry_recipe(&serde_json::from_str(&json).unwrap()).unwrap();
    assert!(sequence.same_work_geometry_recipe(&restored));
}
#[test]
fn older_versions_refuse_ownership_and_equality_includes_it() {
    let sequence = program();
    let mut recipe = sequence.work_geometry_recipe().unwrap();
    recipe.version = 6;
    assert!(
        FeatureSequence::from_work_geometry_recipe(&recipe)
            .unwrap_err()
            .to_string()
            .contains("version 7")
    );
    recipe.features[0].owned_scalars.clear();
    let unowned = FeatureSequence::from_work_geometry_recipe(&recipe).unwrap();
    assert!(!sequence.same_work_geometry_recipe(&unowned));
}
#[test]
fn restored_metadata_cannot_bypass_binding_or_owner_validation() {
    let mut recipe = program().work_geometry_recipe().unwrap();
    recipe.features[0].owned_scalars[0].parameter = 10;
    assert!(FeatureSequence::from_work_geometry_recipe(&recipe).is_err());
    recipe.features[0].owned_scalars[0].parameter = 9;
    let mut second = recipe.features[0].clone();
    second.id = FeatureId(2);
    recipe.features.push(second);
    assert!(FeatureSequence::from_work_geometry_recipe(&recipe).is_err());
}
