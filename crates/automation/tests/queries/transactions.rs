// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_automation::{SessionHost, TRANSACTION_METHODS};

fn transaction_parameters() -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "4 cm")
        .unwrap();
    parameters
}

#[test]
fn nested_wire_batch_commits_once_and_supports_undo_and_redo() {
    let (mut workspace, id) = super::parameters::parameter_workspace(transaction_parameters());
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        assert!(
            client
                .transactions()
                .begin(&TransactionBeginArgs {
                    label: "Resize".into()
                })
                .unwrap()
                .ok
        );
        client
            .parameters()
            .set(&ParameterSetArgs {
                name: "width".into(),
                expression: "5 cm".into(),
                ..Default::default()
            })
            .unwrap();
        client
            .transactions()
            .begin(&TransactionBeginArgs {
                label: "Inner".into(),
            })
            .unwrap();
        client
            .parameters()
            .rename(&ParameterRenameArgs {
                name: "width".into(),
                new_name: "span".into(),
            })
            .unwrap();
        assert_eq!(client.transactions().end().unwrap(), UndoState::default());
        assert!(client.transactions().undo().is_err());
        let state = client.transactions().end().unwrap();
        assert!(state.can_undo && !state.can_redo);
        assert_eq!(state.next_undo, "Resize");
        let state = client.transactions().undo().unwrap();
        assert!(!state.can_undo && state.can_redo);
        assert_eq!(state.next_redo, "Resize");
        assert_eq!(
            client
                .parameters()
                .get(&ParameterNameArgs {
                    name: "width".into()
                })
                .unwrap()
                .expression,
            "4 cm"
        );
        let state = client.transactions().redo().unwrap();
        assert_eq!(state.next_undo, "Resize");
        assert_eq!(
            client
                .parameters()
                .get(&ParameterNameArgs {
                    name: "span".into()
                })
                .unwrap()
                .expression,
            "5 cm"
        );
        assert_eq!(client.transactions().state().unwrap(), state);
    }
    assert_eq!(workspace.session(id).unwrap().undo_labels(), ["Resize"]);
}

#[test]
fn abort_discards_all_nested_edits_and_preserves_redo() {
    let (mut workspace, id) = super::parameters::parameter_workspace(transaction_parameters());
    let client = Client::new(SessionHost::new(&mut workspace));
    client
        .parameters()
        .set(&ParameterSetArgs {
            name: "width".into(),
            expression: "6 cm".into(),
            ..Default::default()
        })
        .unwrap();
    let before = client.transactions().undo().unwrap();
    for label in ["Outer", "Inner"] {
        client
            .transactions()
            .begin(&TransactionBeginArgs {
                label: label.into(),
            })
            .unwrap();
    }
    client
        .parameters()
        .set(&ParameterSetArgs {
            name: "width".into(),
            expression: "9 cm".into(),
            ..Default::default()
        })
        .unwrap();
    assert!(
        client
            .parameters()
            .rename(&ParameterRenameArgs {
                name: "missing".into(),
                new_name: "span".into()
            })
            .is_err()
    );
    assert_eq!(client.transactions().abort().unwrap(), before);
    assert_eq!(
        client
            .parameters()
            .get(&ParameterNameArgs {
                name: "width".into()
            })
            .unwrap()
            .expression,
        "4 cm"
    );
    assert!(client.transactions().end().is_err());
    assert!(client.transactions().abort().is_err());
    assert!(!workspace.session(id).unwrap().in_transaction());
    assert!(!workspace.needs_save(id).unwrap());
}

#[test]
fn empty_context_history_and_transaction_errors_do_not_create_edits() {
    let mut workspace = Workspace::new(QueryObserver::default());
    assert_eq!(
        Client::new(SessionQueryHost::new(&workspace))
            .transactions()
            .state()
            .unwrap(),
        UndoState::default()
    );
    for method in TRANSACTION_METHODS {
        assert!(
            SessionHost::new(&mut workspace)
                .call(method, b"{}")
                .is_err()
        );
    }
    let (mut workspace, id) = super::parameters::parameter_workspace(transaction_parameters());
    assert!(
        SessionHost::new(&mut workspace)
            .call(METHOD_TRANSACTION_BEGIN, b"[]")
            .is_err()
    );
    let client = Client::new(SessionHost::new(&mut workspace));
    assert!(client.transactions().undo().is_err());
    assert!(client.transactions().redo().is_err());
    client
        .transactions()
        .begin(&TransactionBeginArgs {
            label: "No change".into(),
        })
        .unwrap();
    assert_eq!(client.transactions().end().unwrap(), UndoState::default());
    assert!(workspace.session(id).unwrap().undo_labels().is_empty());
    assert!(!workspace.needs_save(id).unwrap());
}

