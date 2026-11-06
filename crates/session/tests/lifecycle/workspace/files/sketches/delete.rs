// SPDX-License-Identifier: GPL-2.0-only
use super::automatic::stale_session;
use super::recompute::saved_driven_sketch;
use super::*;
use oblikovati_model::document_sketch::DocumentSketches;

#[test]
fn default_session_deletion_and_group_abort_preserve_original_state() {
    let (mut session, sketch) = stale_session(Default::default());
    let before = session.snapshot().sketches.clone();
    session.begin("Delete group").unwrap();
    assert_eq!(session.delete_sketch(sketch).unwrap().sketch, sketch);
    assert!(session.snapshot().sketches.sketches().is_empty());
    session.abort().unwrap();
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    assert!(session.undo_labels().is_empty());
}

#[test]
fn rejected_projection_and_missing_sketch_leave_the_owner_unchanged() {
    let (mut session, sketch) = stale_session(Default::default());
    let before = session.snapshot().sketches.clone();
    let result = session.delete_sketch_projected(sketch, |report| {
        assert_eq!(report.unwrap().sketch, sketch);
        Err::<(), _>("reply refused")
    });
    assert!(result.unwrap_err().to_string().contains("reply refused"));
    assert!(session.delete_sketch(DocumentSketchId(99)).is_err());
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    assert!(session.undo_labels().is_empty());
}

#[test]
fn file_owned_deletion_persists_without_releasing_reservations() {
    let (mut host, mut source, document, sketch) = saved_driven_sketch();
    let mut commands = host.commands(document).unwrap();
    commands.delete_sketch(sketch).unwrap();
    assert!(host.workspace().needs_save(document).unwrap());
    host.commands(document).unwrap().undo().unwrap();
    assert!(!host.workspace().needs_save(document).unwrap());
    host.commands(document).unwrap().redo().unwrap();
    host.save(document, "alpha", &mut source).unwrap();
    host.close_discarding_changes(document).unwrap();
    let reopened = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    assert!(deletion_sketches(&host, reopened).sketches().is_empty());
    let next = host
        .commands(reopened)
        .unwrap()
        .run("Next", add_sketch)
        .unwrap();
    assert!(next.0 > sketch.0);
}

#[test]
fn workspace_projection_rejects_deletion_before_owner_adoption() {
    let (mut host, _, document, sketch) = saved_driven_sketch();
    let before = deletion_sketches(&host, document).clone();
    let result = host
        .commands(document)
        .unwrap()
        .delete_sketch_projected(sketch, |report| {
            assert!(report.is_ok());
            Err::<(), _>("cannot encode reply")
        });
    assert!(result.is_err());
    let after = deletion_sketches(&host, document);
    assert!(before.same_evaluated_state(after));
    assert!(!host.workspace().needs_save(document).unwrap());
}

fn deletion_sketches(host: &Host, document: u64) -> &DocumentSketches {
    &host
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
}

#[test]
fn deletion_does_not_solve_or_change_an_unrelated_pending_sketch() {
    let (mut session, sketch) = super::selected::session_with_unverified_neighbor();
    let neighbor = DocumentSketchId(99);
    let before = session.snapshot().sketches.get(neighbor).unwrap().clone();
    session.delete_sketch(sketch).unwrap();
    assert_eq!(&before, session.snapshot().sketches.get(neighbor).unwrap());
}

#[test]
fn copied_deleted_sketches_keep_their_reserved_identity_namespace() {
    let (mut host, mut source, document, sketch) = saved_driven_sketch();
    let original = host.workspace().identity(document).unwrap().clone();
    let mut commands = host.commands(document).unwrap();
    commands.delete_sketch(sketch).unwrap();
    let copied = reopen_deletion_copy(&mut host, document, &mut source);
    assert!(deletion_sketches(&host, copied).sketches().is_empty());
    let next = host
        .commands(copied)
        .unwrap()
        .run("Next copy", add_sketch)
        .unwrap();
    assert!(next.0 > sketch.0);
    assert_eq!(host.workspace().identity(document).unwrap(), &original);
    assert!(host.workspace().needs_save(document).unwrap());
}

fn reopen_deletion_copy(host: &mut Host, document: u64, source: &mut SequenceGuids) -> u64 {
    let filename = "deleted-copy.obk";
    let destination = path(filename);
    host.save_copy(document, &destination, None, "alpha", source)
        .unwrap();
    open_file(host, filename, source, OpenActivation::Background)
}
