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
fn selected_circle_becomes_one_instance_through_undo_redo_and_reopen() {
    let (mut files, document, sketch, mut source) = files_with_circle(Default::default());
    let before = files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .clone();
    let reply = create(
        &mut files,
        &mut source,
        &CreateBlockDefinitionArgs {
            name: "Circle".into(),
            entity_refs: Some(vec![2]),
            ..Default::default()
        },
    )
    .unwrap();
    assert_eq!((reply.entity_count, reply.instance_count), (1, 1));
    assert_grouped_circle(&files, document, sketch);
    files.commands(document).unwrap().undo().unwrap();
    let restored = files.workspace().session(document).unwrap().snapshot();
    assert!(
        restored
            .sketches
            .get(sketch)
            .unwrap()
            .geometry()
            .same_recipe(before.sketches.get(sketch).unwrap().geometry())
    );
    assert!(restored.sketches.block_definitions().is_empty());
    files.commands(document).unwrap().redo().unwrap();
    assert_grouped_circle(&files, document, sketch);
    files.save(document, "test", &mut source).unwrap();
    let reopened =
        reopen_with_observer(&mut files, document, &mut source, QueryObserver::default());
    assert_grouped_circle(&files, reopened, sketch);
}
fn assert_grouped_circle(files: &DiscoveryWorkspace, document: u64, sketch: DocumentSketchId) {
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    let placed = snapshot.sketches.get(sketch).unwrap();
    assert!(placed.geometry().circles().is_empty());
    assert!(placed.geometry().constraints().is_empty());
    assert_eq!(placed.block_instances().len(), 1);
    assert_eq!(placed.block_instances()[0].position(), [0., 0.]);
    assert_eq!(
        snapshot.sketches.block_definitions()[0]
            .geometry()
            .circles()[0]
            .radius,
        1.
    );
}

#[test]
fn grouping_rectangle_leaves_separate_circle_unchanged() {
    let (mut files, document, sketch, mut source) = files_with_circle(Default::default());
    let lines = add_rectangle(&mut files, document, sketch);
    let reply = create(
        &mut files,
        &mut source,
        &CreateBlockDefinitionArgs {
            name: "Rectangle".into(),
            entity_refs: Some(lines.iter().map(|id| id.0).collect()),
            ..Default::default()
        },
    )
    .unwrap();
    assert_eq!((reply.entity_count, reply.instance_count), (4, 1));
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    let geometry = snapshot.sketches.get(sketch).unwrap().geometry();
    assert_eq!(geometry.circles().len(), 1);
    assert_eq!(geometry.circles()[0].radius, 1.);
    assert!(geometry.lines().is_empty());
    assert_eq!(geometry.constraints().len(), 2);
    assert_eq!(
        snapshot.sketches.block_definitions()[0]
            .geometry()
            .lines()
            .len(),
        4
    );
}
fn add_rectangle(
    files: &mut DiscoveryWorkspace,
    document: u64,
    sketch: DocumentSketchId,
) -> Vec<SketchEntityId> {
    files
        .commands(document)
        .unwrap()
        .run("Rectangle", |snapshot, ids| {
            let mut geometry = snapshot.sketches.get(sketch).unwrap().geometry().clone();
            let mut points = Vec::new();
            for position in [[3., 0.], [4., 0.], [4., 1.], [3., 1.]] {
                let point = ids.allocate_sketch_entity().unwrap();
                geometry.add_defining_point(point, position).unwrap();
                points.push(point);
            }
            let mut lines = Vec::new();
            for index in 0..4 {
                let line = ids.allocate_sketch_entity().unwrap();
                geometry
                    .add_line(line, [points[index], points[(index + 1) % 4]])
                    .unwrap();
                lines.push(line);
            }
            snapshot
                .sketches
                .replace_geometry(sketch, geometry)
                .unwrap();
            Ok::<_, String>(lines)
        })
        .unwrap()
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
fn invalid_name_or_selection_does_not_change_history() {
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
fn surviving_block_instance_number_is_preserved_on_save_and_reopen() {
    let (mut files, document, sketch, mut source) = files_with_circle(Default::default());
    files
        .commands(document)
        .unwrap()
        .run("Capture block", |snapshot, _| {
            snapshot
                .sketches
                .capture_block_definition("Circle", sketch, &[SketchEntityId(2)])
        })
        .unwrap();
    let request = AddSketchBlockArgs {
        definition: "Circle".into(),
        position: Some(vec![0., 0.]),
        ..Default::default()
    };
    let first = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .sketch()
        .add_block_instance(&request)
        .unwrap();
    let second = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .sketch()
        .add_block_instance(&request)
        .unwrap();
    files
        .commands(document)
        .unwrap()
        .run("Delete first block", |snapshot, _| {
            snapshot.delete_sketch_selection_with_blocks(
                sketch,
                &Default::default(),
                &[SketchEntityId(first.entity_id)],
            )
        })
        .unwrap();
    files.save(document, "test", &mut source).unwrap();
    let reopened =
        reopen_with_observer(&mut files, document, &mut source, QueryObserver::default());
    let snapshot = files.workspace().session(reopened).unwrap().snapshot();
    let retained = &snapshot.sketches.get(sketch).unwrap().block_instances()[0];
    assert_eq!(
        (retained.id().0, retained.ordinal()),
        (second.entity_id, Some(2))
    );
}

#[test]
fn placed_instances_are_listed_and_survive_save_reopen() {
    let (mut files, document, sketch, mut source) = files_with_circle(Default::default());
    files
        .commands(document)
        .unwrap()
        .run("Capture block", |snapshot, _| {
            snapshot
                .sketches
                .capture_block_definition("Circle", sketch, &[SketchEntityId(2)])
        })
        .unwrap();
    let placed = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .sketch()
        .add_block_instance(&AddSketchBlockArgs {
            definition: "Circle".into(),
            position: Some(vec![2., 3.]),
            rotation_angle: "0.5 rad".into(),
            scale: 2.,
            ..Default::default()
        })
        .unwrap();
    assert_eq!(placed.entity_id, 3);
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
    assert_eq!(definitions(&files)[0].instance_count, 0);
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
