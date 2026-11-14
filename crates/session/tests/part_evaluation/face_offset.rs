// SPDX-License-Identifier: GPL-2.0-only
//! Face-based plane results follow the session generation and undo history.
use super::*;
use oblikovati_model::{
    feature::{
        AuthoredFeature,
        work_plane::{
            WorkPlaneDefinition, WorkPlaneDistance, WorkPlaneFaceReference, WorkPlaneReference,
            WorkPlaneResolver,
        },
    },
    identity::SemanticReferenceName,
    sketch::SketchPlane,
};

struct NamedFaceEvaluator;
impl SketchPrismEvaluator for NamedFaceEvaluator {
    type Body = f64;
    type Failure = &'static str;
    fn verify_body_reference(&self, _: &f64, key: &[u8]) -> Result<(), String> {
        Err(format!("body key {key:?}: expected no body deletion"))
    }
    fn evaluate_prism(
        &mut self,
        _: FeatureId,
        input: ResolvedSketchPrism<'_>,
    ) -> Result<f64, Self::Failure> {
        Ok(input.displacement[2])
    }
    fn resolve_planar_face(&self, body: &f64, key: &[u8]) -> Result<SketchPlane, String> {
        if key != face_key() {
            return Err(format!("face key {key:?}: expected current face"));
        }
        SketchPlane::new(
            [*body, 0., 0.],
            [0., 1., 0.],
            [0., 0., 1.],
            Default::default(),
        )
        .map_err(|cause| cause.to_string())
    }
}
fn face_key() -> Vec<u8> {
    SemanticReferenceName::new(
        "11111111-2222-3333-4444-555555555555".parse().unwrap(),
        1,
        1,
        "sketch-prism",
        "side",
        &[1, 101],
    )
    .unwrap()
    .encode()
    .unwrap()
}
fn face_session() -> PartSession {
    let mut document = fixture::selected_prism_document();
    let definition = WorkPlaneDefinition::Offset {
        base: WorkPlaneReference::Face(WorkPlaneFaceReference::new(face_key()).unwrap()),
        distance: WorkPlaneDistance::Constant(0.5),
    };
    document
        .work_geometry
        .append(AuthoredFeature::datum(
            FeatureId(2),
            "Face offset",
            definition,
        ))
        .unwrap();
    let ids = RecipeIdAllocator::restore(None, document.parameters.ids()).unwrap();
    DocumentSession::new(
        7,
        DocumentEditHistory::new(document, ids).unwrap(),
        RecordingPartObserver::default(),
    )
}
fn publish_face(session: &mut PartSession, slot: &mut PartSlot, expected: f64) {
    let evaluated = session
        .prepare_part_evaluation()
        .evaluate(&mut NamedFaceEvaluator)
        .unwrap();
    session.adopt_part_evaluation(evaluated, slot).unwrap();
    let plane = slot.current(session).unwrap().plane(FeatureId(2)).unwrap();
    assert_eq!(plane.recipe().origin, [expected, 0., 0.]);
}
#[test]
fn named_face_offset_replays_after_session_edit_undo_and_redo() {
    let mut session = face_session();
    let mut slot = PartSlot::default();
    publish_face(&mut session, &mut slot, 2.5);
    edit_depth(&mut session, "4 cm");
    assert!(slot.current(&session).is_err());
    publish_face(&mut session, &mut slot, 4.5);
    session.undo().unwrap();
    assert!(slot.current(&session).is_err());
    publish_face(&mut session, &mut slot, 2.5);
    session.redo().unwrap();
    assert!(slot.current(&session).is_err());
    publish_face(&mut session, &mut slot, 4.5);
}
