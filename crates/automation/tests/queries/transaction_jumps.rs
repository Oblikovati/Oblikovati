// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_automation::SessionHost;

#[test]
fn jump_reads_and_moves_explicit_document_without_changing_focus() {
    let mut workspace = Workspace::new(QueryObserver::default());
    let mut source = QueryGuids(0);
    let first = register(&mut workspace, UnitsOfMeasure::default(), &mut source);
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        for name in ["width", "height", "depth"] {
            client
                .parameters()
                .add(&ParameterSetArgs {
                    name: name.into(),
                    expression: "1 cm".into(),
                    ..Default::default()
                })
                .unwrap();
        }
    }
    let second = register(&mut workspace, UnitsOfMeasure::default(), &mut source);
    let notices = workspace.observer().notifications;
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        for position in [0, 2, 1, 3, 3] {
            let history = client
                .transactions()
                .jump_to(&TransactionJumpToArgs {
                    document: first,
                    position,
                })
                .unwrap();
            assert_eq!(history.document, first);
            assert_eq!(history.position, position);
            assert_eq!(history.entries.unwrap().len(), 3);
        }
        assert_eq!(
            client
                .transactions()
                .jump_to(&TransactionJumpToArgs {
                    document: 0,
                    position: 0
                })
                .unwrap()
                .document,
            second
        );
    }
    assert_eq!(workspace.active_document(), Some(second));
    assert_eq!(workspace.observer().notifications, notices);
    assert_eq!(
        workspace
            .session(first)
            .unwrap()
            .snapshot()
            .parameters
            .ids()
            .len(),
        3
    );
}

#[test]
fn wire_jump_refuses_bad_target_position_and_open_group() {
    let mut workspace = Workspace::new(QueryObserver::default());
    let id = register(
        &mut workspace,
        UnitsOfMeasure::default(),
        &mut QueryGuids(0),
    );
    let host = SessionHost::new(&mut workspace);
    for request in [
        r#"{"position":-1}"#,
        r#"{"position":1}"#,
        r#"{"position":1.5}"#,
        r#"{"document":999,"position":0}"#,
        "[]",
    ] {
        assert!(
            host.call(METHOD_TRANSACTION_JUMP_TO, request.as_bytes())
                .is_err()
        );
    }
    let client = Client::new(host);
    client
        .transactions()
        .begin(&TransactionBeginArgs {
            label: "preview".into(),
        })
        .unwrap();
    assert!(
        client
            .transactions()
            .jump_to(&TransactionJumpToArgs {
                document: id,
                position: 0
            })
            .is_err()
    );
    client.transactions().abort().unwrap();
    assert_eq!(client.transactions().state().unwrap(), UndoState::default());
}
