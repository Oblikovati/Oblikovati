// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::attribute::AttributeTarget;

#[test]
fn attribute_edits_participate_in_history_noop_detection_and_abort() {
    let mut session = session(None);
    let target = AttributeTarget::from_encoded(&[0; 16]).unwrap();
    for _ in 0..2 {
        session.begin("Tag").unwrap();
        session
            .edit(|snapshot, _| {
                snapshot
                    .attributes
                    .put(&target, "AddIn", "tag", VariantValue::Integer(7))
            })
            .unwrap();
        session.end().unwrap();
    }
    assert_eq!(session.undo_labels(), ["Tag"]);
    session.undo().unwrap();
    assert!(session.snapshot().attributes.anchors().is_empty());
    session.redo().unwrap();
    let saved = session.snapshot().attributes.clone();
    session.begin("Remove").unwrap();
    session
        .edit(|snapshot, _| {
            snapshot.attributes.remove_target(&target);
            Ok::<(), String>(())
        })
        .unwrap();
    session.abort().unwrap();
    assert!(session.snapshot().attributes.same_recipe(&saved));
}

#[test]
fn attribute_commit_veto_restores_the_original_collection() {
    let mut session = session(Some(EVENT_TRANSACTION_COMMITTED));
    let target = AttributeTarget::from_encoded(&[0; 16]).unwrap();
    session.begin("Tag").unwrap();
    session
        .edit(|snapshot, _| {
            snapshot
                .attributes
                .put(&target, "AddIn", "tag", VariantValue::Boolean(true))
        })
        .unwrap();
    assert!(session.end().is_err());
    assert!(session.snapshot().attributes.anchors().is_empty());
    assert!(session.undo_labels().is_empty());
}
