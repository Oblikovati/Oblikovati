// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::{MODEL_PARAM, ParameterKind, USER_PARAM};
use oblikovati_model::{
    feature::{
        AuthoredFeature, FeatureId, FeatureSequence,
        work_geometry::{DatumLength, WorkGeometryDefinition},
        work_plane::{WorkPlaneOrigin, WorkPlaneReference},
    },
    parameter::ParameterCollection,
    units::UnitsOfMeasure,
};
use oblikovati_persistence::{
    load::{LoadedParameterPackage, PackageLoadError, load_parameter_package},
    package::DocumentPackage,
};

fn owned_program() -> FeatureSequence<WorkGeometryDefinition> {
    let definition = WorkGeometryDefinition::Offset {
        base: WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
        distance: DatumLength::Parameter(9),
    };
    let feature = AuthoredFeature::datum(FeatureId(3), "Offset", definition)
        .with_owned_scalar("offset", 9)
        .unwrap();
    let mut program = FeatureSequence::default();
    program.append(feature).unwrap();
    program
}
fn encoded_owner(kind: Option<ParameterKind>) -> Vec<u8> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    if let Some(kind) = kind {
        parameters
            .insert_expression(9, "offset", kind, "2 cm")
            .unwrap();
    }
    let package = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n").unwrap();
    package
        .with_work_geometry(&owned_program(), 3)
        .unwrap()
        .with_parameters(&parameters, UnitsOfMeasure::default())
        .unwrap()
        .encode()
        .unwrap()
}
fn load_owner(
    encoded: &[u8],
    index: &mut DocumentIdentityIndex,
    generator: &mut FakeGuidSequence,
) -> Result<LoadedParameterPackage, PackageLoadError> {
    load_parameter_package(
        Path::new("owned.opd"),
        1,
        "alpha",
        index,
        generator,
        &mut reader(encoded),
    )
}
#[test]
fn invalid_owned_parameters_fail_before_identity_registration_and_entropy() {
    let mut index = DocumentIdentityIndex::default();
    let mut generator = source();
    for kind in [None, Some(USER_PARAM)] {
        let error = load_owner(&encoded_owner(kind), &mut index, &mut generator).unwrap_err();
        assert!(error.to_string().contains("existing Model parameter"));
        assert!(std::error::Error::source(&error).is_some());
        assert!(index.identity(1).is_none());
        assert_eq!(generator.next, 0);
    }
}
#[test]
fn valid_owned_parameters_load_with_their_stable_identity_and_formula() {
    let mut index = DocumentIdentityIndex::default();
    let mut generator = source();
    let loaded = load_owner(
        &encoded_owner(Some(MODEL_PARAM)),
        &mut index,
        &mut generator,
    )
    .unwrap();
    let feature = loaded.work_geometry.program.get(FeatureId(3)).unwrap();
    assert_eq!(feature.owned_scalars()[0].parameter, 9);
    assert_eq!(loaded.restored.parameters.get(9).unwrap().source, "2 cm");
    assert!(index.identity(1).is_some());
}
