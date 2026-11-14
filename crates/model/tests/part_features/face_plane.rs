// SPDX-License-Identifier: GPL-2.0-only
//! Ordered replay of a work plane based on a named face.
use super::*;
use oblikovati_model::{
    feature::{
        FeatureEvaluationStatus, FeatureReadKey,
        part::SketchPrismEvaluator,
        sketch_prism::ResolvedSketchPrism,
        work_plane::{WorkPlaneDistance, WorkPlaneFaceReference, WorkPlaneReference},
    },
    identity::SemanticReferenceName,
};

fn face_key(kind: u32) -> Vec<u8> {
    SemanticReferenceName::new(
        "11111111-2222-3333-4444-555555555555".parse().unwrap(),
        2,
        kind,
        "sketch-prism",
        "side",
        &[1, 101],
    )
    .unwrap()
    .encode()
    .unwrap()
}
#[derive(Default)]
struct FaceEvaluator {
    fail_body: bool,
    fail_face: bool,
}
impl SketchPrismEvaluator for FaceEvaluator {
    type Body = ();
    type Failure = &'static str;
    fn verify_body_reference(&self, _: &(), key: &[u8]) -> Result<(), String> {
        if key == face_key(4) {
            return Ok(());
        }
        Err(format!("body key {key:?}: expected the named test body"))
    }
    fn evaluate_prism(
        &mut self,
        _: FeatureId,
        _: ResolvedSketchPrism<'_>,
    ) -> Result<(), Self::Failure> {
        if self.fail_body {
            return Err("prism refused");
        }
        Ok(())
    }
    fn resolve_planar_face(&self, _: &(), key: &[u8]) -> Result<SketchPlane, String> {
        if self.fail_face || key != face_key(1) {
            return Err(format!("face key {key:?}: expected current face"));
        }
        SketchPlane::new([1., 0., 0.], [0., 1., 0.], [0., 0., 1.], Default::default())
            .map_err(|e| e.to_string())
    }
}
fn face_document() -> DocumentSnapshot {
    let mut owner = part_owner();
    append_prism(&mut owner);
    let mut snapshot = owner.snapshot().clone();
    let source = WorkPlaneFaceReference::new(face_key(1)).unwrap();
    let definition = WorkPlaneDefinition::Offset {
        base: WorkPlaneReference::Face(source),
        distance: WorkPlaneDistance::Constant(0.5),
    };
    let entry = AuthoredFeature::datum(FeatureId(3), "Face offset", definition);
    assert!(
        entry
            .reads()
            .keys()
            .contains(&FeatureReadKey::Feature(FeatureId(2)))
    );
    snapshot.work_geometry.append(entry).unwrap();
    snapshot
}
#[test]
fn face_offset_replays_through_current_named_body_and_round_trips_recipe() {
    let snapshot = face_document();
    let mut edited = snapshot.clone();
    assert!(edited.work_geometry.remove(FeatureId(2)).is_err());
    assert!(edited.work_geometry.reorder(FeatureId(3), 1).is_err());
    let result = snapshot
        .evaluate_part_features(&mut FaceEvaluator::default())
        .unwrap();
    assert_eq!(
        result.plane(FeatureId(3)).unwrap().recipe().origin,
        [1.5, 0., 0.]
    );
    let recipe = snapshot.work_geometry.part_recipe().unwrap();
    assert_eq!(recipe.version, 18);
    let mut older = recipe.clone();
    older.version = 17;
    assert!(FeatureSequence::from_part_recipe(&older).is_err());
    let encoded = serde_json::to_string(&recipe).unwrap();
    let restored =
        FeatureSequence::from_part_recipe(&serde_json::from_str(&encoded).unwrap()).unwrap();
    assert!(snapshot.work_geometry.same_work_geometry_recipe(&restored));
}
#[test]
fn failed_producer_blocks_face_plane_and_independent_work_continues() {
    let mut snapshot = face_document();
    snapshot
        .work_geometry
        .append(AuthoredFeature::datum(
            FeatureId(4),
            "Independent",
            WorkPlaneDefinition::Point([WorkPlaneDistance::Constant(0.); 3]),
        ))
        .unwrap();
    let result = snapshot
        .evaluate_part_features(&mut FaceEvaluator {
            fail_body: true,
            fail_face: false,
        })
        .unwrap();
    assert!(result.plane(FeatureId(3)).is_err());
    assert!(
        matches!(&result.health()[2].status, FeatureEvaluationStatus::Blocked { drivers } if drivers == &[FeatureId(2)])
    );
    assert_eq!(result.point(FeatureId(4)).unwrap(), [0.; 3]);
}
#[test]
fn missing_face_fails_locally_and_replay_recovers_without_old_output() {
    let snapshot = face_document();
    let failed = snapshot
        .evaluate_part_features(&mut FaceEvaluator {
            fail_body: false,
            fail_face: true,
        })
        .unwrap();
    assert!(failed.plane(FeatureId(3)).is_err());
    assert!(matches!(
        failed.health()[2].status,
        FeatureEvaluationStatus::Failed(_)
    ));
    let recovered = snapshot
        .evaluate_part_features(&mut FaceEvaluator::default())
        .unwrap();
    assert_eq!(
        recovered.plane(FeatureId(3)).unwrap().recipe().origin,
        [1.5, 0., 0.]
    );
    assert!(failed.plane(FeatureId(3)).is_err());
}
#[test]
fn consumed_body_cannot_satisfy_a_later_face_reference() {
    use oblikovati_model::feature::part::{DeleteBodyDefinition, PartFeatureFailure};
    let mut owner = part_owner();
    append_prism(&mut owner);
    let mut snapshot = owner.snapshot().clone();
    let delete = WorkPlaneDefinition::DeleteBody(DeleteBodyDefinition::new(face_key(4)).unwrap());
    snapshot
        .work_geometry
        .append(AuthoredFeature::datum(FeatureId(3), "Delete body", delete))
        .unwrap();
    let plane = WorkPlaneDefinition::Offset {
        base: WorkPlaneReference::Face(WorkPlaneFaceReference::new(face_key(1)).unwrap()),
        distance: WorkPlaneDistance::Constant(0.5),
    };
    snapshot
        .work_geometry
        .append(AuthoredFeature::datum(FeatureId(4), "Face offset", plane))
        .unwrap();
    let result = snapshot
        .evaluate_part_features(&mut FaceEvaluator::default())
        .unwrap();
    assert!(result.plane(FeatureId(4)).is_err());
    assert!(
        matches!(&result.health()[3].status, FeatureEvaluationStatus::Failed(PartFeatureFailure::Datum(cause)) if cause.to_string().contains("unconsumed"))
    );
}
#[test]
fn face_reference_refuses_malformed_or_wrong_kind_keys() {
    assert!(WorkPlaneFaceReference::new(vec![0]).is_err());
    assert!(WorkPlaneFaceReference::new(face_key(4)).is_err());
    assert!(serde_json::from_str::<WorkPlaneFaceReference>("[0]").is_err());
    let wrong_kind = serde_json::to_string(&face_key(4)).unwrap();
    assert!(serde_json::from_str::<WorkPlaneFaceReference>(&wrong_kind).is_err());
}

