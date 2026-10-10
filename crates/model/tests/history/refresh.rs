// SPDX-License-Identifier: GPL-2.0-only

use super::*;

#[test]
fn evaluation_refresh_preserves_redo_and_advances_generation_only_when_changed() {
    let mut history = history();
    let token = history.begin("authored").unwrap();
    set_width(&mut history, &token, "2 cm");
    history.commit(&token, false).unwrap();
    history.mark_saved().unwrap();
    history.undo().unwrap();
    let entries = history.entries();
    let generation = history.generation();
    assert!(
        history
            .refresh(|snapshot| snapshot.parameters.set_expression(1, "3 cm"))
            .unwrap()
    );
    assert_eq!(history.entries(), entries);
    assert_eq!(history.generation(), generation + 1);
    assert_eq!(history.high_water(), 1);
    assert!(
        !history
            .refresh(|snapshot| snapshot.parameters.set_expression(1, "3 cm"))
            .unwrap()
    );
    assert_eq!(history.generation(), generation + 1);
    history.redo().unwrap();
    assert_eq!(width(&history), 2.0);
}

#[test]
fn failed_refresh_and_unreserved_identity_are_atomic_and_open_groups_are_refused() {
    let mut history = history();
    let generation = history.generation();
    assert!(
        history
            .refresh(|snapshot| {
                snapshot.parameters.set_expression(1, "3 cm").unwrap();
                Err::<(), _>("failed evaluation")
            })
            .is_err()
    );
    assert!(matches!(
        history.refresh(|snapshot| snapshot.parameters.insert_expression(
            2,
            "unreserved",
            USER_PARAM,
            "1"
        )),
        Err(EditError::History(HistoryError::UnreservedIdentity(2)))
    ));
    assert_eq!(width(&history), 1.0);
    assert_eq!(history.generation(), generation);
    assert_eq!(history.high_water(), 1);
    let token = history.begin("open").unwrap();
    assert!(history.refresh(|_| Ok::<(), String>(())).is_err());
    history.abort(&token).unwrap();
}

#[test]
fn prepared_refresh_can_be_discarded_before_infallible_publication() {
    let mut history = history();
    let generation = history.generation();
    let prepared = history
        .prepare_refresh(|snapshot| snapshot.parameters.set_expression(1, "7 cm"))
        .unwrap();
    drop(prepared);
    assert_eq!(history.generation(), generation);
    assert_eq!(width(&history), 1.0);
    let prepared = history
        .prepare_refresh(|snapshot| snapshot.parameters.set_expression(1, "7 cm"))
        .unwrap();
    assert!(prepared.publish());
    assert_eq!(width(&history), 7.0);
    assert!(history.undo_labels().is_empty());
}

#[test]
fn attribute_refresh_detects_signed_zero_and_publishes_only_real_changes() {
    use oblikovati_model::attribute::AttributeTarget;
    let mut history = history();
    let target = AttributeTarget::from_encoded(&[0; 16]).unwrap();
    let generation = history.generation();
    assert!(
        history
            .refresh(|snapshot| snapshot.attributes.put(
                &target,
                "AddIn",
                "zero",
                VariantValue::Double(-0.0)
            ))
            .unwrap()
    );
    assert_eq!(history.generation(), generation + 1);
    assert!(
        !history
            .refresh(|snapshot| snapshot.attributes.put(
                &target,
                "AddIn",
                "zero",
                VariantValue::Double(-0.0)
            ))
            .unwrap()
    );
    assert!(
        history
            .refresh(|snapshot| snapshot.attributes.put(
                &target,
                "AddIn",
                "zero",
                VariantValue::Double(0.0)
            ))
            .unwrap()
    );
    assert_eq!(history.generation(), generation + 2);
    assert!(history.entries().is_empty());
}

#[test]
fn interest_refresh_publishes_changed_metadata_without_an_undo_step() {
    use oblikovati_api::types::DocumentInterestRecord;
    let mut history = history();
    let generation = history.generation();
    let record = DocumentInterestRecord {
        client_id: "cam".into(),
        name: "paths".into(),
        ..Default::default()
    };
    for expected_change in [true, false] {
        assert_eq!(
            history
                .refresh(|snapshot| snapshot.interests.put(record.clone()))
                .unwrap(),
            expected_change
        );
        assert_eq!(history.generation(), generation + 1);
    }
    assert!(history.entries().is_empty());
}
