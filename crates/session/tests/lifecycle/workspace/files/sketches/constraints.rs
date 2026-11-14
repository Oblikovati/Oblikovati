// SPDX-License-Identifier: GPL-2.0-only
use super::recompute::saved_driven_sketch;
use super::*;
use oblikovati_model::history::SketchConstraintReservations;
fn reserve_only(host: &mut Host, document: u64, sketch: DocumentSketchId) -> u64 {
    host.commands(document)
        .unwrap()
        .run("Reserve relation", |_, ids| -> Result<_, String> {
            Ok(ids.allocate_sketch_constraint(sketch).unwrap().0)
        })
        .unwrap()
}
fn read_reservations(host: &Host, file: &str) -> SketchConstraintReservations {
    let bytes = host.files().contents.borrow()[&path(file)].clone();
    DocumentPackage::decode(&bytes)
        .unwrap()
        .restore_sketches()
        .unwrap()
        .constraint_ids
}
fn assert_next_saved(host: &Host, file: &str, sketch: DocumentSketchId, expected: u64) {
    let mut restored = read_reservations(host, file);
    assert_eq!(restored.allocate(sketch, sketch.0).unwrap().0, expected);
}
fn failed_reservation(host: &mut Host, document: u64, sketch: DocumentSketchId) {
    let before = owned_sketches(host, document);
    let result =
        host.commands(document)
            .unwrap()
            .run("Refused relation", |_, ids| -> Result<(), String> {
                assert_eq!(ids.allocate_sketch_constraint(sketch).unwrap().0, 5);
                Err("refused after reservation".into())
            });
    assert!(result.is_err());
    assert!(host.workspace().needs_save(document).unwrap());
    assert!(before.same_evaluated_state(&owned_sketches(host, document)));
}
fn owned_sketches(
    host: &Host,
    document: u64,
) -> oblikovati_model::document_sketch::DocumentSketches {
    host.workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .clone()
}

#[test]
fn failed_and_aborted_reservations_dirty_the_owner_and_survive_reopen() {
    let (mut host, mut source, document, sketch) = saved_driven_sketch();
    failed_reservation(&mut host, document, sketch);
    host.commands(document)
        .unwrap()
        .begin("Abort relation")
        .unwrap();
    assert_eq!(reserve_only(&mut host, document, sketch), 6);
    host.commands(document).unwrap().abort().unwrap();
    host.save(document, "alpha", &mut source).unwrap();
    assert!(!host.workspace().needs_save(document).unwrap());
    assert_next_saved(&host, "a.obk", sketch, 7);
    host.close_discarding_changes(document).unwrap();
    let reopened = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    assert_eq!(reserve_only(&mut host, reopened, sketch), 7);
}
#[test]
fn deleted_sketch_constraint_marks_survive_saved_copies() {
    let (mut host, mut source, document, sketch) = saved_driven_sketch();
    assert_eq!(reserve_only(&mut host, document, sketch), 5);
    host.commands(document)
        .unwrap()
        .delete_sketch(sketch)
        .unwrap();
    let copied = save_constraint_copy(&mut host, document, sketch, &mut source);
    assert_eq!(reserve_only(&mut host, copied, sketch), 6);
}
fn save_constraint_copy(
    host: &mut Host,
    document: u64,
    sketch: DocumentSketchId,
    source: &mut SequenceGuids,
) -> u64 {
    host.save_copy(
        document,
        &path("constraints-copy.obk"),
        None,
        "alpha",
        source,
    )
    .unwrap();
    assert!(host.workspace().needs_save(document).unwrap());
    assert_next_saved(host, "constraints-copy.obk", sketch, 6);
    let copied = open_file(
        host,
        "constraints-copy.obk",
        source,
        OpenActivation::Background,
    );
    assert!(owned_sketches(host, copied).sketches().is_empty());
    copied
}
#[test]
fn malformed_constraint_marks_fail_before_registration_or_guid_allocation() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1700);
    let bytes = b"schemaVersion: 3\ndocumentType: 1\nmodel:\n  parameters: []\n  planarSketches:\n    sketchHighWater: 1\n    entityHighWater: 0\n    constraintHighWater: {0: 1}\n    recipe: {version: 1, sketches: []}\n";
    host.files()
        .contents
        .borrow_mut()
        .insert(path("a.obk"), bytes.to_vec());
    let result = host.open(
        &path("a.obk"),
        RecordingObserver::default(),
        "alpha",
        &mut source,
        OpenActivation::Activate,
    );
    assert!(result.unwrap_err().to_string().contains("recipe ID 0"));
    assert!(host.workspace().documents().is_empty());
    assert_eq!(source.0, 1700);
}
fn vetoed_constraint_session() -> DocumentSession<RecordingObserver> {
    let (host, _, document, _) = saved_driven_sketch();
    let snapshot = host
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .clone();
    let history =
        DocumentEditHistory::new(snapshot, RecipeIdAllocator::restore(Some(1), &[1]).unwrap())
            .unwrap();
    let observer = RecordingObserver {
        veto: Some(EVENT_TRANSACTION_COMMITTED),
        ..Default::default()
    };
    DocumentSession::new(document, history, observer)
}
#[test]
fn commit_veto_retains_constraint_ids_after_geometry_rollback() {
    let mut session = vetoed_constraint_session();
    let before = session.snapshot().sketches.clone();
    for expected in [5, 6] {
        session.begin("Veto relation").unwrap();
        edit_vetoed_reservation(&mut session, expected);
        assert_eq!(
            session.end(),
            Err(SessionError::Veto("export locked".into()))
        );
        assert!(before.same_evaluated_state(&session.snapshot().sketches));
    }
    assert!(session.undo_labels().is_empty());
}
fn edit_vetoed_reservation(session: &mut DocumentSession<RecordingObserver>, expected: u64) {
    session
        .edit(|snapshot, ids| -> Result<(), String> {
            let allocated = ids.allocate_sketch_constraint(DocumentSketchId(1)).unwrap();
            assert_eq!(allocated.0, expected);
            snapshot
                .sketches
                .rename(DocumentSketchId(1), "Vetoed")
                .map_err(|cause| cause.to_string())
        })
        .unwrap();
}
fn replace_first_constraint(host: &mut Host, document: u64, sketch: DocumentSketchId) {
    host.commands(document)
        .unwrap()
        .run("Replace relation", |snapshot, ids| {
            let mut recipe = snapshot.sketches.get(sketch).unwrap().geometry().recipe();
            recipe.constraints[0].id = ids.allocate_sketch_constraint(sketch).unwrap();
            let geometry = Sketch2D::from_recipe(&recipe).unwrap();
            snapshot.sketches.replace_geometry(sketch, geometry)
        })
        .unwrap();
}
#[test]
fn undo_to_the_saved_recipe_stays_dirty_until_the_new_reservations_are_saved() {
    let (mut host, mut source, document, sketch) = saved_driven_sketch();
    let before = owned_sketches(&host, document);
    replace_first_constraint(&mut host, document, sketch);
    host.commands(document).unwrap().undo().unwrap();
    assert!(before.same_evaluated_state(&owned_sketches(&host, document)));
    assert!(host.workspace().needs_save(document).unwrap());
    host.save(document, "alpha", &mut source).unwrap();
    assert!(!host.workspace().needs_save(document).unwrap());
    assert_next_saved(&host, "a.obk", sketch, 6);
    host.commands(document).unwrap().redo().unwrap();
    assert!(host.workspace().needs_save(document).unwrap());
}
