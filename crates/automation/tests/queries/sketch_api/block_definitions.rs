// SPDX-License-Identifier: GPL-2.0-only
use super::*;

fn definitions(files: &DiscoveryWorkspace) -> Vec<SketchBlockDefinitionInfo> {
    Client::new(FileQueryHost::new(files))
        .sketch()
        .block_definitions()
        .unwrap()
        .definitions
        .unwrap()
}
fn create(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    request: &CreateBlockDefinitionArgs,
) -> Result<SketchBlockDefinitionInfo, oblikovati_api::Error> {
    Client::new(FileHost::new(files, source, "test"))
        .sketch()
        .create_block_definition(request)
}
#[test]
fn empty_definition_survives_history_and_save_reopen() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let request = CreateBlockDefinitionArgs {
        name: "Rib".into(),
        ..Default::default()
    };
    let reply = create(&mut files, &mut source, &request).unwrap();
    assert_eq!(
        (reply.index, reply.entity_count, reply.instance_count),
        (0, 0, 0)
    );
    assert_eq!(definitions(&files)[0].name, "Rib");
    files.commands(document).unwrap().undo().unwrap();
    assert!(definitions(&files).is_empty());
    files.commands(document).unwrap().redo().unwrap();
    files.save(document, "test", &mut source).unwrap();
    let reopened =
        reopen_with_observer(&mut files, document, &mut source, QueryObserver::default());
    assert_eq!(definitions(&files)[0].name, "Rib");
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .sketch()
        .delete_block_definition(&DeleteBlockDefinitionArgs { name: "Rib".into() })
        .unwrap();
    assert!(definitions(&files).is_empty());
    files.save(reopened, "test", &mut source).unwrap();
    reopen_with_observer(&mut files, reopened, &mut source, QueryObserver::default());
    assert!(definitions(&files).is_empty());
}
#[test]
fn invalid_or_unimplemented_selection_does_not_change_history() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let before = owner_revision(&files, document);
    for request in [
        CreateBlockDefinitionArgs {
            name: "".into(),
            ..Default::default()
        },
        CreateBlockDefinitionArgs {
            name: "Rib".into(),
            entity_refs: Some(vec![7]),
            ..Default::default()
        },
    ] {
        assert!(create(&mut files, &mut source, &request).is_err());
        assert_eq!(owner_revision(&files, document), before);
        assert!(definitions(&files).is_empty());
    }
}

#[test]
fn duplicate_and_missing_names_do_not_change_the_registry_or_history() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let request = CreateBlockDefinitionArgs {
        name: "Rib".into(),
        ..Default::default()
    };
    create(&mut files, &mut source, &request).unwrap();
    let before = owner_revision(&files, document);
    assert!(create(&mut files, &mut source, &request).is_err());
    assert!(
        Client::new(FileHost::new(&mut files, &mut source, "test"))
            .sketch()
            .delete_block_definition(&DeleteBlockDefinitionArgs {
                name: "missing".into()
            })
            .is_err()
    );
    assert_eq!(owner_revision(&files, document), before);
    assert_eq!(
        definitions(&files)
            .iter()
            .map(|row| row.name.as_str())
            .collect::<Vec<_>>(),
        ["Rib"]
    );
}

#[test]
fn listing_reports_captured_definition_geometry() {
    let (mut files, document, sketch, _) = files_with_circle(Default::default());
    files
        .commands(document)
        .unwrap()
        .run("Capture definition", |snapshot, _| {
            snapshot
                .sketches
                .capture_block_definition("Circle", sketch, &[SketchEntityId(2)])
        })
        .unwrap();
    let rows = definitions(&files);
    assert_eq!(rows.len(), 1);
    assert_eq!(rows[0].entity_count, 1);
    assert_eq!(rows[0].instance_count, 0);
}

#[test]
fn placed_instances_are_listed_and_survive_save_reopen() {
    use oblikovati_model::document_sketch::SketchBlockInstance;
    let (mut files, document, sketch, mut source) = files_with_circle(Default::default());
    files
        .commands(document)
        .unwrap()
        .run("Place block", |snapshot, ids| -> Result<(), String> {
            snapshot
                .sketches
                .capture_block_definition("Circle", sketch, &[SketchEntityId(2)])
                .map_err(|e| e.to_string())?;
            let id = ids.allocate_sketch_entity().map_err(|e| e.to_string())?;
            let instance = SketchBlockInstance::new(id, "Circle", [2., 3.], 0.5, 2.)
                .map_err(|e| e.to_string())?;
            snapshot
                .sketches
                .insert_block_instance(sketch, instance)
                .map_err(|e| e.to_string())
        })
        .unwrap();
    let rows = Client::new(FileQueryHost::new(&files))
        .sketch()
        .block_instances(&SketchArgs::default())
        .unwrap()
        .instances
        .unwrap();
    assert_eq!(rows.len(), 1);
    assert_eq!(
        (
            rows[0].id,
            rows[0].position.clone(),
            rows[0].rotation,
            rows[0].scale,
            rows[0].entity_count
        ),
        (3, Some(vec![2., 3.]), 0.5, 2., 1)
    );
    assert_eq!(definitions(&files)[0].instance_count, 1);
    files.commands(document).unwrap().undo().unwrap();
    assert!(definitions(&files).is_empty());
    files.commands(document).unwrap().redo().unwrap();
    assert_eq!(definitions(&files)[0].instance_count, 1);
    files.save(document, "test", &mut source).unwrap();
    let reopened =
        reopen_with_observer(&mut files, document, &mut source, QueryObserver::default());
    assert_eq!(definitions(&files)[0].instance_count, 1);
    let error = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .sketch()
        .delete_block_definition(&DeleteBlockDefinitionArgs {
            name: "Circle".into(),
        })
        .unwrap_err();
    assert!(error.to_string().contains("Driven circle"));
    assert_eq!(
        files
            .workspace()
            .session(reopened)
            .unwrap()
            .snapshot()
            .sketches
            .entity_ids(),
        [1, 2, 3]
    );
}
