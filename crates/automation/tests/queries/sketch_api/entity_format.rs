// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::{Color, OVERRIDE_COLOR_SOURCE, SKETCH_LINE_DASHED, SketchEntityFormat};

fn red() -> SketchEntityFormat {
    SketchEntityFormat {
        line_type: SKETCH_LINE_DASHED,
        override_color: Color {
            r: 255,
            g: 0,
            b: 0,
            opacity: 1.0,
            source: OVERRIDE_COLOR_SOURCE,
        },
        line_weight: 0.5,
    }
}

#[test]
fn typed_format_edit_undo_redo_and_reopen_preserve_format_and_solve() {
    let (mut files, document, sketch, mut source) = files_with_circle(Default::default());
    let edit = Some(super::inspection::inspection_edit(
        &mut files, document, sketch,
    ));
    let before = verification(&files, document, sketch);
    let request = SetSketchEntityFormatArgs {
        entity_id: 2,
        format: red(),
    };
    let reply =
        Client::new(FileHost::new(&mut files, &mut source, "test").with_sketch_edit(edit.clone()))
            .sketch()
            .set_entity_format(&request)
            .unwrap();
    assert!(reply.has_format);
    assert_eq!(reply.format, red());
    let read = Client::new(FileQueryHost::new(&files).with_sketch_edit(edit))
        .sketch()
        .entity_format(&SketchEntityFormatArgs { entity_id: 2 })
        .unwrap();
    assert_eq!(read, reply);
    assert_eq!(verification(&files, document, sketch), before);
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(stored_format(&files, document, sketch), None);
    files.commands(document).unwrap().redo().unwrap();
    assert_eq!(stored_format(&files, document, sketch), Some(red()));
    files.save(document, "test", &mut source).unwrap();
    let reopened =
        reopen_with_observer(&mut files, document, &mut source, QueryObserver::default());
    assert_eq!(stored_format(&files, reopened, sketch), Some(red()));
}

fn verification(
    files: &DiscoveryWorkspace,
    document: u64,
    sketch: DocumentSketchId,
) -> SketchEvaluation {
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    match snapshot
        .sketches
        .get(sketch)
        .unwrap()
        .health(&snapshot.parameters)
    {
        PlacedSketchHealth::Current(evaluation) => evaluation.clone(),
        other => panic!("expected current solve evidence, got {other:?}"),
    }
}

fn stored_format(
    files: &DiscoveryWorkspace,
    document: u64,
    sketch: DocumentSketchId,
) -> Option<SketchEntityFormat> {
    files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .get(sketch)
        .unwrap()
        .geometry()
        .entity_format(SketchEntityId(2))
        .unwrap()
        .cloned()
}

#[test]
fn format_requires_active_sketch_and_rejects_unknown_entities() {
    let (mut files, document, sketch, mut source) = files_with_circle(Default::default());
    let missing = SetSketchEntityFormatArgs {
        entity_id: 999,
        format: red(),
    };
    let before = owner_revision(&files, document);
    assert!(
        Client::new(FileHost::new(&mut files, &mut source, "test"))
            .sketch()
            .set_entity_format(&missing)
            .is_err()
    );
    let edit = Some(super::inspection::inspection_edit(
        &mut files, document, sketch,
    ));
    assert!(
        Client::new(FileHost::new(&mut files, &mut source, "test").with_sketch_edit(edit))
            .sketch()
            .set_entity_format(&missing)
            .is_err()
    );
    assert_eq!(owner_revision(&files, document), before);
}

#[test]
fn typed_default_clears_override_and_query_reports_inheritance() {
    let (mut files, document, sketch, mut source) = files_with_circle(Default::default());
    let edit = Some(super::inspection::inspection_edit(
        &mut files, document, sketch,
    ));
    let client = Client::new(FileHost::new(&mut files, &mut source, "test").with_sketch_edit(edit));
    let mut request = SetSketchEntityFormatArgs {
        entity_id: 2,
        format: red(),
    };
    client.sketch().set_entity_format(&request).unwrap();
    request.format = SketchEntityFormat::default();
    let cleared = client.sketch().set_entity_format(&request).unwrap();
    assert!(!cleared.has_format);
    assert_eq!(cleared.format, SketchEntityFormat::default());
    let read = client
        .sketch()
        .entity_format(&SketchEntityFormatArgs { entity_id: 2 })
        .unwrap();
    assert_eq!(read, cleared);
}
