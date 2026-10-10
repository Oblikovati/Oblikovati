// SPDX-License-Identifier: GPL-2.0-only

use super::*;

#[test]
fn owned_refresh_changes_values_and_health_without_target_undo_events() {
    let (mut files, source_id, target, _) = linked_files();
    let generation = files.workspace().session(target).unwrap().generation();
    let notices = files
        .workspace()
        .session(target)
        .unwrap()
        .observer()
        .notifications;
    assert!(files.refresh_derived_table(target, 7).unwrap());
    let session = files.workspace().session(target).unwrap();
    assert_eq!(session.generation(), generation + 1);
    assert_eq!(session.observer().notifications, notices + 1);
    assert!(session.undo_labels().is_empty());
    assert!(!files.refresh_derived_table(target, 7).unwrap());
    files
        .commands(source_id)
        .unwrap()
        .run("Width", |snapshot, _| {
            snapshot.parameters.set_expression(1, "5 cm")
        })
        .unwrap();
    assert!(files.refresh_derived_table(target, 7).unwrap());
    assert_eq!(
        files
            .workspace()
            .session(target)
            .unwrap()
            .snapshot()
            .parameters
            .get(1)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        5.0
    );
    assert_eq!(files.workspace().active_document(), Some(target));
    assert!(
        files
            .workspace()
            .session(target)
            .unwrap()
            .undo_labels()
            .is_empty()
    );
}

#[test]
fn refresh_preserves_redo_and_refuses_open_groups_before_adoption() {
    let (mut files, source_id, target, _) = linked_files();
    files
        .commands(target)
        .unwrap()
        .run("Local name", |snapshot, _| {
            snapshot.parameters.rename(1, "local_width")
        })
        .unwrap();
    files.commands(target).unwrap().undo().unwrap();
    let entries = files.workspace().session(target).unwrap().history_entries();
    files
        .commands(source_id)
        .unwrap()
        .run("Width", |snapshot, _| {
            snapshot.parameters.set_expression(1, "6 cm")
        })
        .unwrap();
    files.commands(target).unwrap().begin("pending").unwrap();
    let generation = files.workspace().session(target).unwrap().generation();
    assert!(files.refresh_derived_table(target, 7).is_err());
    assert_eq!(
        files.workspace().session(target).unwrap().generation(),
        generation
    );
    files.commands(target).unwrap().abort().unwrap();
    assert!(files.refresh_derived_table(target, 7).unwrap());
    assert_eq!(
        files.workspace().session(target).unwrap().history_entries(),
        entries
    );
    files.commands(target).unwrap().redo().unwrap();
    assert!(files.refresh_derived_table(target, 7).unwrap());
    let parameter = files
        .workspace()
        .session(target)
        .unwrap()
        .snapshot()
        .parameters
        .get(1)
        .unwrap();
    assert_eq!(parameter.name, "local_width");
    assert_eq!(parameter.current_value().unwrap().value, 6.0);
}

#[test]
fn unavailable_and_invalid_sources_keep_failure_local() {
    let (mut files, source_id, target, _) = linked_files();
    assert!(files.refresh_derived_table(999, 7).is_err());
    assert!(files.refresh_derived_table(target, 999).is_err());
    files.close_discarding_changes(source_id).unwrap();
    assert!(files.refresh_derived_table(target, 7).unwrap());
    assert!(
        files
            .workspace()
            .session(target)
            .unwrap()
            .snapshot()
            .parameters
            .get(1)
            .unwrap()
            .current_value()
            .is_none()
    );
    assert!(!files.refresh_derived_table(target, 7).unwrap());
    assert!(
        files
            .workspace()
            .session(target)
            .unwrap()
            .undo_labels()
            .is_empty()
    );
}

#[test]
fn self_source_is_refused_without_changes() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let target = files.create(Path::new("/self.opd"), DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\nmodel:\n  derivedParameterTables: [{id: 1, sourceDocument: /self.opd}]\n").unwrap(), QueryObserver::default(), "test", &mut QueryGuids(0)).unwrap().session_id;
    let error = files
        .refresh_derived_table(target, 1)
        .unwrap_err()
        .to_string();
    assert!(error.contains("different document"));
    assert_eq!(files.workspace().session(target).unwrap().generation(), 0);
}

#[test]
fn refresh_updates_exposed_properties_and_publishes_source_failure() {
    let (mut files, source_id, target, mut source) = linked_files();
    files
        .commands(target)
        .unwrap()
        .run("Expose", |snapshot, _| {
            let mut annotations = snapshot.parameters.get(1).unwrap().annotations.clone();
            annotations.exposed_as_property = true;
            snapshot.parameters.set_annotations(1, annotations)
        })
        .unwrap();
    files
        .commands(source_id)
        .unwrap()
        .run("Width", |snapshot, _| {
            snapshot.parameters.set_expression(1, "9 cm")
        })
        .unwrap();
    assert!(files.refresh_derived_table(target, 7).unwrap());
    let snapshot = files.workspace().session(target).unwrap().snapshot();
    assert_eq!(
        snapshot
            .properties
            .get(oblikovati_model::property::USER_DEFINED, "width")
            .unwrap()
            .value,
        VariantValue::String("90 mm".into())
    );
    let generation = files.workspace().session(target).unwrap().generation();
    files.close_discarding_changes(source_id).unwrap();
    assert!(files.refresh_derived_table(target, 7).unwrap());
    let session = files.workspace().session(target).unwrap();
    assert!(
        session
            .snapshot()
            .parameters
            .get(1)
            .unwrap()
            .current_value()
            .is_none()
    );
    assert!(
        session
            .snapshot()
            .properties
            .get(oblikovati_model::property::USER_DEFINED, "width")
            .is_none()
    );
    assert_eq!(session.generation(), generation + 1);
    assert_eq!(session.undo_labels(), ["Expose"]);
    let error = Client::new(FileQueryHost::new(&files))
        .documents()
        .get_property(&GetPropertyArgs {
            document: target,
            set: oblikovati_model::property::USER_DEFINED.into(),
            name: "width".into(),
        })
        .unwrap_err()
        .to_string();
    assert!(error.contains("unhealthy") && error.contains("width"));
    let error = Client::new(oblikovati_automation::FileHost::new(
        &mut files,
        &mut source,
        "test",
    ))
    .documents()
    .set_property(&SetPropertyArgs {
        document: target,
        set: oblikovati_model::property::USER_DEFINED.into(),
        name: "width".into(),
        value: Variant {
            value: VariantValue::Double(12.0),
            ..Default::default()
        },
    })
    .unwrap_err()
    .to_string();
    assert!(error.contains("owned by an exposed parameter"));
}
