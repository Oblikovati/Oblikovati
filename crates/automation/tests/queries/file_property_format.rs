// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_model::property::USER_DEFINED;
use std::boxed::Box;

fn exported(files: &DiscoveryWorkspace, id: u64) -> VariantValue {
    files
        .workspace()
        .session(id)
        .unwrap()
        .snapshot()
        .properties
        .get(USER_DEFINED, "width")
        .unwrap()
        .value
        .clone()
}

#[test]
fn formatted_export_follows_units_history_and_file_reopen() {
    let (mut files, id, mut guids) = created_files(false);
    let client = Client::new(FileHost::new(&mut files, &mut guids, "test"));
    client
        .parameters()
        .update(&ParameterUpdateArgs {
            name: "width".into(),
            exposed_as_property: Some(Box::new(true)),
            ..Default::default()
        })
        .unwrap();
    assert_eq!(exported(&files, id), VariantValue::String("10 mm".into()));
    let client = Client::new(FileHost::new(&mut files, &mut guids, "test"));
    client
        .documents()
        .set_units(&SetDocumentUnitsArgs {
            length_unit: Some(Box::new("cm".into())),
            ..Default::default()
        })
        .unwrap();
    assert_eq!(exported(&files, id), VariantValue::String("1 cm".into()));
    let client = Client::new(FileHost::new(&mut files, &mut guids, "test"));
    client
        .parameters()
        .update(&ParameterUpdateArgs {
            name: "width".into(),
            custom_property_format: Some(Box::new(CustomPropertyFormatInfo {
                property_type: "number".into(),
                units: "mm".into(),
                precision: "twoDecimalPlaces".into(),
                ..Default::default()
            })),
            ..Default::default()
        })
        .unwrap();
    assert_eq!(exported(&files, id), VariantValue::Double(10.0));
    Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .transactions()
        .undo()
        .unwrap();
    assert_eq!(exported(&files, id), VariantValue::String("1 cm".into()));
    let client = Client::new(FileHost::new(&mut files, &mut guids, "test"));
    client.transactions().redo().unwrap();
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
    let property = client
        .documents()
        .get_property(&GetPropertyArgs {
            document: reopened.id,
            set: USER_DEFINED.into(),
            name: "width".into(),
        })
        .unwrap()
        .property;
    assert_eq!(property.value.value, VariantValue::Double(10.0));
    assert_eq!(property.from_parameter, "width");
    client
        .parameters()
        .set(&ParameterSetArgs {
            name: "width".into(),
            expression: "2 cm".into(),
            ..Default::default()
        })
        .unwrap();
    assert_eq!(exported(&files, reopened.id), VariantValue::Double(20.0));
}

#[test]
fn incompatible_exposed_formats_are_atomic_and_do_not_add_history() {
    let (mut files, id, mut guids) = created_files(false);
    let client = Client::new(FileHost::new(&mut files, &mut guids, "test"));
    client
        .parameters()
        .update(&ParameterUpdateArgs {
            name: "width".into(),
            exposed_as_property: Some(Box::new(true)),
            ..Default::default()
        })
        .unwrap();
    let before = files.workspace().session(id).unwrap().snapshot().clone();
    let history: Vec<String> = files
        .workspace()
        .session(id)
        .unwrap()
        .undo_labels()
        .iter()
        .map(|label| (*label).into())
        .collect();
    for (units, precision) in [
        ("deg", "threeDecimalPlaces"),
        ("cm", "seconds"),
        ("unknown", "threeDecimalPlaces"),
    ] {
        let client = Client::new(FileHost::new(&mut files, &mut guids, "test"));
        let error = client
            .parameters()
            .update(&ParameterUpdateArgs {
                name: "width".into(),
                custom_property_format: Some(Box::new(CustomPropertyFormatInfo {
                    property_type: "text".into(),
                    units: units.into(),
                    precision: precision.into(),
                    ..Default::default()
                })),
                ..Default::default()
            })
            .unwrap_err();
        assert!(error.to_string().contains("width"));
        let after = files.workspace().session(id).unwrap().snapshot();
        assert!(after.parameters.same_evaluated_state(&before.parameters));
        assert_eq!(after.properties, before.properties);
        assert_eq!(
            files.workspace().session(id).unwrap().undo_labels(),
            history
        );
    }
}
