// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn face_sketch_creation_checks_part_generation_and_solves_projected_origin() {
    use oblikovati_model::{
        document_sketch::{PlacedSketchHealth, SketchEvaluation, SketchPlacement},
        history::DocumentSketchCreation,
    };
    let mut session = face_session();
    let mut slot = PartSlot::default();
    publish_face(&mut session, &mut slot, 2.5);
    let datums = slot.current_datums(&session).unwrap();
    let request = DocumentSketchCreation {
        name: String::new(),
        placement: SketchPlacement::WorkPlane(FeatureId(2)),
        orientation: None,
        project_origin: true,
    };
    let sketch = session
        .create_automatic_sketch_with_part_datums(&request, &datums)
        .unwrap();
    assert!(
        session
            .create_automatic_sketch_with_part_datums(&request, &datums)
            .is_err()
    );
    publish_face(&mut session, &mut slot, 2.5);
    let current = slot.current_datums(&session).unwrap();
    let health = session
        .snapshot()
        .document_sketch_health_with_datums(sketch, current.current(&session).unwrap())
        .unwrap();
    assert!(
        matches!(
            health,
            PlacedSketchHealth::Current(SketchEvaluation::Numerical(_))
        ),
        "{health:?}"
    );
    session.undo().unwrap();
    assert_eq!(session.snapshot().sketches.sketches().len(), 1);
    session.redo().unwrap();
    assert_eq!(session.snapshot().sketches.sketches().len(), 2);
}

#[test]
fn face_sketch_reply_refusal_is_atomic_and_named_creation_preserves_name() {
    use oblikovati_model::{document_sketch::SketchPlacement, history::DocumentSketchCreation};
    let mut session = face_session();
    let mut slot = PartSlot::default();
    publish_face(&mut session, &mut slot, 2.5);
    let datums = slot.current_datums(&session).unwrap();
    let request = DocumentSketchCreation {
        name: "Named face sketch".into(),
        placement: SketchPlacement::WorkPlane(FeatureId(2)),
        orientation: None,
        project_origin: true,
    };
    let before = session.generation();
    let rejected = session.create_sketch_with_part_datums_projected(&request, &datums, |result| {
        assert!(result.is_ok());
        Err::<(), _>("reply refused")
    });
    assert!(rejected.is_err());
    assert_eq!(session.generation(), before);
    assert_eq!(session.snapshot().sketches.sketches().len(), 1);
    assert!(session.undo_labels().is_empty());
    let created = session
        .create_sketch_with_part_datums_projected(&request, &datums, |result| result)
        .unwrap();
    assert_eq!(
        session.snapshot().sketches.get(created).unwrap().name(),
        "Named face sketch"
    );
}

struct UnusedCircularConstruction;
impl oblikovati_model::sketch::SketchCircleConstruction for UnusedCircularConstruction {
    fn arc_direction(&self, _: [[f64; 2]; 3]) -> Result<bool, Box<dyn std::error::Error>> {
        Err("unexpected arc construction".into())
    }
    fn circumcircle(
        &self,
        _: [[f64; 2]; 3],
        _: oblikovati_model::sketch::CircumcircleOptions,
    ) -> Result<oblikovati_model::sketch::PlanarCircumcircle, Box<dyn std::error::Error>> {
        Err("unexpected circle construction".into())
    }
    fn verify(
        &self,
        _: [[f64; 2]; 3],
        _: oblikovati_model::sketch::SolvedCircularSeed,
        _: oblikovati_model::sketch::CircumcircleOptions,
    ) -> Result<(), Box<dyn std::error::Error>> {
        Err("unexpected circular verification".into())
    }
}

