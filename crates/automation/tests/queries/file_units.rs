// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use std::boxed::Box;

#[test]
fn unit_preferences_are_partial_undoable_and_do_not_rescale_parameters() {
    let (mut files, id, mut source) = created_files(false);
    let stored = files
        .workspace()
        .session(id)
        .unwrap()
        .snapshot()
        .parameters
        .get(1)
        .unwrap()
        .clone();
    let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
    let before = client.documents().get_units().unwrap();
    let width = client
        .parameters()
        .get(&ParameterNameArgs {
            name: "width".into(),
        })
        .unwrap();
    let updated = client
        .documents()
        .set_units(&SetDocumentUnitsArgs {
            length_unit: Some(Box::new("in".into())),
            angle_unit: Some(Box::new("rad".into())),
            mass_unit: Some(Box::new("g".into())),
            time_unit: Some(Box::new("min".into())),
            length_display_precision: Some(Box::new(4)),
            angle_display_precision: Some(Box::new(5)),
            length_display_format: Some(Box::new("fractional".into())),
        })
        .unwrap();
    assert_eq!(updated.length_unit, "in");
    assert_eq!(updated.angle_unit, "rad");
    assert_eq!(updated.mass_unit, "g");
    assert_eq!(updated.time_unit, "min");
    assert_eq!(updated.length_display_precision, 4);
    assert_eq!(updated.angle_display_precision, 5);
    assert_eq!(updated.length_display_format, "fractional");
    assert_eq!(updated.working_scale_cm, before.working_scale_cm);
    assert_eq!(
        client
            .parameters()
            .get(&ParameterNameArgs {
                name: "width".into()
            })
            .unwrap()
            .expression,
        width.expression
    );
    assert_eq!(
        client
            .documents()
            .set_units(&SetDocumentUnitsArgs::default())
            .unwrap(),
        updated
    );
    client.transactions().undo().unwrap();
    assert_eq!(client.documents().get_units().unwrap(), before);
    client.transactions().redo().unwrap();
    assert_eq!(client.documents().get_units().unwrap(), updated);
    assert_eq!(
        client
            .transactions()
            .history(&TransactionHistoryArgs { document: id })
            .unwrap()
            .entries
            .unwrap()
            .len(),
        1
    );
    assert_eq!(
        files
            .workspace()
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .get(1)
            .unwrap(),
        &stored
    );
}

#[test]
fn invalid_unit_updates_are_atomic_and_leave_no_undo_entries() {
    let (mut files, _, mut source) = created_files(false);
    let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
    let before = client.documents().get_units().unwrap();
    for request in [
        br#"{"lengthUnit":"in","angleUnit":"mm"}"#.as_slice(),
        br#"{"lengthUnit":"","massUnit":"g"}"#,
        br#"{"lengthUnit":"in","lengthDisplayPrecision":-1}"#,
        br#"{"lengthUnit":"in","angleDisplayPrecision":256}"#,
        br#"{"lengthUnit":"in","lengthDisplayFormat":"bad"}"#,
    ] {
        assert!(
            client
                .documents()
                .set_units(&serde_json::from_slice(request).unwrap())
                .is_err()
        );
        assert_eq!(client.documents().get_units().unwrap(), before);
    }
    assert!(
        client
            .transactions()
            .history(&TransactionHistoryArgs::default())
            .unwrap()
            .entries
            .unwrap()
            .is_empty()
    );
}

#[test]
fn unit_preferences_survive_save_close_and_reopen() {
    let (mut files, id, mut source) = created_files(false);
    let mut factory = QueryObserverFactory {
        created: 0,
        limit: 1,
    };
    let client = Client::new(
        FileHost::new(&mut files, &mut source, "test").with_observer_factory(&mut factory),
    );
    let updated = client
        .documents()
        .set_units(&SetDocumentUnitsArgs {
            length_unit: Some(Box::new("in".into())),
            length_display_format: Some(Box::new("architectural".into())),
            ..Default::default()
        })
        .unwrap();
    client
        .documents()
        .close(&CloseDocumentArgs { id, force: false })
        .unwrap();
    client
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: "/fixtures/part.opd".into(),
            ..Default::default()
        })
        .unwrap();
    assert_eq!(client.documents().get_units().unwrap(), updated);
    let decimal = client
        .documents()
        .set_units(&SetDocumentUnitsArgs {
            length_display_format: Some(Box::new("decimal".into())),
            ..Default::default()
        })
        .unwrap();
    assert_eq!(decimal.length_unit, "in");
    assert_eq!(decimal.length_display_format, "decimal");
}
