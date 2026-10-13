// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::{DocumentSketchId, SketchPlacement},
    history::DocumentSketchCreation,
    sketch::{Sketch2D, SketchError, SketchPlane},
};
#[test]
fn sketch_continuity_cannot_be_revived_by_redo_and_new_watch_is_independent() {
    let mut session = session(None);
    let sketch = session
        .create_sketch(&DocumentSketchCreation {
            name: "Observed".into(),
            placement: SketchPlacement::Fixed(SketchPlane::xy()),
            orientation: None,
            project_origin: false,
        })
        .unwrap();
    let watch = session.watch_sketch_continuity(sketch).unwrap();
    let generation = session.generation();
    assert_eq!(watch.sketch(), sketch);
    assert!(watch.is_live());
    assert_eq!(session.generation(), generation);
    session.undo().unwrap();
    session.redo().unwrap();
    assert!(!watch.is_live());
    let restored = session.watch_sketch_continuity(sketch).unwrap();
    assert!(restored.is_live());
    drop(session);
    assert!(!restored.is_live());
}
#[test]
fn refused_removal_preserves_continuity_and_published_removal_invalidates_clones() {
    let mut refused = sketch_fixture::owned_sketch_session(
        Some(EVENT_TRANSACTION_COMMITTED),
        "Observed",
        Sketch2D::default(),
    );
    let first = refused
        .watch_sketch_continuity(DocumentSketchId(1))
        .unwrap();
    assert!(
        refused
            .run("Remove", |snapshot, _| snapshot
                .sketches
                .remove(DocumentSketchId(1)))
            .is_err()
    );
    assert!(first.is_live());
    let mut accepted = sketch_fixture::owned_sketch_session(None, "Observed", Sketch2D::default());
    let second = accepted
        .watch_sketch_continuity(DocumentSketchId(1))
        .unwrap();
    let clone = second.clone();
    accepted
        .run("Remove", |snapshot, _| {
            snapshot.sketches.remove(DocumentSketchId(1))
        })
        .unwrap();
    accepted.undo().unwrap();
    assert!(!second.is_live() && !clone.is_live());
    refused.close();
    assert!(!first.is_live());
}
#[test]
fn missing_watch_is_refused_and_retired_readers_do_not_affect_live_readers() {
    let mut session = sketch_fixture::owned_sketch_session(None, "Observed", Sketch2D::default());
    assert!(
        session
            .watch_sketch_continuity(DocumentSketchId(999))
            .unwrap_err()
            .to_string()
            .contains("999")
    );
    drop(
        session
            .watch_sketch_continuity(DocumentSketchId(1))
            .unwrap(),
    );
    let live = session
        .watch_sketch_continuity(DocumentSketchId(1))
        .unwrap();
    drop(
        session
            .watch_sketch_continuity(DocumentSketchId(1))
            .unwrap(),
    );
    session
        .run("Rename", |snapshot, _| {
            snapshot.sketches.rename(DocumentSketchId(1), "Renamed")
        })
        .unwrap();
    assert!(live.is_live());
    session.begin("Unchanged").unwrap();
    session
        .edit(|_: &mut DocumentSnapshot, _| Ok::<_, SketchError>(()))
        .unwrap();
    session.abort().unwrap();
    assert!(live.is_live());
}