#[test]
fn face_sketch_placement_previews_locks_and_rejects_stale_datums() {
    use oblikovati_model::{
        document_sketch::SketchPlacement,
        history::{DocumentSketchCreation, SketchPlacementFieldLock},
        sketch::SketchPlacementRecipe,
    };
    let mut session = face_session();
    let mut slot = PartSlot::default();
    publish_face(&mut session, &mut slot, 2.5);
    let datums = slot.current_datums(&session).unwrap();
    let request = DocumentSketchCreation {
        name: "Face sketch".into(),
        placement: SketchPlacement::WorkPlane(FeatureId(2)),
        orientation: None,
        project_origin: true,
    };
    let sketch = session
        .create_sketch_with_part_datums_projected(&request, &datums, |result| result)
        .unwrap();
    publish_face(&mut session, &mut slot, 2.5);
    let datums = slot.current_datums(&session).unwrap();
    let recipe = SketchPlacementRecipe::straight_slot([[10., 10.], [20., 10.]], 2.).unwrap();
    let lock = |expression: &str| {
        Some(SketchPlacementFieldLock {
            reference: false,
            expression: expression.into(),
            text_point: None,
        })
    };
    let locks = [lock("10 cm"), None, lock("2 cm")];
    let before = session.generation();
    let preview = session
        .preview_sketch_placement_with_datums(
            sketch,
            &recipe,
            &locks,
            &UnusedCircularConstruction,
            &datums,
        )
        .unwrap();
    assert_eq!(session.generation(), before);
    let report = session
        .create_sketch_placement_with_datums(
            sketch,
            &recipe,
            &locks,
            &UnusedCircularConstruction,
            &datums,
        )
        .unwrap();
    assert_eq!(report.dimensions.len(), 2);
    assert_eq!(
        preview.geometry,
        *session.snapshot().sketches.get(sketch).unwrap().geometry()
    );
    let generation = session.generation();
    assert!(
        session
            .create_sketch_placement_with_datums(
                sketch,
                &recipe,
                &locks,
                &UnusedCircularConstruction,
                &datums
            )
            .is_err()
    );
    assert_eq!(session.generation(), generation);
    session.undo().unwrap();
    assert!(
        session
            .snapshot()
            .sketches
            .get(sketch)
            .unwrap()
            .geometry()
            .lines()
            .is_empty()
    );
    session.redo().unwrap();
    assert_eq!(
        session
            .snapshot()
            .sketches
            .get(sketch)
            .unwrap()
            .geometry()
            .lines()
            .len(),
        3
    );
}

#[test]
fn direct_face_creation_requires_generation_checked_prospective_support() {
    use oblikovati_model::{document_sketch::SketchPlacement, history::DocumentSketchCreation};
    let mut session = face_session();
    let mut slot = PartSlot::default();
    publish_face(&mut session, &mut slot, 2.5);
    let face = WorkPlaneFaceReference::new(face_key()).unwrap();
    let token = slot
        .current_datums_for_face(&session, &NamedFaceEvaluator, &face)
        .unwrap();
    let request = DocumentSketchCreation {
        name: "Direct".into(),
        placement: SketchPlacement::Face(face.clone()),
        orientation: None,
        project_origin: true,
    };
    let sketch = session
        .create_sketch_with_part_datums_projected(&request, &token, |result| result)
        .unwrap();
    assert!(
        session
            .create_sketch_with_part_datums_projected(&request, &token, |result| result)
            .is_err()
    );
    publish_face(&mut session, &mut slot, 2.5);
    let current = slot.current_datums(&session).unwrap();
    let frame = session
        .snapshot()
        .resolve_sketch_plane_with_datums(sketch, current.current(&session).unwrap())
        .unwrap();
    assert_eq!(frame.recipe().origin, [2., 0., 0.]);
    session.undo().unwrap();
    assert!(session.snapshot().sketches.get(sketch).is_err());
    session.redo().unwrap();
    publish_face(&mut session, &mut slot, 2.5);
    let current = slot.current_datums(&session).unwrap();
    assert!(
        session
            .snapshot()
            .resolve_sketch_plane_with_datums(sketch, current.current(&session).unwrap())
            .is_ok()
    );
}
