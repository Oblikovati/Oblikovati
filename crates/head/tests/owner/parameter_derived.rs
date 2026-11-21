// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn linked_parameter_owner() -> (TestOwner, u64, i64) {
    let (mut owner, awake) = owner();
    Client::new(owner.document_host())
        .documents()
        .create(&CreateDocumentArgs {
            type_: "part".into(),
            name: "Source part".into(),
            ..Default::default()
        })
        .unwrap();
    Client::new(owner.document_host())
        .parameters()
        .add(&ParameterSetArgs {
            name: "span".into(),
            expression: "4 cm".into(),
            ..Default::default()
        })
        .unwrap();
    let target = create(&mut owner, &awake);
    let table = Client::new(owner.document_host())
        .parameters()
        .add_derived_table(&DerivedParameterTableAddArgs {
            source_document: "Source part".into(),
            linked: Some(vec!["span".into()]),
        })
        .unwrap();
    (owner, target, table.id)
}
#[test]
fn linked_table_views_keep_canonical_identity_health_and_read_only_parameters() {
    let (owner, target, table) = linked_parameter_owner();
    let view = owner.gui_parameter_snapshot().unwrap().unwrap();
    assert_eq!(view.document, target);
    assert_eq!(view.linked_tables.len(), 1);
    assert_eq!(view.linked_tables[0].id, table);
    assert_eq!(view.linked_tables[0].source_document, "Source part");
    assert_eq!(view.linked_tables[0].linked, Some(vec!["span".into()]));
    assert!(view.linked_tables[0].health.is_empty());
    assert_eq!(view.rows[0].detail.parameter_info.kind, "derived");
    assert_eq!(view.rows[0].detail.parameter_info.value, "40 mm");
    view.validate().unwrap();
}
#[test]
fn gui_link_picker_projects_stable_sources_and_links_only_checked_parameter_identities() {
    let (mut owner, target, table) = linked_parameter_owner();
    let view = owner.gui_parameter_snapshot().unwrap().unwrap();
    let source = view.link_sources.first().unwrap();
    assert_ne!(source.document, target);
    assert_eq!(source.source_document, "Source part");
    assert_eq!(source.parameters.len(), 1);
    owner
        .apply_gui_parameter(&GuiParameterAction::DeleteLinkedTable {
            document: target,
            table,
        })
        .unwrap();
    let action = GuiParameterAction::Link(oblikovati_gui::GuiParameterLinkEdit {
        document: target,
        source_document: source.document,
        request: 1,
        linked: vec![source.parameters[0].parameter],
    });
    owner.apply_gui_parameter(&action).unwrap();
    let linked = owner.gui_parameter_snapshot().unwrap().unwrap();
    assert_eq!(linked.linked_tables.len(), 1);
    assert_eq!(linked.linked_tables[0].linked, Some(vec!["span".into()]));
}
#[test]
fn gui_link_picker_refuses_stale_source_parameter_identities_without_a_table() {
    let (mut owner, target, _) = linked_parameter_owner();
    let source = owner
        .gui_parameter_snapshot()
        .unwrap()
        .unwrap()
        .link_sources
        .remove(0);
    let action = GuiParameterAction::Link(oblikovati_gui::GuiParameterLinkEdit {
        document: target,
        source_document: source.document,
        request: 1,
        linked: vec![source.parameters[0].parameter + 1],
    });
    assert!(owner.apply_gui_parameter(&action).is_err());
    assert_eq!(
        owner
            .gui_parameter_snapshot()
            .unwrap()
            .unwrap()
            .linked_tables
            .len(),
        1
    );
}
#[test]
fn gui_link_picker_refuses_duplicate_and_self_link_packets_before_mutation() {
    let (mut owner, target, _) = linked_parameter_owner();
    let source = owner
        .gui_parameter_snapshot()
        .unwrap()
        .unwrap()
        .link_sources
        .remove(0);
    for edit in [
        oblikovati_gui::GuiParameterLinkEdit {
            document: target,
            source_document: target,
            request: 1,
            linked: Vec::new(),
        },
        oblikovati_gui::GuiParameterLinkEdit {
            document: target,
            source_document: source.document,
            request: 1,
            linked: vec![
                source.parameters[0].parameter,
                source.parameters[0].parameter,
            ],
        },
    ] {
        assert!(
            owner
                .apply_gui_parameter(&GuiParameterAction::Link(edit))
                .is_err()
        );
    }
    assert_eq!(
        owner
            .gui_parameter_snapshot()
            .unwrap()
            .unwrap()
            .linked_tables
            .len(),
        1
    );
}
#[test]
fn gui_link_picker_skips_ambiguous_unsaved_source_names() {
    let (mut owner, awake) = owner();
    let source = create(&mut owner, &awake);
    Client::new(owner.document_host())
        .documents()
        .create(&CreateDocumentArgs {
            type_: "part".into(),
            name: "Owner part".into(),
            ..Default::default()
        })
        .unwrap();
    let target = create(&mut owner, &awake);
    let view = owner.gui_parameter_snapshot().unwrap().unwrap();
    assert_eq!(view.document, target);
    assert!(
        !view
            .link_sources
            .iter()
            .any(|candidate| candidate.document == source)
    );
}
#[test]
fn gui_linked_table_delete_removes_derived_parameters_and_undo_restores_the_same_identities() {
    let (mut owner, target, table) = linked_parameter_owner();
    let before = owner.gui_parameter_snapshot().unwrap().unwrap();
    let action = GuiParameterAction::DeleteLinkedTable {
        document: target,
        table,
    };
    owner.apply_gui_parameter(&action).unwrap();
    let deleted = owner.gui_parameter_snapshot().unwrap().unwrap();
    assert!(deleted.linked_tables.is_empty());
    assert!(deleted.rows.is_empty());
    owner.files.commands(target).unwrap().undo().unwrap();
    assert_eq!(owner.gui_parameter_snapshot().unwrap().unwrap(), before);
    owner.files.commands(target).unwrap().redo().unwrap();
    assert_eq!(owner.gui_parameter_snapshot().unwrap().unwrap(), deleted);
}
#[test]
fn missing_and_stale_table_targets_are_refused_without_changing_the_active_document() {
    let (mut owner, target, table) = linked_parameter_owner();
    let before = owner.gui_parameter_snapshot().unwrap().unwrap();
    for table in [0, -1, table + 1] {
        assert!(
            owner
                .apply_gui_parameter(&GuiParameterAction::DeleteLinkedTable {
                    document: target,
                    table
                })
                .is_err()
        );
        assert_eq!(owner.gui_parameter_snapshot().unwrap().unwrap(), before);
    }
    Client::new(owner.document_host())
        .documents()
        .create(&CreateDocumentArgs {
            type_: "part".into(),
            name: "Next part".into(),
            ..Default::default()
        })
        .unwrap();
    let active = owner.gui_parameter_snapshot().unwrap().unwrap();
    assert!(
        owner
            .apply_gui_parameter(&GuiParameterAction::DeleteLinkedTable {
                document: target,
                table
            })
            .unwrap_err()
            .contains("expected current document")
    );
    assert_eq!(owner.gui_parameter_snapshot().unwrap().unwrap(), active);
}
#[test]
fn linked_table_delete_reports_reader_failure_and_undo_restores_reader_health() {
    let (mut owner, target, table) = linked_parameter_owner();
    Client::new(owner.document_host())
        .parameters()
        .add(&ParameterSetArgs {
            name: "double_span".into(),
            expression: "2 * span".into(),
            ..Default::default()
        })
        .unwrap();
    let before = owner.gui_parameter_snapshot().unwrap().unwrap();
    owner
        .apply_gui_parameter(&GuiParameterAction::DeleteLinkedTable {
            document: target,
            table,
        })
        .unwrap();
    let deleted = owner.gui_parameter_snapshot().unwrap().unwrap();
    assert!(deleted.linked_tables.is_empty());
    assert_eq!(deleted.rows.len(), 1);
    assert_eq!(deleted.rows[0].detail.parameter_info.name, "double_span");
    assert!(!deleted.rows[0].detail.parameter_info.health.is_empty());
    assert_eq!(deleted.rows[0].detail.parameter_info.value, "80 mm");
    owner.files.commands(target).unwrap().undo().unwrap();
    assert_eq!(owner.gui_parameter_snapshot().unwrap().unwrap(), before);
}
#[test]
fn linked_tables_owned_by_features_refuse_direct_gui_deletion() {
    let (mut owner, target, table) = linked_parameter_owner();
    owner
        .files
        .commands(target)
        .unwrap()
        .run("Own Derived Table", |snapshot, _| {
            snapshot.parameters.mark_derived_table_owned(table)
        })
        .unwrap();
    let before = owner.gui_parameter_snapshot().unwrap().unwrap();
    assert!(before.linked_tables[0].has_reference_component);
    assert!(
        owner
            .apply_gui_parameter(&GuiParameterAction::DeleteLinkedTable {
                document: target,
                table
            })
            .unwrap_err()
            .contains("owning component")
    );
    assert_eq!(owner.gui_parameter_snapshot().unwrap().unwrap(), before);
}
