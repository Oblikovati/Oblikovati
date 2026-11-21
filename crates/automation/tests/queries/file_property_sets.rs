// SPDX-License-Identifier: GPL-2.0-only

use super::*;

#[test]
fn removing_the_last_property_preserves_its_set_through_file_save_and_reopen() {
    let (mut files, id, mut guids) = created_files(false);
    files
        .commands(id)
        .unwrap()
        .run("Custom set", |snapshot, _| {
            snapshot
                .properties
                .put("Reserved", "temporary", VariantValue::Integer(1))?;
            assert!(snapshot.properties.remove("Reserved", "temporary"));
            snapshot.properties.ensure_set("Next")
        })
        .unwrap();
    let expected = files
        .workspace()
        .session(id)
        .unwrap()
        .snapshot()
        .properties
        .clone();
    let client = Client::new(FileHost::new(&mut files, &mut guids, "test"));
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
        files
            .workspace()
            .session(reopened.id)
            .unwrap()
            .snapshot()
            .properties,
        expected
    );
}

#[test]
fn external_property_write_survives_switching_model_states() {
    use oblikovati_api::{
        json::EmptyObject,
        types::{ModelStatePropertyKey, Variant},
        wire::{CreateModelStateArgs, RepRef, SetPropertyArgs},
    };
    let (mut files, id, mut guids) = created_files(false);
    let client = Client::new(FileHost::new(&mut files, &mut guids, "test"));
    let second = client
        .model_states()
        .create(&CreateModelStateArgs {
            name: "Second".into(),
            ..Default::default()
        })
        .unwrap()
        .model_state
        .id;
    let view = client
        .model_states()
        .get_table(&EmptyObject::default())
        .unwrap();
    let key = ModelStatePropertyKey {
        set: "Summary Information".into(),
        name: "Category".into(),
    };
    let draft = with_property_column(view, key.clone(), &[]);
    client.model_states().set_table(&draft).unwrap();
    client
        .model_states()
        .activate(&RepRef { id: second })
        .unwrap();
    client
        .documents()
        .set_property(&SetPropertyArgs {
            document: id,
            set: key.set.clone(),
            name: key.name.clone(),
            value: Variant {
                value: VariantValue::String("External".into()),
                unit: String::new(),
            },
        })
        .unwrap();
    client.model_states().activate(&RepRef { id: 1 }).unwrap();
    assert!(
        client
            .documents()
            .get_property(&GetPropertyArgs {
                document: id,
                set: key.set.clone(),
                name: key.name.clone()
            })
            .is_err()
    );
    client
        .model_states()
        .activate(&RepRef { id: second })
        .unwrap();
    let value = client
        .documents()
        .get_property(&GetPropertyArgs {
            document: id,
            set: key.set,
            name: key.name,
        })
        .unwrap()
        .property
        .value
        .value;
    assert_eq!(value, VariantValue::String("External".into()));
}

#[test]
fn untracked_snapshot_property_write_is_rejected() {
    use oblikovati_api::{json::EmptyObject, types::ModelStatePropertyKey};
    let (mut files, id, mut guids) = created_files(false);
    let client = Client::new(FileHost::new(&mut files, &mut guids, "test"));
    let view = client
        .model_states()
        .get_table(&EmptyObject::default())
        .unwrap();
    let key = ModelStatePropertyKey {
        set: "Summary Information".into(),
        name: "Category".into(),
    };
    let draft = with_property_column(view, key.clone(), &[]);
    client.model_states().set_table(&draft).unwrap();
    drop(client);
    let error = files
        .commands(id)
        .unwrap()
        .run("Bypass", |snapshot, _| {
            snapshot
                .properties
                .put(&key.set, &key.name, VariantValue::String("Lost".into()))
        })
        .unwrap_err();
    assert!(
        error
            .to_string()
            .contains("expected the active member value")
    );
    assert!(
        files
            .workspace()
            .session(id)
            .unwrap()
            .snapshot()
            .properties
            .get(&key.set, &key.name)
            .is_none()
    );
}

#[test]
fn tracked_property_cannot_be_taken_over_by_an_exposed_parameter() {
    use oblikovati_api::{
        json::EmptyObject,
        types::{ModelStatePropertyKey, Variant},
        wire::SetPropertyArgs,
    };
    let (mut files, id, mut guids) = created_files(false);
    let client = Client::new(FileHost::new(&mut files, &mut guids, "test"));
    let key = ModelStatePropertyKey {
        set: "User Defined Properties".into(),
        name: "width".into(),
    };
    let property = client
        .documents()
        .set_property(&SetPropertyArgs {
            document: id,
            set: key.set.clone(),
            name: key.name.clone(),
            value: Variant {
                value: VariantValue::String("Independent".into()),
                unit: String::new(),
            },
        })
        .unwrap()
        .property;
    let view = client
        .model_states()
        .get_table(&EmptyObject::default())
        .unwrap();
    let draft = with_property_column(view, key.clone(), &[]);
    assert_eq!(
        draft
            .properties
            .as_ref()
            .unwrap()
            .last()
            .unwrap()
            .base
            .as_deref(),
        Some(&property.value)
    );
    client.model_states().set_table(&draft).unwrap();
    drop(client);
    let result = files
        .commands(id)
        .unwrap()
        .run("Expose width", |snapshot, _| {
            let parameter = snapshot.parameters.find("width").unwrap();
            let mut annotations = snapshot
                .parameters
                .get(parameter)
                .unwrap()
                .annotations
                .clone();
            annotations.exposed_as_property = true;
            snapshot.parameters.set_annotations(parameter, annotations)
        });
    let error = result.unwrap_err().to_string();
    assert!(
        error.contains("expected a property not owned by a parameter"),
        "{error}"
    );
    let snapshot = files.workspace().session(id).unwrap().snapshot();
    assert!(
        !snapshot
            .parameters
            .get(snapshot.parameters.find("width").unwrap())
            .unwrap()
            .annotations
            .exposed_as_property
    );
    assert_eq!(
        snapshot.properties.get(&key.set, &key.name).unwrap().value,
        VariantValue::String("Independent".into())
    );
}
