// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_automation::FileHost;

fn link_workspace(veto: bool) -> (DiscoveryWorkspace, u64, u64, QueryGuids) {
    let (mut files, source, old, mut guids) = linked_files();
    files.close_discarding_changes(old).unwrap();
    let target = files
        .create_unbound(
            DocumentPackage::empty(DOCUMENT_PART, "Target"),
            QueryObserver {
                veto,
                ..Default::default()
            },
            "test",
            &mut guids,
        )
        .unwrap()
        .session_id;
    (files, source, target, guids)
}

fn add_link(
    files: &mut DiscoveryWorkspace,
    guids: &mut QueryGuids,
    names: &[&str],
) -> Result<DerivedParameterTableInfo, oblikovati_api::Error> {
    Client::new(FileHost::new(files, guids, "test"))
        .parameters()
        .add_derived_table(&DerivedParameterTableAddArgs {
            source_document: "/source.opd".into(),
            linked: Some(names.iter().map(|name| (*name).into()).collect()),
        })
}

#[test]
fn add_and_relink_export_sources_preserve_kept_ids_and_follow_source_edits() {
    let (mut files, source, target, mut guids) = link_workspace(false);
    let table = add_link(&mut files, &mut guids, &["width"]).unwrap();
    assert_eq!(table.linked, Some(vec!["width".into()]));
    assert_eq!(table.available, Some(vec!["width".into(), "height".into()]));
    assert!(table.health.is_empty());
    let width = files
        .workspace()
        .session(target)
        .unwrap()
        .snapshot()
        .parameters
        .find("width")
        .unwrap();
    let exported = files.workspace().session(source).unwrap();
    assert!(
        exported
            .snapshot()
            .parameters
            .get(1)
            .unwrap()
            .annotations
            .exposed_as_property
    );
    assert!(exported.undo_labels().is_empty());
    assert!(
        exported
            .snapshot()
            .properties
            .get(oblikovati_model::property::USER_DEFINED, "width")
            .is_some()
    );
    Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .parameters()
        .set_derived_table_linked(&DerivedParameterTableSetLinkedArgs {
            id: table.id,
            linked: Some(vec!["width".into(), "height".into()]),
        })
        .unwrap();
    assert_eq!(
        files
            .workspace()
            .session(target)
            .unwrap()
            .snapshot()
            .parameters
            .find("width"),
        Some(width)
    );
    assert!(
        files
            .workspace()
            .session(source)
            .unwrap()
            .snapshot()
            .parameters
            .get(2)
            .unwrap()
            .annotations
            .exposed_as_property
    );
    files.activate(source).unwrap();
    Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .parameters()
        .set(&ParameterSetArgs {
            name: "width".into(),
            expression: "6 cm".into(),
            ..Default::default()
        })
        .unwrap();
    assert_eq!(
        files
            .workspace()
            .session(target)
            .unwrap()
            .snapshot()
            .parameters
            .get(width)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        6.0
    );
}

#[test]
fn target_veto_and_invalid_links_leave_source_export_unchanged() {
    for veto in [true, false] {
        let (mut files, source, target, mut guids) = link_workspace(veto);
        let names = if veto {
            vec!["width"]
        } else {
            vec!["width", "absent"]
        };
        assert!(add_link(&mut files, &mut guids, &names).is_err());
        assert!(
            !files
                .workspace()
                .session(source)
                .unwrap()
                .snapshot()
                .parameters
                .get(1)
                .unwrap()
                .annotations
                .exposed_as_property
        );
        assert_eq!(files.workspace().session(source).unwrap().generation(), 0);
        assert!(
            files
                .workspace()
                .session(target)
                .unwrap()
                .snapshot()
                .parameters
                .derived_tables()
                .is_empty()
        );
    }
}

#[test]
fn cycles_open_sources_and_missing_sources_are_refused_before_linking() {
    let (mut files, source, target, mut guids) = link_workspace(false);
    files.commands(source).unwrap().begin("Preview").unwrap();
    assert!(add_link(&mut files, &mut guids, &["width"]).is_err());
    files.commands(source).unwrap().abort().unwrap();
    add_link(&mut files, &mut guids, &[]).unwrap();
    files.activate(source).unwrap();
    let error = Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .parameters()
        .add_derived_table(&DerivedParameterTableAddArgs {
            source_document: "Target".into(),
            linked: None,
        })
        .unwrap_err()
        .to_string();
    assert!(error.contains("acyclic"), "{error}");
    assert!(
        files
            .workspace()
            .session(source)
            .unwrap()
            .snapshot()
            .parameters
            .derived_tables()
            .is_empty()
    );
    files.activate(target).unwrap();
    files.close_discarding_changes(source).unwrap();
    assert!(add_link(&mut files, &mut guids, &[]).is_err());
}

#[test]
fn target_undo_retains_source_exports_and_does_not_reuse_table_ids() {
    let (mut files, source, target, mut guids) = link_workspace(false);
    let first = add_link(&mut files, &mut guids, &["width"]).unwrap().id;
    Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .transactions()
        .undo()
        .unwrap();
    assert!(
        files
            .workspace()
            .session(target)
            .unwrap()
            .snapshot()
            .parameters
            .derived_tables()
            .is_empty()
    );
    assert!(
        files
            .workspace()
            .session(source)
            .unwrap()
            .snapshot()
            .parameters
            .get(1)
            .unwrap()
            .annotations
            .exposed_as_property
    );
    let second = add_link(&mut files, &mut guids, &["width"]).unwrap().id;
    assert!(second > first);
}

