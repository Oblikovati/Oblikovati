// SPDX-License-Identifier: GPL-2.0-only

use super::*;

fn reader(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    path: &str,
    driver: &str,
    name: &str,
) -> u64 {
    let recipe = format!(
        "schemaVersion: 2\ndocumentType: 1\nmodel:\n  parameters: [{{name: {name}, kind: derived, value: 2, unit: length}}]\n  derivedParameterTables: [{{id: 7, sourceDocument: {driver}, linked: [{name}]}}]\n"
    );
    files
        .create(
            Path::new(path),
            DocumentPackage::decode(recipe.as_bytes()).unwrap(),
            QueryObserver::default(),
            "test",
            source,
        )
        .unwrap()
        .session_id
}

fn chain() -> (DiscoveryWorkspace, u64, u64, u64, u64) {
    let (mut files, source_id, middle, mut source) = linked_files();
    files
        .commands(middle)
        .unwrap()
        .run("Output", |snapshot, ids| {
            let id = ids.allocate().unwrap();
            snapshot
                .parameters
                .insert_expression(id, "output", USER_PARAM, "width * 2")
        })
        .unwrap();
    let leaf = reader(
        &mut files,
        &mut source,
        "/leaf.opd",
        "/target.opd",
        "output",
    );
    let sibling = reader(
        &mut files,
        &mut source,
        "/sibling.opd",
        "/source.opd",
        "width",
    );
    (files, source_id, middle, leaf, sibling)
}

#[test]
fn source_changes_refresh_chains_in_driver_order_without_target_undo_steps() {
    let (mut files, source, middle, leaf, sibling) = chain();
    files
        .commands(source)
        .unwrap()
        .run("Width", |snapshot, _| {
            snapshot.parameters.set_expression(1, "5 cm")
        })
        .unwrap();
    let before = files.workspace().session(middle).unwrap().history_entries();
    let report = files.refresh_derived_from(source).unwrap();
    assert!(report.failures.is_empty() && report.blocked.is_empty());
    assert_eq!(report.changed, [middle, leaf, sibling]);
    assert_eq!(
        files
            .workspace()
            .session(leaf)
            .unwrap()
            .snapshot()
            .parameters
            .get(1)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        10.0
    );
    assert_eq!(
        files.workspace().session(middle).unwrap().history_entries(),
        before
    );
    assert!(
        files
            .workspace()
            .session(leaf)
            .unwrap()
            .undo_labels()
            .is_empty()
    );
    assert_eq!(files.workspace().active_document(), Some(sibling));
    assert!(
        files
            .refresh_derived_from(source)
            .unwrap()
            .changed
            .is_empty()
    );
    files.commands(source).unwrap().undo().unwrap();
    files.refresh_derived_from(source).unwrap();
    assert_eq!(
        files
            .workspace()
            .session(leaf)
            .unwrap()
            .snapshot()
            .parameters
            .get(1)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        4.0
    );
}

#[test]
fn failed_target_quarantines_readers_while_independent_branch_refreshes() {
    let (mut files, source, middle, leaf, sibling) = chain();
    files.commands(middle).unwrap().begin("busy").unwrap();
    files
        .commands(source)
        .unwrap()
        .run("Width", |snapshot, _| {
            snapshot.parameters.set_expression(1, "7 cm")
        })
        .unwrap();
    let report = files.refresh_derived_from(source).unwrap();
    assert_eq!(report.changed, [sibling]);
    assert_eq!(report.failures.len(), 2);
    assert!(
        report
            .failures
            .iter()
            .all(|failure| failure.document == middle)
    );
    assert_eq!(report.blocked.len(), 1);
    assert_eq!(report.blocked[0].document, leaf);
    assert_eq!(report.blocked[0].sources, [middle]);
    assert_eq!(files.workspace().session(leaf).unwrap().generation(), 0);
    files.commands(middle).unwrap().abort().unwrap();
    let retry = files.refresh_derived_from(source).unwrap();
    assert!(retry.failures.is_empty() && retry.blocked.is_empty());
    assert_eq!(retry.changed, [middle, leaf]);
}

#[test]
fn cycles_and_open_sources_are_refused_before_any_refresh() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut source = QueryGuids(0);
    let first = reader(
        &mut files,
        &mut source,
        "/first.opd",
        "/second.opd",
        "width",
    );
    let second = reader(
        &mut files,
        &mut source,
        "/second.opd",
        "/first.opd",
        "width",
    );
    let error = files.refresh_derived_from(first).unwrap_err().to_string();
    assert!(error.contains("acyclic"));
    for id in [first, second] {
        assert_eq!(files.workspace().session(id).unwrap().generation(), 0);
    }
    assert!(files.refresh_derived_from(999).is_err());
    files.commands(first).unwrap().begin("preview").unwrap();
    assert!(
        files
            .refresh_derived_from(first)
            .unwrap_err()
            .to_string()
            .contains("closed transaction")
    );
}

#[test]
fn refresh_after_target_navigation_refuses_uncommitted_source_values() {
    let (mut files, source, target, _) = linked_files();
    files.commands(source).unwrap().begin("preview").unwrap();
    files
        .commands(source)
        .unwrap()
        .run("Preview width", |snapshot, _| {
            snapshot.parameters.set_expression(1, "17 cm")
        })
        .unwrap();
    let report = files.refresh_derived_from(target).unwrap();
    assert_eq!(report.failures.len(), 1);
    assert_eq!(report.failures[0].document, target);
    assert_eq!(report.failures[0].table, 7);
    assert!(report.failures[0].reason.contains("closed transaction"));
    let width = files
        .workspace()
        .session(target)
        .unwrap()
        .snapshot()
        .parameters
        .get(1)
        .unwrap();
    assert_eq!(width.current_value().unwrap().value, 2.0);
    files.commands(source).unwrap().abort().unwrap();
    assert!(
        files
            .refresh_derived_from(target)
            .unwrap()
            .failures
            .is_empty()
    );
}

#[test]
fn ambiguous_source_refuses_the_whole_plan_before_refresh() {
    let (mut files, source, target, mut guids) = linked_files();
    let _duplicate = files
        .create_unbound(
            DocumentPackage::empty(DOCUMENT_PART, "/source.opd"),
            QueryObserver::default(),
            "test",
            &mut guids,
        )
        .unwrap();
    let before = files.workspace().session(target).unwrap().generation();
    let error = files.refresh_derived_from(source).unwrap_err().to_string();
    assert!(error.contains("one unique open document"), "{error}");
    assert_eq!(
        files.workspace().session(target).unwrap().generation(),
        before
    );
}
