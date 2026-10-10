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
        .collect();
    assert_eq!(inventory.len(), 40);
}
