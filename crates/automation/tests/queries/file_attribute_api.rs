// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn typed_attribute_api_preserves_targets_order_history_and_saved_values() {
    let (mut files, id, mut guids) = created_files(false);
    let client = Client::new(FileHost::new(&mut files, &mut guids, "test"));
    let missing = client
        .attributes()
        .get_on(&GetAttributeArgs {
            document: id,
            set: "AddIn".into(),
            name: "tag".into(),
            ..Default::default()
        })
        .unwrap();
    assert!(!missing.found);
    assert_eq!(
        client
            .attributes()
            .list_sets(&ListAttributeSetsArgs { document: id })
            .unwrap()
            .sets,
        Some(Vec::new())
    );
    assert_eq!(
        missing.attribute.value.value,
        VariantValue::String(String::new())
    );
    for (set, target, number) in [("Z", "", 1), ("A", "", 2), ("Z", "face:opaque", 3)] {
        let result = client
            .attributes()
            .set_on(&SetAttributeArgs {
                document: id,
                set: set.into(),
                name: "tag".into(),
                target: target.into(),
                value: Variant {
                    value: VariantValue::Integer(number),
                    unit: "ignored".into(),
                },
            })
            .unwrap();
        assert!(result.found);
        assert_eq!(result.attribute.target, target);
        assert!(result.attribute.value.unit.is_empty());
    }
    assert_eq!(
        client
            .attributes()
            .list_sets(&ListAttributeSetsArgs { document: id })
            .unwrap()
            .sets
            .unwrap(),
        ["A", "Z"]
    );
    assert_eq!(
        client
            .attributes()
            .list(&ListAttributesArgs {
                document: id,
                ..Default::default()
            })
            .unwrap()
            .attributes
            .unwrap()
            .len(),
        2
    );
    let all = client
        .attributes()
        .list(&ListAttributesArgs {
            document: id,
            all_targets: true,
            set: "Z".into(),
            ..Default::default()
        })
        .unwrap()
        .attributes
        .unwrap();
    assert_eq!(
        all.iter()
            .map(|entry| entry.target.as_str())
            .collect::<Vec<_>>(),
        ["", "face:opaque"]
    );
    assert_eq!(
        client
            .attributes()
            .find(&FindByAttributeArgs {
                set: "Z".into(),
                name: "tag".into()
            })
            .unwrap()
            .matches
            .unwrap()
            .len(),
        2
    );
    assert_eq!(
        client
            .attributes()
            .delete_on(&DeleteAttributeArgs {
                document: id,
                set: "Z".into(),
                target: "face:opaque".into(),
                name: "tag".into()
            })
            .unwrap()
            .removed,
        1
    );
    client.transactions().undo().unwrap();
    client
        .documents()
        .save(&SaveDocumentArgs { document: id })
        .unwrap();
    client
        .documents()
        .close(&CloseDocumentArgs { id, force: false })
        .unwrap();
    let mut factory = QueryObserverFactory {
        created: 0,
        limit: 1,
    };
    let client = Client::new(
        FileHost::new(&mut files, &mut guids, "test").with_observer_factory(&mut factory),
    );
    let reopened = client
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: "/fixtures/part.opd".into(),
            ..Default::default()
        })
        .unwrap();
    assert_eq!(
        client
            .attributes()
            .get_on(&GetAttributeArgs {
                document: reopened.id,
                set: "Z".into(),
                name: "tag".into(),
                target: "face:opaque".into()
            })
            .unwrap()
            .attribute
            .value
            .value,
        VariantValue::Integer(3)
    );
    assert_eq!(
        client
            .attributes()
            .delete_on(&DeleteAttributeArgs {
                document: reopened.id,
                set: "Z".into(),
                ..Default::default()
            })
            .unwrap()
            .removed,
        1
    );
    assert_eq!(
        client
            .attributes()
            .delete_on(&DeleteAttributeArgs {
                document: reopened.id,
                set: "Z".into(),
                ..Default::default()
            })
            .unwrap()
            .removed,
        0
    );
}

