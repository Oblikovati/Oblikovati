// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn verifying_linked_real_solids_changes_no_history_ids_or_geometry() {
    let (host, source, target) = linked_prism_documents("master");
    let source_session = host.workspace.session(source).unwrap();
    let generation = source_session.generation();
    let high_water = source_session.high_water();
    let undo_count = source_session.undo_labels().len();
    let target_generation = host.workspace.session(target).unwrap().generation();
    let candidate = source_session
        .preview_projected(
            |snapshot, _| {
                snapshot
                    .parameters
                    .set_expression(4, "3 cm")
                    .map_err(|error| error.to_string())
            },
            |snapshot, ()| Ok(snapshot.clone()),
        )
        .unwrap();
    host.verify_named_candidate(source, &candidate).unwrap();
    assert_eq!(
        host.workspace.session(source).unwrap().generation(),
        generation
    );
    assert_eq!(
        host.workspace.session(source).unwrap().high_water(),
        high_water
    );
    assert_eq!(
        host.workspace.session(source).unwrap().undo_labels().len(),
        undo_count
    );
    assert_eq!(
        host.workspace.session(target).unwrap().generation(),
        target_generation
    );
    assert!((part_volume(&host, source) - 24.0).abs() < 1e-8);
    assert!((part_volume(&host, target) - 24.0).abs() < 1e-8);
}

#[test]
fn verification_reports_a_failed_linked_replay_without_mutation() {
    let (host, source, target) = linked_prism_documents("4 cm - master");
    let session = host.workspace.session(source).unwrap();
    let generation = session.generation();
    let candidate = session
        .preview_projected(
            |snapshot, _| {
                snapshot
                    .parameters
                    .set_expression(4, "4 cm")
                    .map_err(|error| error.to_string())
            },
            |snapshot, ()| Ok(snapshot.clone()),
        )
        .unwrap();
    assert!(host.verify_named_candidate(source, &candidate).is_err());
    assert_eq!(
        host.workspace.session(source).unwrap().generation(),
        generation
    );
    assert!((part_volume(&host, source) - 24.0).abs() < 1e-8);
    assert!((part_volume(&host, target) - 24.0).abs() < 1e-8);
}
