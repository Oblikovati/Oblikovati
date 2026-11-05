// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::feature::{
    AuthoredFeature, FeatureId, FeatureSequence,
    work_axis::WorkAxisReference,
    work_geometry::{DatumAngle, WorkGeometryDefinition, WorkPlaneLineAngle},
    work_plane::{WorkPlaneDefinitionRecipe, WorkPlaneOrigin, WorkPlaneReference},
};
use oblikovati_persistence::package::DocumentPackage;
fn angle_definition(angle: DatumAngle) -> WorkGeometryDefinition {
    WorkGeometryDefinition::AtAngleAboutAxis(WorkPlaneLineAngle {
        line: WorkAxisReference::OriginX,
        base: WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
        angle,
    })
}
fn program() -> FeatureSequence<WorkGeometryDefinition> {
    let mut program = FeatureSequence::default();
    for (id, angle) in [
        (1, DatumAngle::constant(-0.0).unwrap()),
        (2, DatumAngle::parameter(7)),
    ] {
        let definition = angle_definition(angle);
        program
            .append(AuthoredFeature::datum(FeatureId(id), "Swing", definition))
            .unwrap();
    }
    program.flip_plane_normal(FeatureId(2)).unwrap();
    program
}
#[test]
fn angular_sources_orientation_and_reservations_survive_package_round_trip() {
    let package =
        DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\nmodel: {}\n").unwrap();
    let program = program();
    let captured = package.with_work_planes(&program, 11).unwrap();
    let bytes = captured.encode().unwrap();
    let restored = DocumentPackage::decode(&bytes)
        .unwrap()
        .restore_work_planes()
        .unwrap();
    assert!(program.same_work_geometry_recipe(&restored.program));
    assert_eq!(restored.program.work_geometry_recipe().unwrap().version, 15);
    let mut ids = restored.ids;
    assert_eq!(ids.allocate().unwrap(), 12);
}
#[test]
fn nonfinite_decoded_angular_constants_cannot_restore_a_definition() {
    for invalid in ["!Constant .nan", "!Constant .inf", "!Constant -.inf"] {
        let angle: DatumAngle = serde_yaml_ng::from_str(invalid).unwrap();
        assert!(angle.validate_source().is_err());
        let WorkGeometryDefinition::AtAngleAboutAxis(source) = angle_definition(angle) else {
            unreachable!()
        };
        let mut recipe = angle_definition(DatumAngle::constant(0.0).unwrap()).recipe();
        recipe.definition = WorkPlaneDefinitionRecipe::AtAngleAboutAxis(source);
        let error = WorkGeometryDefinition::from_recipe(recipe).unwrap_err();
        assert!(error.to_string().contains("finite Angle in radians"));
    }
}
#[test]
fn malformed_angular_sources_and_extra_constructor_fields_do_not_decode() {
    for invalid in ["!Parameter text", "!Constant wrong", "!Other 1"] {
        assert!(serde_yaml_ng::from_str::<DatumAngle>(invalid).is_err());
    }
    let definition = angle_definition(DatumAngle::parameter(7));
    let mut source = serde_yaml_ng::to_value(definition.recipe()).unwrap();
    source["definition"]["AtAngleAboutAxis"]["unknown"] = true.into();
    assert!(
        serde_yaml_ng::from_value::<oblikovati_model::feature::work_plane::WorkPlaneRecipe>(source)
            .is_err()
    );
}