#[test]
fn relink_failure_keeps_both_recipes_and_empty_links_remove_parameters() {
    let (mut files, source, target, mut guids) = link_workspace(false);
    let table = add_link(&mut files, &mut guids, &["width"]).unwrap();
    let generation = files.workspace().session(source).unwrap().generation();
    let error = Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .parameters()
        .set_derived_table_linked(&DerivedParameterTableSetLinkedArgs {
            id: table.id,
            linked: Some(vec!["height".into(), "height".into()]),
        })
        .unwrap_err()
        .to_string();
    assert!(error.contains("unique"), "{error}");
    assert_eq!(
        files.workspace().session(source).unwrap().generation(),
        generation
    );
    assert!(
        !files
            .workspace()
            .session(source)
            .unwrap()
            .snapshot()
            .parameters
            .get(2)
            .unwrap()
            .annotations
            .exposed_as_property
    );
    let emptied = Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .parameters()
        .set_derived_table_linked(&DerivedParameterTableSetLinkedArgs {
            id: table.id,
            linked: None,
        })
        .unwrap();
    assert_eq!(emptied.linked, None);
    assert!(
        files
            .workspace()
            .session(target)
            .unwrap()
            .snapshot()
            .parameters
            .ids()
            .is_empty()
    );
}

#[test]
fn nested_link_abort_keeps_source_export_as_in_go() {
    let (mut files, source, target, mut guids) = link_workspace(false);
    Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .transactions()
        .begin(&TransactionBeginArgs {
            label: "Preview links".into(),
        })
        .unwrap();
    add_link(&mut files, &mut guids, &["height"]).unwrap();
    Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .transactions()
        .abort()
        .unwrap();
    assert!(
        files
            .workspace()
            .session(target)
            .unwrap()
            .snapshot()
            .parameters
            .derived_tables()
            .is_empty()
    );
    assert!(
        files
            .workspace()
            .session(source)
            .unwrap()
            .snapshot()
            .parameters
            .get(2)
            .unwrap()
            .annotations
            .exposed_as_property
    );
}

struct ReservationObserverFactory;
impl oblikovati_automation::DocumentObserverFactory for ReservationObserverFactory {
    type Observer = QueryObserver;
    fn create(&mut self) -> Result<QueryObserver, String> {
        Ok(QueryObserver::default())
    }
}

#[test]
fn undo_reservations_make_a_document_dirty_and_survive_close_saving() {
    let (mut files, _, target, mut guids) = link_workspace(false);
    Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document: target,
            new_full_document_name: "/reserved.opd".into(),
        })
        .unwrap();
    assert!(!files.workspace().needs_save(target).unwrap());
    let first = add_link(&mut files, &mut guids, &["width"]).unwrap().id;
    Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .transactions()
        .undo()
        .unwrap();
    assert!(
        files
            .workspace()
            .session(target)
            .unwrap()
            .snapshot()
            .parameters
            .derived_tables()
            .is_empty()
    );
    assert!(files.workspace().needs_save(target).unwrap());
    Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .documents()
        .close(&CloseDocumentArgs {
            id: target,
            force: false,
        })
        .unwrap();
    let reopened = Client::new(
        FileHost::new(&mut files, &mut guids, "test")
            .with_observer_factory(&mut ReservationObserverFactory),
    )
    .documents()
    .open(&OpenDocumentArgs {
        full_document_name: "/reserved.opd".into(),
        ..Default::default()
    })
    .unwrap();
    assert!(!files.workspace().needs_save(reopened.id).unwrap());
    let next = add_link(&mut files, &mut guids, &["width"]).unwrap().id;
    assert!(next > first);
}

#[test]
fn save_copy_preserves_deleted_table_reservations() {
    let (mut files, _, target, mut guids) = link_workspace(false);
    let first = add_link(&mut files, &mut guids, &["width"]).unwrap().id;
    Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .parameters()
        .delete_derived_table(&DerivedParameterTableDeleteArgs { id: first })
        .unwrap();
    Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .documents()
        .save_copy_as(&SaveCopyAsArgs {
            document: target,
            target_file_name: "/reserved-copy.opd".into(),
            ..Default::default()
        })
        .unwrap();
    assert!(files.workspace().needs_save(target).unwrap());
    let _opened = Client::new(
        FileHost::new(&mut files, &mut guids, "test")
            .with_observer_factory(&mut ReservationObserverFactory),
    )
    .documents()
    .open(&OpenDocumentArgs {
        full_document_name: "/reserved-copy.opd".into(),
        ..Default::default()
    })
    .unwrap();
    assert!(add_link(&mut files, &mut guids, &["width"]).unwrap().id > first);
}

#[test]
fn creation_from_a_package_restores_deleted_reservation_marks() {
    let (mut files, _, _, mut guids) = link_workspace(false);
    let package = DocumentPackage::decode(
        b"schemaVersion: 3\ndocumentType: 1\nmodel:\n  derivedTableHighWater: 12\n",
    )
    .unwrap();
    let _created = files
        .create_unbound(package, QueryObserver::default(), "test", &mut guids)
        .unwrap();
    assert_eq!(add_link(&mut files, &mut guids, &[]).unwrap().id, 13);
}