#[test]
fn every_face_based_plane_recipe_requires_the_new_version_and_tracks_its_producer() {
    use oblikovati_model::feature::{
        work_axis::WorkAxisReference,
        work_geometry::{
            DatumAngle, WorkPlaneLineAngle, WorkPlaneThroughPoint, WorkPointReference,
        },
    };
    let base = WorkPlaneReference::Face(WorkPlaneFaceReference::new(face_key(1)).unwrap());
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
        assert_eq!(recipe.version, 13);
        let encoded = serde_json::to_string(&recipe).unwrap();
        let reopened =
            WorkPlaneDefinition::from_recipe(serde_json::from_str(&encoded).unwrap()).unwrap();
        assert!(definition.same_recipe(&reopened));
        let mut old = recipe;
        old.version = 12;
        assert!(WorkPlaneDefinition::from_recipe(old).is_err());
    }
}

#[test]
fn datum_view_uses_the_same_part_replay_geometry_and_health() {
    let snapshot = face_document();
    let replay = snapshot
        .evaluate_part_features(&mut FaceEvaluator::default())
        .unwrap();
    let datums = replay.datum_evaluation();
    assert_eq!(
        datums.plane(FeatureId(3)).unwrap().recipe().origin,
        [1.5, 0., 0.]
    );
    assert!(matches!(
        datums
            .health
            .iter()
            .find(|record| record.feature == FeatureId(3))
            .unwrap()
            .status,
        FeatureEvaluationStatus::Healthy
    ));

    let replay = snapshot
        .evaluate_part_features(&mut FaceEvaluator {
            fail_body: true,
            fail_face: false,
        })
        .unwrap();
    let datums = replay.datum_evaluation();
    assert!(datums.plane(FeatureId(3)).is_err());
    assert!(
        matches!(&datums.health.iter().find(|record| record.feature == FeatureId(3)).unwrap().status, FeatureEvaluationStatus::Blocked { drivers } if drivers == &[FeatureId(2)])
    );

    let replay = snapshot
        .evaluate_part_features(&mut FaceEvaluator {
            fail_body: false,
            fail_face: true,
        })
        .unwrap();
    let datums = replay.datum_evaluation();
    assert!(
        matches!(&datums.health.iter().find(|record| record.feature == FeatureId(3)).unwrap().status, FeatureEvaluationStatus::Failed(error) if error.to_string().contains("expected current face"))
    );
}
