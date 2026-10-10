// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_automation::SessionHost;
use oblikovati_model::property::{SUMMARY_INFORMATION, USER_DEFINED};

#[test]
fn typed_properties_round_trip_on_explicit_document_with_undo() {
    let mut workspace = Workspace::new(QueryObserver::default());
    let mut guids = QueryGuids(0);
    let first = register(&mut workspace, UnitsOfMeasure::default(), &mut guids);
    let second = register(&mut workspace, UnitsOfMeasure::default(), &mut guids);
    let values = [
        VariantValue::String("Title Ω".into()),
        VariantValue::Boolean(false),
        VariantValue::Integer(i64::MAX),
        VariantValue::Double(1.25),
        VariantValue::Bytes(oblikovati_api::json::Bytes(Some(vec![0, 255]))),
        VariantValue::String(String::new()),
    ];
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        for (index, value) in values.iter().enumerate() {
            let name = format!("field{index}");
            let result = client
                .documents()
                .set_property(&SetPropertyArgs {
                    document: first,
                    set: USER_DEFINED.into(),
                    name: name.clone(),
                    value: Variant {
                        value: value.clone(),
                        unit: "ignored".into(),
                    },
                })
                .unwrap();
            let expected = if matches!(value, VariantValue::Uninitialized) {
                VariantValue::String(String::new())
            } else {
                value.clone()
            };
            assert_eq!(result.property.value.value, expected);
            assert!(result.property.value.unit.is_empty());
            assert_eq!(
                client
                    .documents()
                    .get_property(&GetPropertyArgs {
                        document: first,
                        set: USER_DEFINED.into(),
                        name
                    })
                    .unwrap(),
                result
            );
        }
        let listed = client
            .documents()
            .list_properties(&ListPropertiesArgs { document: first })
            .unwrap()
            .properties
            .unwrap();
        assert_eq!(listed.len(), values.len());
        assert_eq!(listed[0].name, "field0");
        assert_eq!(listed[5].name, "field5");
        assert_eq!(
            client
                .documents()
                .list_properties(&ListPropertiesArgs { document: second })
                .unwrap()
                .properties,
            None
        );
    }
    assert_eq!(workspace.active_document(), Some(second));
    assert_eq!(workspace.session(first).unwrap().undo_labels().len(), 6);
    workspace.commands(first).unwrap().undo().unwrap();
    assert!(
        workspace
            .session(first)
            .unwrap()
            .snapshot()
            .properties
            .get(USER_DEFINED, "field5")
            .is_none()
    );
    workspace.commands(first).unwrap().redo().unwrap();
    assert!(
        workspace
            .session(first)
            .unwrap()
            .snapshot()
            .properties
            .get(USER_DEFINED, "field5")
            .is_some()
    );
    assert!(!workspace.needs_save(second).unwrap());
}

#[test]
fn property_sets_keep_standard_order_and_replacements_keep_position() {
    let mut workspace = Workspace::new(QueryObserver::default());
    let id = register(
        &mut workspace,
        UnitsOfMeasure::default(),
        &mut QueryGuids(0),
    );
    let client = Client::new(SessionHost::new(&mut workspace));
    for (set, name, value) in [
        ("Custom", "First", "a"),
        (SUMMARY_INFORMATION, "Title", "b"),
        ("Custom", "Second", "c"),
        ("Custom", "First", "d"),
        ("Custom", "First", "d"),
    ] {
        client
            .documents()
            .set_property(&SetPropertyArgs {
                document: id,
                set: set.into(),
                name: name.into(),
                value: Variant {
                    value: VariantValue::String(value.into()),
                    ..Default::default()
                },
            })
            .unwrap();
    }
    let properties = client
        .documents()
        .list_properties(&ListPropertiesArgs { document: id })
        .unwrap()
        .properties
        .unwrap();
    assert_eq!(
        properties
            .iter()
            .map(|property| property.name.as_str())
            .collect::<Vec<_>>(),
        ["Title", "First", "Second"]
    );
    assert_eq!(properties[1].value.value, VariantValue::String("d".into()));
    assert_eq!(workspace.session(id).unwrap().undo_labels().len(), 4);
}

#[test]
fn bad_properties_and_parameter_owned_writes_are_refused_without_history() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "4 cm")
        .unwrap();
    parameters
        .set_annotations(
            1,
            oblikovati_model::parameter::ParameterAnnotations {
                exposed_as_property: true,
                ..Default::default()
            },
        )
        .unwrap();
    let (mut workspace, id) = super::parameters::parameter_workspace(parameters);
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        for (document, set, name) in [
            (0, USER_DEFINED, "x"),
            (999, USER_DEFINED, "x"),
            (id, "", "x"),
            (id, USER_DEFINED, ""),
            (id, USER_DEFINED, "width"),
        ] {
            assert!(
                client
                    .documents()
                    .set_property(&SetPropertyArgs {
                        document,
                        set: set.into(),
                        name: name.into(),
                        value: Variant {
                            value: VariantValue::Integer(2),
                            ..Default::default()
                        }
                    })
                    .is_err()
            );
        }
        assert!(
            client
                .documents()
                .get_property(&GetPropertyArgs {
                    document: id,
                    set: "missing".into(),
                    name: "missing".into()
                })
                .is_err()
        );
        let property = client
            .documents()
            .get_property(&GetPropertyArgs {
                document: id,
                set: USER_DEFINED.into(),
                name: "width".into(),
            })
            .unwrap()
            .property;
        assert_eq!(property.from_parameter, "width");
        assert_eq!(property.value.value, VariantValue::Double(4.0));
    }
    assert!(workspace.session(id).unwrap().undo_labels().is_empty());
    assert!(!workspace.needs_save(id).unwrap());
}

#[test]
fn observer_veto_restores_property_sets() {
    let (mut workspace, id) = super::parameters::parameter_workspace_observer(
        ParameterCollection::new(UnitsOfMeasure::default()),
        QueryObserver {
            veto: true,
            ..Default::default()
        },
    );
    assert!(
        Client::new(SessionHost::new(&mut workspace))
            .documents()
            .set_property(&SetPropertyArgs {
                document: id,
                set: "New set".into(),
                name: "Title".into(),
                value: Variant {
                    value: VariantValue::String("vetoed".into()),
                    ..Default::default()
                }
            })
            .is_err()
    );
    assert!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .properties
            .set("New set")
            .is_none()
    );
    assert!(!workspace.needs_save(id).unwrap());
}

#[test]
fn omitted_variant_uses_go_empty_string_default() {
    let mut workspace = Workspace::new(QueryObserver::default());
    let id = register(
        &mut workspace,
        UnitsOfMeasure::default(),
        &mut QueryGuids(0),
    );
    let request = serde_json::json!({"document": id, "set": USER_DEFINED, "name": "Empty"});
    let reply = SessionHost::new(&mut workspace)
        .call(
            METHOD_DOCUMENTS_SET_PROPERTY,
            request.to_string().as_bytes(),
        )
        .unwrap();
    let result: PropertyResult = serde_json::from_slice(&reply).unwrap();
    assert_eq!(
        result.property.value.value,
        VariantValue::String(String::new())
    );
}