#[test]
fn commit_veto_restores_the_entire_wire_batch() {
    let (mut workspace, id) = super::parameters::parameter_workspace_observer(
        transaction_parameters(),
        QueryObserver {
            veto: true,
            ..Default::default()
        },
    );
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        client
            .transactions()
            .begin(&TransactionBeginArgs {
                label: "Vetoed".into(),
            })
            .unwrap();
        client
            .parameters()
            .set(&ParameterSetArgs {
                name: "width".into(),
                expression: "8 cm".into(),
                ..Default::default()
            })
            .unwrap();
        assert!(client.transactions().end().is_err());
        assert_eq!(client.transactions().state().unwrap(), UndoState::default());
        assert_eq!(
            client
                .parameters()
                .get(&ParameterNameArgs {
                    name: "width".into()
                })
                .unwrap()
                .expression,
            "4 cm"
        );
    }
    assert!(!workspace.session(id).unwrap().in_transaction());
    assert!(!workspace.needs_save(id).unwrap());
}

#[test]
fn history_commands_only_change_the_active_document() {
    let mut workspace = Workspace::new(QueryObserver::default());
    let mut guids = QueryGuids(0);
    let first = register(&mut workspace, UnitsOfMeasure::default(), &mut guids);
    Client::new(SessionHost::new(&mut workspace))
        .parameters()
        .add(&ParameterSetArgs {
            name: "first".into(),
            expression: "1 cm".into(),
            ..Default::default()
        })
        .unwrap();
    let second = register(&mut workspace, UnitsOfMeasure::default(), &mut guids);
    assert_eq!(
        Client::new(SessionQueryHost::new(&workspace))
            .transactions()
            .state()
            .unwrap(),
        UndoState::default()
    );
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        assert!(client.transactions().undo().is_err());
        client
            .parameters()
            .add(&ParameterSetArgs {
                name: "second".into(),
                expression: "2 cm".into(),
                ..Default::default()
            })
            .unwrap();
        client.transactions().undo().unwrap();
    }
    assert_eq!(workspace.active_document(), Some(second));
    assert_eq!(
        workspace
            .session(first)
            .unwrap()
            .snapshot()
            .parameters
            .find("first"),
        Some(1)
    );
    assert!(
        workspace
            .session(second)
            .unwrap()
            .snapshot()
            .parameters
            .ids()
            .is_empty()
    );
    let host = SessionQueryHost::new(&workspace);
    for method in TRANSACTION_METHODS {
        assert!(host.call(method, b"{}").is_err());
    }
    let inventory: std::collections::BTreeSet<&str> = QUERY_METHODS
        .into_iter()
        .chain(oblikovati_automation::EDIT_METHODS)
        .chain(TRANSACTION_METHODS)
        .chain(oblikovati_automation::CLIENT_COMMAND_METHODS)
        .collect();
    assert_eq!(inventory.len(), 65);
}

