// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::{MODEL_PARAM, USER_PARAM};
use oblikovati_model::{
    feature::{
        AuthoredFeature, FeatureId, FeatureSequence,
        work_geometry::{DatumLength, WorkGeometryDefinition},
        work_plane::{WorkPlaneOrigin, WorkPlaneReference, WorkPlaneResolver},
    },
    parameter::ParameterCollection,
    units::UnitsOfMeasure,
};
use oblikovati_persistence::package::DocumentPackage;

fn parameters() -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "4 cm")
        .unwrap();
    parameters
        .insert_expression(9, "offset", MODEL_PARAM, "width / 2")
        .unwrap();
    parameters
}
fn sequence() -> FeatureSequence<WorkGeometryDefinition> {
    let definition = WorkGeometryDefinition::Offset {
        base: WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
        distance: DatumLength::Parameter(9),
    };
    let feature = AuthoredFeature::datum(FeatureId(3), "Offset", definition)
        .with_owned_scalar("offset", 9)
        .unwrap();
    let mut sequence = FeatureSequence::default();
    sequence.append(feature).unwrap();
    sequence
}
fn stored() -> DocumentPackage {
    let original = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n").unwrap();
    original
        .with_work_geometry(&sequence(), 3)
        .unwrap()
        .with_parameters(&parameters(), UnitsOfMeasure::default())
        .unwrap()
}
#[test]
fn reopened_owned_scalar_retains_formula_and_updates_current_geometry() {
    let saved = stored();
    let reopened = DocumentPackage::decode(&saved.encode().unwrap()).unwrap();
    let sequence = reopened.restore_work_geometry().unwrap().program;
    let mut parameters = reopened.restore_document_parameters().unwrap().parameters;
    assert_eq!(parameters.get(9).unwrap().source, "width / 2");
    assert_eq!(parameters.drivers(9).unwrap(), [1]);
    assert_eq!(
        sequence.get(FeatureId(3)).unwrap().owned_scalars()[0].parameter,
        9
    );
    parameters.set_expression(1, "8 cm").unwrap();
    let evaluation = sequence.evaluate_work_geometry(&parameters).unwrap();
    assert_eq!(
        evaluation.plane(FeatureId(3)).unwrap().recipe().origin,
        [0.0, 0.0, 4.0]
    );
}
