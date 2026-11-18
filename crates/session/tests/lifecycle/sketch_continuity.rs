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

struct PermissiveSketchResolver;
impl oblikovati_session::SketchContinuityResolver for PermissiveSketchResolver {
    fn resolve(
        &self,
        _: &DocumentEditHistory,
        sketches: &[DocumentSketchId],
    ) -> Vec<Result<(), String>> {
        sketches.iter().map(|_| Ok(())).collect()
    }
}
#[test]
fn injected_resolver_cannot_keep_a_removed_sketch_live() {
    let mut session = sketch_fixture::owned_sketch_session(None, "Observed", Sketch2D::default());
    session.set_sketch_continuity_resolver(std::boxed::Box::new(PermissiveSketchResolver));
    let watch = session
        .watch_sketch_continuity(DocumentSketchId(1))
        .unwrap();
    session
        .run("Remove", |snapshot, _| {
            snapshot.sketches.remove(DocumentSketchId(1))
        })
        .unwrap();
    session.undo().unwrap();
    assert!(!watch.is_live());
}

struct CountingSketchResolver {
    calls: std::sync::Arc<std::sync::atomic::AtomicUsize>,
    mode: std::sync::Arc<std::sync::atomic::AtomicUsize>,
    rejected: DocumentSketchId,
}
impl oblikovati_session::SketchContinuityResolver for CountingSketchResolver {
    fn resolve(
        &self,
        _: &DocumentEditHistory,
        sketches: &[DocumentSketchId],
    ) -> Vec<Result<(), String>> {
        use std::sync::atomic::Ordering;
        self.calls.fetch_add(1, Ordering::SeqCst);
        let mode = self.mode.load(Ordering::SeqCst);
        if mode == 2 {
            return Vec::new();
        }
        sketches
            .iter()
            .map(|sketch| {
                if mode == 1 && *sketch == self.rejected {
                    Err("unavailable sketch".into())
                } else {
                    Ok(())
                }
            })
            .collect()
    }
}

#[test]
fn continuity_refresh_batches_watches_and_rejects_invalid_results() {
    use std::sync::{
        Arc,
        atomic::{AtomicUsize, Ordering},
    };
    let mut session = sketch_fixture::owned_sketch_session(None, "First", Sketch2D::default());
    let second = session
        .create_sketch(&DocumentSketchCreation {
            name: "Second".into(),
            placement: SketchPlacement::Fixed(SketchPlane::xy()),
            orientation: None,
            project_origin: false,
        })
        .unwrap();
    let calls = Arc::new(AtomicUsize::new(0));
    let mode = Arc::new(AtomicUsize::new(0));
    session.set_sketch_continuity_resolver(std::boxed::Box::new(CountingSketchResolver {
        calls: calls.clone(),
        mode: mode.clone(),
        rejected: second,
    }));
    let first_watch = session
        .watch_sketch_continuity(DocumentSketchId(1))
        .unwrap();
    let second_watch = session.watch_sketch_continuity(second).unwrap();
    let duplicate = session.watch_sketch_continuity(second).unwrap();
    calls.store(0, Ordering::SeqCst);
    session
        .run("Rename first", |snapshot, _| {
            snapshot.sketches.rename(DocumentSketchId(1), "Renamed")
        })
        .unwrap();
    assert_eq!(calls.load(Ordering::SeqCst), 1);
    assert!(first_watch.is_live() && second_watch.is_live() && duplicate.is_live());
    mode.store(1, Ordering::SeqCst);
    session
        .run("Rename second", |snapshot, _| {
            snapshot.sketches.rename(second, "Other")
        })
        .unwrap();
    assert_eq!(calls.load(Ordering::SeqCst), 2);
    assert!(first_watch.is_live());
    assert!(!second_watch.is_live() && !duplicate.is_live());
    mode.store(2, Ordering::SeqCst);
    session
        .run("Rename again", |snapshot, _| {
            snapshot.sketches.rename(DocumentSketchId(1), "Final")
        })
        .unwrap();
    assert!(!first_watch.is_live());
}