#[test]
fn attribute_reads_and_invalid_edits_do_not_change_snapshot_or_history() {
    let (mut files, id, mut guids) = created_files(false);
    let client = Client::new(FileHost::new(&mut files, &mut guids, "test"));
    assert!(
        client
            .attributes()
            .list(&ListAttributesArgs {
                document: id,
                target: "unseen".into(),
                ..Default::default()
            })
            .unwrap()
            .attributes
            .is_none()
    );
    assert!(
        client
            .attributes()
            .get_on(&GetAttributeArgs {
                document: id,
                target: "unseen".into(),
                ..Default::default()
            })
            .is_ok()
    );
    assert!(
        client
            .attributes()
            .find(&FindByAttributeArgs::default())
            .is_err()
    );
    assert!(
        client
            .attributes()
            .get_on(&GetAttributeArgs {
                document: 999,
                ..Default::default()
            })
            .is_err()
    );
    for (set, name) in [("", "tag"), ("AddIn", "")] {
        assert!(
            client
                .attributes()
                .set_on(&SetAttributeArgs {
                    document: id,
                    set: set.into(),
                    name: name.into(),
                    value: Variant {
                        value: VariantValue::String(String::new()),
                        unit: String::new()
                    },
                    ..Default::default()
                })
                .is_err()
        );
    }
    assert!(
        files
            .workspace()
            .session(id)
            .unwrap()
            .snapshot()
            .attributes
            .anchors()
            .is_empty()
    );
    assert!(
        files
            .workspace()
            .session(id)
            .unwrap()
            .undo_labels()
            .is_empty()
    );
}

#[test]
fn vetoed_attribute_commands_leave_the_document_and_history_unchanged() {
    let (mut workspace, id) = crate::parameters::parameter_workspace_observer(
        ParameterCollection::new(UnitsOfMeasure::default()),
        QueryObserver {
            veto: true,
            ..Default::default()
        },
    );
    let client = Client::new(oblikovati_automation::SessionHost::new(&mut workspace));
    let error = client
        .attributes()
        .set_on(&SetAttributeArgs {
            document: id,
            set: "AddIn".into(),
            name: "tag".into(),
            value: Variant {
                value: VariantValue::String(String::new()),
                unit: String::new(),
            },
            ..Default::default()
        })
        .unwrap_err();
    assert!(error.to_string().contains("settings are locked"));
    assert!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .attributes
            .anchors()
            .is_empty()
    );
    assert!(workspace.session(id).unwrap().undo_labels().is_empty());
}

#[test]
fn find_attributes_searches_all_documents_in_workspace_order() {
    let (mut files, first, mut guids) = created_files(false);
    let second = files
        .create(
            Path::new("/fixtures/second.opd"),
            DocumentPackage::empty(DOCUMENT_PART, "Second"),
            QueryObserver::default(),
            "test",
            &mut guids,
        )
        .unwrap()
        .session_id;
    let client = Client::new(FileHost::new(&mut files, &mut guids, "test"));
    for document in [second, first] {
        client
            .attributes()
            .set_on(&SetAttributeArgs {
                document,
                set: "AddIn".into(),
                name: "tag".into(),
                value: Variant {
                    value: VariantValue::String(String::new()),
                    unit: String::new(),
                },
                ..Default::default()
            })
            .unwrap();
    }
    let matches = client
        .attributes()
        .find(&FindByAttributeArgs {
            set: "AddIn".into(),
            ..Default::default()
        })
        .unwrap()
        .matches
        .unwrap();
    assert_eq!(
        matches.iter().map(|hit| hit.document).collect::<Vec<_>>(),
        [first, second]
    );
    assert!(
        matches
            .iter()
            .all(|hit| hit.attribute.value.value == VariantValue::String(String::new()))
    );
}
