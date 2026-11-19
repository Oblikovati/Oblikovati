// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn typed_explode_survives_undo_redo_and_reopen() {
    let (mut files, document, sketch, mut source) = files_with_circle(Default::default());
    let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
    client
        .sketch()
        .create_block_definition(&CreateBlockDefinitionArgs {
            name: "Circle".into(),
            entity_refs: Some(vec![2]),
            ..Default::default()
        })
        .unwrap();
    let second = client
        .sketch()
        .add_block_instance(&AddSketchBlockArgs {
            definition: "Circle".into(),
            position: Some(vec![3.0, 4.0]),
            scale: 2.0,
            ..Default::default()
        })
        .unwrap()
        .entity_id;
    let args = ExplodeSketchBlockArgs {
        sketch_index: 0,
        instance_id: second,
    };
    let result = client.sketch().explode_block_instance(&args).unwrap();
    let circle = result.entity_refs.unwrap()[0];
    assert_exploded(&files, document, sketch, circle);
    files.commands(document).unwrap().undo().unwrap();
    let placed = files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .get(sketch)
        .unwrap();
    assert_eq!(placed.block_instances().len(), 2);
    assert!(placed.geometry().circles().is_empty());
    files.commands(document).unwrap().redo().unwrap();
    assert_exploded(&files, document, sketch, circle);
    files.save(document, "test", &mut source).unwrap();
    let reopened =
        reopen_with_observer(&mut files, document, &mut source, QueryObserver::default());
    assert_exploded(&files, reopened, sketch, circle);
}

fn assert_exploded(files: &DiscoveryWorkspace, document: u64, sketch: DocumentSketchId, id: u64) {
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    let placed = snapshot.sketches.get(sketch).unwrap();
    assert_eq!(placed.block_instances().len(), 1);
    assert_eq!(snapshot.sketches.block_definitions().len(), 1);
    let circle = placed.geometry().circle(SketchEntityId(id)).unwrap();
    assert_eq!(circle.radius, 2.0);
    assert_eq!(
        placed.geometry().point(circle.center).unwrap().position,
        [3.0, 4.0]
    );
}

#[test]
fn typed_explode_rejects_missing_instances_without_changing_history() {
    let (mut files, document, _, mut source) = files_with_circle(Default::default());
    let before = owner_revision(&files, document);
    let request = ExplodeSketchBlockArgs {
        sketch_index: 0,
        instance_id: 999,
    };
    assert!(
        Client::new(FileHost::new(&mut files, &mut source, "test"))
            .sketch()
            .explode_block_instance(&request)
            .is_err()
    );
    assert_eq!(owner_revision(&files, document), before);
}
