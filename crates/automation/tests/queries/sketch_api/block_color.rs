// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::{Color, OVERRIDE_COLOR_SOURCE};

fn color(r: u8, g: u8, b: u8) -> Color {
    Color {
        r,
        g,
        b,
        opacity: 1.0,
        source: OVERRIDE_COLOR_SOURCE,
    }
}

#[test]
fn block_colors_are_listed_and_survive_undo_redo_and_reopen() {
    let (mut files, document, sketch, mut source) = files_with_circle(Default::default());
    let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
    client
        .sketch()
        .create_block_definition(&CreateBlockDefinitionArgs {
            name: "Rib".into(),
            entity_refs: Some(vec![2]),
            ..Default::default()
        })
        .unwrap();
    let instance = client
        .sketch()
        .block_instances(&SketchArgs::default())
        .unwrap()
        .instances
        .unwrap()[0]
        .id;
    let cyan = color(0, 255, 255);
    let red = color(255, 0, 0);
    let definition = client
        .sketch()
        .set_block_definition_color(&SetBlockDefinitionColorArgs {
            name: "Rib".into(),
            color: Some(std::boxed::Box::new(cyan.clone())),
        })
        .unwrap();
    assert_eq!(definition.color.as_deref(), Some(&cyan));
    let row = client
        .sketch()
        .set_block_instance_color(&SetBlockInstanceColorArgs {
            sketch_index: 0,
            instance_id: instance,
            color: Some(std::boxed::Box::new(red.clone())),
        })
        .unwrap();
    assert_eq!(row.color.as_deref(), Some(&red));
    assert_eq!(
        client
            .sketch()
            .block_definitions()
            .unwrap()
            .definitions
            .unwrap()[0]
            .color
            .as_deref(),
        Some(&cyan)
    );
    assert_eq!(
        client
            .sketch()
            .block_instances(&SketchArgs::default())
            .unwrap()
            .instances
            .unwrap()[0]
            .color
            .as_deref(),
        Some(&red)
    );
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(instance_color(&files, document, sketch), None);
    files.commands(document).unwrap().redo().unwrap();
    assert_eq!(instance_color(&files, document, sketch), Some(red.clone()));
    files.save(document, "test", &mut source).unwrap();
    let reopened =
        reopen_with_observer(&mut files, document, &mut source, QueryObserver::default());
    assert_eq!(instance_color(&files, reopened, sketch), Some(red));
    assert_eq!(
        files
            .workspace()
            .session(reopened)
            .unwrap()
            .snapshot()
            .sketches
            .block_definitions()[0]
            .color(),
        Some(&cyan)
    );
}
fn instance_color(
    files: &DiscoveryWorkspace,
    document: u64,
    sketch: DocumentSketchId,
) -> Option<Color> {
    files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .get(sketch)
        .unwrap()
        .block_instances()[0]
        .color()
        .cloned()
}

#[test]
fn invalid_block_colors_do_not_modify_history() {
    let (mut files, document, _, mut source) = files_with_circle(Default::default());
    let before = owner_revision(&files, document);
    let request = SetBlockDefinitionColorArgs {
        name: "missing".into(),
        color: Some(std::boxed::Box::new(color(255, 0, 0))),
    };
    assert!(
        Client::new(FileHost::new(&mut files, &mut source, "test"))
            .sketch()
            .set_block_definition_color(&request)
            .is_err()
    );
    assert_eq!(owner_revision(&files, document), before);
}
