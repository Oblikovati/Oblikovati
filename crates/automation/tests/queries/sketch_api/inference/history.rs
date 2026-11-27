// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn inference_settings_join_history_and_equal_edits_do_not_create_steps() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    files.save(document, "test", &mut source).unwrap();
    for _ in 0..2 {
        set(&mut files, &mut source, &input("none")).unwrap();
    }
    assert_eq!(
        owner_revision(&files, document).1,
        ["Set sketch inference options"]
    );
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(get(&files).priority, "horizontalVertical");
    assert!(!owner_revision(&files, document).2);
    files.commands(document).unwrap().redo().unwrap();
    assert_eq!(get(&files), input("none"));
    assert!(owner_revision(&files, document).2);
}
#[test]
fn grouped_settings_commit_as_one_step_and_abort_restores_previous_values() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    files
        .commands(document)
        .unwrap()
        .begin("Preferences")
        .unwrap();
    set(&mut files, &mut source, &input("none")).unwrap();
    set(&mut files, &mut source, &input("parallelPerpendicular")).unwrap();
    files.commands(document).unwrap().end().unwrap();
    assert_eq!(owner_revision(&files, document).1, ["Preferences"]);
    files.commands(document).unwrap().begin("Aborted").unwrap();
    set(&mut files, &mut source, &input("none")).unwrap();
    files.commands(document).unwrap().abort().unwrap();
    assert_eq!(get(&files), input("parallelPerpendicular"));
}
#[test]
fn observer_veto_discards_preferences_and_does_not_return_success() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let observer = QueryObserver {
        veto: true,
        ..Default::default()
    };
    let document = reopen_with_observer(&mut files, document, &mut source, observer);
    assert!(
        set(&mut files, &mut source, &input("none"))
            .unwrap_err()
            .to_string()
            .contains("locked")
    );
    assert_eq!(get(&files).priority, "horizontalVertical");
    assert_eq!(owner_settings(&files, document), None);
    assert!(!owner_revision(&files, document).2);
}
#[test]
fn edited_preferences_survive_save_reopen_and_copy_without_shared_mutation() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    set(&mut files, &mut source, &input("none")).unwrap();
    let document = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_eq!(get(&files), input("none"));
    entities::activate_copy(
        &mut files,
        document,
        &mut source,
        Path::new("/fixtures/inference-copy.obk"),
    );
    set(&mut files, &mut source, &input("parallelPerpendicular")).unwrap();
    assert_eq!(get(&files), input("parallelPerpendicular"));
    files.activate(document).unwrap();
    assert_eq!(get(&files), input("none"));
}
#[test]
fn failed_fallback_response_leaves_shared_preferences_unchanged() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let before = get(&files);
    let result = files.set_sketch_inference_options_projected(&input("none"), |candidate| {
        assert_eq!(
            candidate.unwrap().constraint_priority,
            oblikovati_api::types::PRIORITY_NONE
        );
        Err::<(), _>("response refused")
    });
    assert!(result.unwrap_err().to_string().contains("response refused"));
    assert_eq!(get(&files), before);
}
#[test]
fn decline_ignored_by_a_projection_still_cannot_replace_fallback() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let before = get(&files);
    files
        .set_sketch_inference_options_projected(&input("bad"), |candidate| {
            assert_eq!(candidate.unwrap_err().0, "bad");
            Ok::<(), String>(())
        })
        .unwrap();
    assert_eq!(get(&files), before);
}
