// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::{DocumentSketchId, SketchPlacement},
    history::DocumentSketchCreation,
    sketch::SketchPlane,
};

fn request(name: &str) -> DocumentSketchCreation {
    DocumentSketchCreation {
        name: name.into(),
        placement: SketchPlacement::Fixed(SketchPlane::xy()),
        orientation: None,
        project_origin: false,
    }
}
fn sketch_name(session: &DocumentSession<RecordingObserver>, id: DocumentSketchId) -> String {
    session
        .snapshot()
        .sketches
        .get(id)
        .unwrap()
        .name()
        .to_owned()
}
#[test]
fn automatic_names_skip_explicit_names_and_do_not_rewind_after_undo() {
    let mut session = session(None);
    let explicit = session.create_sketch(&request("Sketch1")).unwrap();
    let first = session.create_automatic_sketch(&request("")).unwrap();
    let second = session.create_automatic_sketch(&request("")).unwrap();
    assert_eq!(sketch_name(&session, explicit), "Sketch1");
    assert_eq!(sketch_name(&session, first), "Sketch2");
    assert_eq!(sketch_name(&session, second), "Sketch3");
    session.undo().unwrap();
    assert!(session.snapshot().sketches.get(second).is_err());
    session.redo().unwrap();
    assert_eq!(sketch_name(&session, second), "Sketch3");
    session.undo().unwrap();
    let next = session.create_automatic_sketch(&request("")).unwrap();
    assert_eq!(sketch_name(&session, next), "Sketch4");
}
#[test]
fn rejected_projection_does_not_consume_the_next_name() {
    let mut session = session(None);
    let refused =
        session.create_automatic_sketch_projected(&request(""), |_| Err::<(), _>("refused"));
    assert!(refused.is_err());
    let created = session.create_automatic_sketch(&request("")).unwrap();
    assert_eq!(sketch_name(&session, created), "Sketch1");
}
#[test]
fn separate_documents_each_start_at_one() {
    let mut first = session(None);
    let mut second = session(None);
    let a = first.create_automatic_sketch(&request("")).unwrap();
    let b = second.create_automatic_sketch(&request("")).unwrap();
    assert_eq!(sketch_name(&first, a), "Sketch1");
    assert_eq!(sketch_name(&second, b), "Sketch1");
}

#[test]
fn aborted_transaction_does_not_reuse_an_allocated_name() {
    let mut session = session(None);
    session.begin("draft").unwrap();
    let discarded = session.create_automatic_sketch(&request("")).unwrap();
    assert_eq!(sketch_name(&session, discarded), "Sketch1");
    session.abort().unwrap();
    assert!(session.snapshot().sketches.get(discarded).is_err());
    let next = session.create_automatic_sketch(&request("")).unwrap();
    assert_eq!(sketch_name(&session, next), "Sketch2");
}
#[test]
fn reopened_document_skips_live_names_and_starts_a_fresh_sequence() {
    let mut original = session(None);
    original.create_automatic_sketch(&request("")).unwrap();
    original.create_sketch(&request("Sketch8")).unwrap();
    let snapshot = original.snapshot().clone();
    let ids = RecipeIdAllocator::restore(None, snapshot.parameters.ids()).unwrap();
    let history = DocumentEditHistory::new(snapshot, ids).unwrap();
    let mut reopened = DocumentSession::new(91, history, RecordingObserver::default());
    let created = reopened.create_automatic_sketch(&request("")).unwrap();
    assert_eq!(sketch_name(&reopened, created), "Sketch2");
}