#[test]
fn history_query_reads_full_stream_without_activating_another_document() {
    let mut workspace = Workspace::new(QueryObserver::default());
    let mut guids = QueryGuids(0);
    let first = register(&mut workspace, UnitsOfMeasure::default(), &mut guids);
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        client
            .parameters()
            .add(&ParameterSetArgs {
                name: "width".into(),
                expression: "1 cm".into(),
                ..Default::default()
            })
            .unwrap();
        client
            .parameters()
            .set(&ParameterSetArgs {
                name: "width".into(),
                expression: "2 cm".into(),
                ..Default::default()
            })
            .unwrap();
        client.transactions().undo().unwrap();
    }
    let second = register(&mut workspace, UnitsOfMeasure::default(), &mut guids);
    let before = workspace.observer().notifications;
    let history = Client::new(SessionQueryHost::new(&workspace))
        .transactions()
        .history(&TransactionHistoryArgs { document: first })
        .unwrap();
    assert_eq!(history.document, first);
    assert_eq!(history.position, 1);
    let entries = history.entries.unwrap();
    assert_eq!(
        entries
            .iter()
            .map(|entry| entry.label.as_str())
            .collect::<Vec<_>>(),
        ["Add Parameter", "Edit Parameter"]
    );
    assert!(entries.iter().all(|entry| !entry.saved));
    let history = Client::new(SessionQueryHost::new(&workspace))
        .transactions()
        .history(&TransactionHistoryArgs::default())
        .unwrap();
    assert_eq!(history.document, second);
    assert_eq!(history.position, 0);
    assert_eq!(history.entries, Some(vec![]));
    assert_eq!(workspace.active_document(), Some(second));
    assert_eq!(workspace.observer().notifications, before);
}

#[test]
fn history_excludes_preview_and_refuses_missing_documents() {
    let mut workspace = Workspace::new(QueryObserver::default());
    assert!(
        Client::new(SessionQueryHost::new(&workspace))
            .transactions()
            .history(&TransactionHistoryArgs::default())
            .is_err()
    );
    let _id = register(
        &mut workspace,
        UnitsOfMeasure::default(),
        &mut QueryGuids(0),
    );
    let client = Client::new(SessionHost::new(&mut workspace));
    client
        .transactions()
        .begin(&TransactionBeginArgs {
            label: "Preview".into(),
        })
        .unwrap();
    client
        .parameters()
        .add(&ParameterSetArgs {
            name: "width".into(),
            expression: "1 cm".into(),
            ..Default::default()
        })
        .unwrap();
    let history = client
        .transactions()
        .history(&TransactionHistoryArgs::default())
        .unwrap();
    assert_eq!(history.position, 0);
    assert_eq!(history.entries, Some(vec![]));
    assert!(
        client
            .transactions()
            .history(&TransactionHistoryArgs { document: 999 })
            .is_err()
    );
    client.transactions().abort().unwrap();
}

#[derive(Default)]
struct MemoryHistorySave {
    prepared: Option<DocumentSnapshot>,
    persisted: bool,
}
impl oblikovati_model::identity::DocumentSaveTarget for MemoryHistorySave {
    type Failure = std::io::Error;
    fn persist(&mut self, _: &FileIdentity) -> Result<(), Self::Failure> {
        assert!(self.prepared.is_some());
        self.persisted = true;
        Ok(())
    }
}
impl oblikovati_model::history::DocumentSnapshotTarget for MemoryHistorySave {
    fn prepare(
        &mut self,
        snapshot: &DocumentSnapshot,
        _derived_high_water: i64,
        _parameter_high_water: u64,
        _sketch_marks: oblikovati_model::history::SketchReservationMarks,
        _feature_high_water: u64,
    ) -> Result<Option<oblikovati_model::identity::RecipeDigest>, Self::Failure> {
        self.prepared = Some(snapshot.clone());
        Ok(None)
    }
}

#[test]
fn history_wire_retains_save_markers_in_the_redo_suffix() {
    let (mut workspace, id) = super::parameters::parameter_workspace(transaction_parameters());
    Client::new(SessionHost::new(&mut workspace))
        .parameters()
        .set(&ParameterSetArgs {
            name: "width".into(),
            expression: "5 cm".into(),
            ..Default::default()
        })
        .unwrap();
    let mut target = MemoryHistorySave::default();
    workspace
        .save(id, "test", &mut QueryGuids(100), &mut target)
        .unwrap();
    assert!(target.persisted);
    let client = Client::new(SessionHost::new(&mut workspace));
    client.transactions().undo().unwrap();
    let history = client
        .transactions()
        .history(&TransactionHistoryArgs::default())
        .unwrap();
    assert_eq!(history.position, 0);
    let entries = history.entries.unwrap();
    assert_eq!(entries.len(), 1);
    assert!(entries[0].saved);
    assert_eq!(entries[0].label, "Edit Parameter");
}
