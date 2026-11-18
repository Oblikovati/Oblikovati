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
impl PartFeatureEvaluator for NamedFaceEvaluator {
    type Body = f64;
    type Failure = &'static str;
    fn evaluate_blend(
        &mut self,
        kind: oblikovati_model::feature::part::PartBlendKind,
        _feature: oblikovati_model::feature::FeatureId,
        _body: &Self::Body,
        _edges: &[Vec<u8>],
        _distance: f64,
    ) -> Result<Self::Body, oblikovati_model::feature::part::PartFeatureFailure<Self::Failure>>
    {
        Err(kind.unsupported())
    }

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

#[test]
fn face_binding_requires_a_current_result_and_a_resolvable_planar_face() {
    let mut session = face_session();
    let mut slot = PartSlot::default();
    let reference = WorkPlaneFaceReference::new(face_key()).unwrap();
    assert!(
        slot.resolve_planar_face(&session, &NamedFaceEvaluator, &reference)
            .is_err()
    );
    publish_face(&mut session, &mut slot, 2.5);
    let frame = slot
        .resolve_planar_face(&session, &NamedFaceEvaluator, &reference)
        .unwrap();
    assert_eq!(frame.recipe().origin, [2., 0., 0.]);
    let other_key = SemanticReferenceName::new(
        "11111111-2222-3333-4444-555555555555".parse().unwrap(),
        1,
        1,
        "sketch-prism",
        "side",
        &[1, 102],
    )
    .unwrap()
    .encode()
    .unwrap();
    let other = WorkPlaneFaceReference::new(other_key).unwrap();
    assert!(
        slot.resolve_planar_face(&session, &NamedFaceEvaluator, &other)
            .is_err()
    );
    let missing_key = SemanticReferenceName::new(
        "11111111-2222-3333-4444-555555555555".parse().unwrap(),
        99,
        1,
        "sketch-prism",
        "side",
        &[1, 101],
    )
    .unwrap()
    .encode()
    .unwrap();
    let missing = WorkPlaneFaceReference::new(missing_key).unwrap();
    let error = slot
        .resolve_planar_face(&session, &NamedFaceEvaluator, &missing)
        .unwrap_err();
    assert!(error.contains("available unconsumed body"));
    edit_depth(&mut session, "4 cm");
    assert!(
        slot.resolve_planar_face(&session, &NamedFaceEvaluator, &reference)
            .is_err()
    );
    publish_face(&mut session, &mut slot, 4.5);
    assert_eq!(
        slot.resolve_planar_face(&session, &NamedFaceEvaluator, &reference)
            .unwrap()
            .recipe()
            .origin,
        [4., 0., 0.]
    );
}

#[test]
fn part_datum_snapshot_rejects_stale_and_foreign_sessions() {
    let mut session = face_session();
    let mut slot = PartSlot::default();
    publish_face(&mut session, &mut slot, 2.5);
    let datums = slot.current_datums(&session).unwrap();
    assert_eq!(
        datums
            .current(&session)
            .unwrap()
            .plane(FeatureId(2))
            .unwrap()
            .recipe()
            .origin,
        [2.5, 0., 0.]
    );
    let other = face_session();
    assert!(datums.current(&other).is_err());
    edit_depth(&mut session, "4 cm");
    assert!(datums.current(&session).is_err());
}

#[test]
fn recomputed_part_sketches_adopt_only_for_the_current_generation() {
    use oblikovati_model::{
        document_sketch::{PlacedSketchHealth, SketchEvaluation, SketchPlacement},
        history::DocumentSketchCreation,
    };
    let mut session = face_session();
    let mut slot = PartSlot::default();
    publish_face(&mut session, &mut slot, 2.5);
    let datums = slot.current(&session).unwrap().datum_evaluation();
    let request = DocumentSketchCreation {
        name: "Face sketch".into(),
        placement: SketchPlacement::WorkPlane(FeatureId(2)),
        orientation: None,
        project_origin: true,
    };
    let sketch = session
        .run("Create face sketch", |snapshot, ids| {
            snapshot.create_document_sketch_with_datums(&request, ids, &datums)
        })
        .unwrap();
    let undo_count = session.undo_labels().len();
    let stale = session
        .prepare_part_evaluation()
        .recompute_sketches(&mut NamedFaceEvaluator, Default::default())
        .unwrap();
    edit_depth(&mut session, "4 cm");
    assert!(session.adopt_recomputed_part_sketches(stale).is_err());
    let count_after_edit = session.undo_labels().len();
    assert_eq!(count_after_edit, undo_count + 1);
    let completed = session
        .prepare_part_evaluation()
        .recompute_sketches(&mut NamedFaceEvaluator, Default::default())
        .unwrap();
    assert!(session.adopt_recomputed_part_sketches(completed).unwrap());
    assert_eq!(session.undo_labels().len(), count_after_edit);
    publish_face(&mut session, &mut slot, 4.5);
    let datums = slot.current(&session).unwrap().datum_evaluation();
    let health = session
        .snapshot()
        .document_sketch_health_with_datums(sketch, &datums)
        .unwrap();
    assert!(
        matches!(
            health,
            PlacedSketchHealth::Current(SketchEvaluation::Numerical(_))
        ),
        "{health:?}"
    );
}

#[test]
fn part_sketch_refresh_rejects_foreign_history() {
    let session = face_session();
    let completed = session
        .prepare_part_evaluation()
        .recompute_sketches(&mut NamedFaceEvaluator, Default::default())
        .unwrap();
    let mut other = face_session();
    assert!(other.adopt_recomputed_part_sketches(completed).is_err());
}

fn add_face_sketch(
    session: &mut PartSession,
) -> oblikovati_model::document_sketch::DocumentSketchId {
    use oblikovati_model::{document_sketch::SketchPlacement, history::DocumentSketchCreation};
    let mut slot = PartSlot::default();
    publish_face(session, &mut slot, 2.5);
    let datums = slot.current(session).unwrap().datum_evaluation();
    let request = DocumentSketchCreation {
        name: "Face sketch".into(),
        placement: SketchPlacement::WorkPlane(FeatureId(2)),
        orientation: None,
        project_origin: true,
    };
    session
        .run("Create sketch", |snapshot, ids| {
            snapshot.create_document_sketch_with_datums(&request, ids, &datums)
        })
        .unwrap()
}
fn recompute_in_group(session: &mut PartSession) -> bool {
    let result = session
        .prepare_part_evaluation()
        .recompute_sketches(&mut NamedFaceEvaluator, Default::default())
        .unwrap();
    session.adopt_recomputed_part_sketches(result).unwrap()
}
#[test]
fn ordered_recompute_joins_transaction_and_abort_restores_state() {
    let mut session = face_session();
    let sketch = add_face_sketch(&mut session);
    let undo = session.undo_labels().len();
    let baseline = session.snapshot().sketches.clone();
    session.begin("Change depth").unwrap();
    edit_depth(&mut session, "4 cm");
    assert!(recompute_in_group(&mut session));
    let generation = session.generation();
    assert!(!recompute_in_group(&mut session));
    assert_eq!(session.generation(), generation);
    assert_eq!(session.undo_labels().len(), undo);
    session.abort().unwrap();
    assert!(session.snapshot().sketches.same_evaluated_state(&baseline));
    assert_eq!(session.undo_labels().len(), undo);
    assert!(session.snapshot().sketches.get(sketch).is_ok());
}
#[test]
fn ordered_recompute_joins_one_undo_step_and_rejects_stale_group_results() {
    let mut session = face_session();
    add_face_sketch(&mut session);
    let undo = session.undo_labels().len();
    session.begin("Change depth").unwrap();
    edit_depth(&mut session, "4 cm");
    let stale = session
        .prepare_part_evaluation()
        .recompute_sketches(&mut NamedFaceEvaluator, Default::default())
        .unwrap();
    edit_depth(&mut session, "5 cm");
    assert!(session.adopt_recomputed_part_sketches(stale).is_err());
    assert!(recompute_in_group(&mut session));
    session.end().unwrap();
    assert_eq!(session.undo_labels().len(), undo + 1);
    let mut slot = PartSlot::default();
    publish_face(&mut session, &mut slot, 5.5);
    session.undo().unwrap();
    publish_face(&mut session, &mut slot, 2.5);
    session.redo().unwrap();
    publish_face(&mut session, &mut slot, 5.5);
}

#[test]
fn sketch_recompute_alone_does_not_create_an_undo_step() {
    let mut session = face_session();
    add_face_sketch(&mut session);
    let undo = session.undo_labels().len();
    session.begin("Refresh only").unwrap();
    assert!(recompute_in_group(&mut session));
    session.end().unwrap();
    assert_eq!(session.undo_labels().len(), undo);
}

#[test]
fn moving_projected_point_refresh_does_not_create_an_undo_step() {
    use oblikovati_model::{
        document_sketch::SketchPointProjection,
        feature::work_geometry::{DatumLength, WorkPointReference},
        sketch::Sketch2D,
    };
    let mut session = face_session();
    let sketch = add_face_sketch(&mut session);
    let point_feature = session
        .run("Add source point", |snapshot, ids| {
            let id = ids.allocate_feature().unwrap();
            let point = WorkPlaneDefinition::Point([
                DatumLength::Constant(0.),
                DatumLength::Parameter(3),
                DatumLength::Constant(0.),
            ]);
            snapshot
                .work_geometry
                .append(AuthoredFeature::datum(id, "Moving point", point))?;
            Ok::<_, oblikovati_model::feature::FeatureSequenceError>(id)
        })
        .unwrap();
    let mut slot = PartSlot::default();
    publish_face(&mut session, &mut slot, 2.5);
    let datums = slot.current(&session).unwrap().datum_evaluation();
    let reference = WorkPointReference::Feature(point_feature);
    session
        .run("Project point", |snapshot, ids| {
            let projection = snapshot
                .project_datum_point_into_sketch_with_datums(sketch, reference, &datums)
                .unwrap();
            let anchor = ids.allocate_sketch_entity().unwrap();
            let mut geometry: Sketch2D = snapshot.sketches.get(sketch).unwrap().geometry().clone();
            geometry
                .add_point(anchor, projection.local_position)
                .unwrap();
            snapshot
                .sketches
                .replace_geometry(sketch, geometry)
                .unwrap();
            let mut bindings = snapshot
                .sketches
                .get(sketch)
                .unwrap()
                .point_projections()
                .to_vec();
            bindings.push(SketchPointProjection { anchor, reference });
            snapshot
                .sketches
                .set_point_projections(sketch, bindings)
                .unwrap();
            Ok::<_, std::convert::Infallible>(())
        })
        .unwrap();
    let completed = session
        .prepare_part_evaluation()
        .recompute_sketches(&mut NamedFaceEvaluator, Default::default())
        .unwrap();
    session.adopt_recomputed_part_sketches(completed).unwrap();
    edit_depth(&mut session, "4 cm");
    let undo = session.undo_labels().len();
    let before = session.snapshot().sketches.clone();
    session.begin("Abort refresh").unwrap();
    assert!(recompute_in_group(&mut session));
    assert!(!session.snapshot().sketches.same_recipe(&before));
    session.abort().unwrap();
    assert!(session.snapshot().sketches.same_evaluated_state(&before));
    session.begin("Refresh only").unwrap();
    session.begin("Nested refresh").unwrap();
    assert!(recompute_in_group(&mut session));
    assert!(!session.snapshot().sketches.same_recipe(&before));
    session.end().unwrap();
    session.end().unwrap();
    assert_eq!(session.undo_labels().len(), undo);
}

struct FakePartSketchContinuity;
impl oblikovati_session::SketchContinuityResolver for FakePartSketchContinuity {
    fn resolve(
        &self,
        history: &DocumentEditHistory,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
    ) -> Result<(), String> {
        let (part, _) = history
            .snapshot()
            .recompute_part_with_sketches(&mut NamedFaceEvaluator, Default::default())
            .map_err(|error| error.to_string())?;
        history
            .snapshot()
            .resolve_sketch_plane_with_datums(sketch, &part.datum_evaluation())
            .map(|_| ())
            .map_err(|error| error.to_string())
    }
}
#[test]
fn face_sketch_continuity_tracks_published_source_failure() {
    let mut session = face_session();
    let sketch = add_face_sketch(&mut session);
    session.set_sketch_continuity_resolver(Box::new(FakePartSketchContinuity));
    let watch = session.watch_sketch_continuity(sketch).unwrap();
    edit_depth(&mut session, "4 cm");
    assert!(watch.is_live());
    session.undo().unwrap();
    assert!(watch.is_live());
    session.redo().unwrap();
    assert!(watch.is_live());
    edit_depth(&mut session, "0 cm");
    assert!(!watch.is_live());
    edit_depth(&mut session, "4 cm");
    assert!(!watch.is_live());
    let restored = session.watch_sketch_continuity(sketch).unwrap();
    assert!(restored.is_live());
}

#[path = "face_offset/creation.rs"]
mod creation;

#[path = "face_offset/reuse.rs"]
mod reuse;
