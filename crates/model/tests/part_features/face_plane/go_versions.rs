// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::feature::{
    work_axis::WorkAxisReference,
    work_geometry::{DatumAngle, WorkPlaneLineAngle, WorkPlaneThroughPoint, WorkPointReference},
};

#[test]
fn go_face_based_definitions_require_version_14_and_preserve_the_source_dependency() {
    let base = WorkPlaneReference::GoPrismFace(WorkPlaneFaceReference::new(face_key(1)).unwrap());
    let definitions = [
        WorkPlaneDefinition::Offset {
            base: base.clone(),
            distance: WorkPlaneDistance::Constant(0.5),
        },
        WorkPlaneDefinition::ParallelThroughPoint(WorkPlaneThroughPoint {
            base: base.clone(),
            point: WorkPointReference::Origin,
        }),
        WorkPlaneDefinition::AtAngleAboutAxis(WorkPlaneLineAngle {
            base,
            line: WorkAxisReference::OriginX,
            angle: DatumAngle::constant(0.5).unwrap(),
        }),
    ];
    for definition in definitions {
        assert!(
            definition
                .reads()
                .keys()
                .contains(&FeatureReadKey::Feature(FeatureId(2)))
        );
        let recipe = definition.recipe();
        assert_eq!(recipe.version, 14);
        let restored = WorkPlaneDefinition::from_recipe(recipe.clone()).unwrap();
        assert!(definition.same_recipe(&restored));
        let mut old = recipe;
        old.version = 13;
        assert!(WorkPlaneDefinition::from_recipe(old).is_err());
    }
}
