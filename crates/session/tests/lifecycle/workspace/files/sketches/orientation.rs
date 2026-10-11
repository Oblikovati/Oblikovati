// SPDX-License-Identifier: GPL-2.0-only
use super::recompute::saved_driven_sketch;
use super::*;
use oblikovati_model::{
    document_sketch::{SketchAxisReference, SketchOrientationDefinition},
    sketch::SketchFrameAxis,
};

fn definition() -> SketchOrientationDefinition {
    SketchOrientationDefinition {
        reference: SketchAxisReference::OriginX,
        sketch_axis: SketchFrameAxis::Y,
        reverse: true,
        origin: Some([7.0, 11.0, 13.0]),
    }
}
fn stored(
    host: &Host,
    document: u64,
    sketch: DocumentSketchId,
) -> Option<SketchOrientationDefinition> {
    host.workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .get(sketch)
        .unwrap()
        .orientation()
}
fn change(host: &mut Host, document: u64, sketch: DocumentSketchId) {
    host.commands(document)
        .unwrap()
        .run("Orient sketch", |snapshot, _| {
            snapshot
                .sketches
                .set_orientation(sketch, Some(definition()))
        })
        .unwrap();
}
#[test]
fn orientation_is_one_history_edit_and_survives_saved_document_reopen() {
    let (mut host, mut source, document, sketch) = saved_driven_sketch();
    change(&mut host, document, sketch);
    assert!(host.workspace().needs_save(document).unwrap());
    host.commands(document).unwrap().undo().unwrap();
    assert_eq!(stored(&host, document, sketch), None);
    assert!(!host.workspace().needs_save(document).unwrap());
    host.commands(document).unwrap().redo().unwrap();
    assert_eq!(stored(&host, document, sketch), Some(definition()));
    host.save(document, "alpha", &mut source).unwrap();
    host.close_discarding_changes(document).unwrap();
    let reopened = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    assert_eq!(stored(&host, reopened, sketch), Some(definition()));
    assert!(!host.workspace().needs_save(reopened).unwrap());
}
#[test]
fn copy_keeps_orientation_recipe_and_independent_source_identity() {
    let (mut host, mut source, document, sketch) = saved_driven_sketch();
    let original = host.workspace().identity(document).unwrap().clone();
    change(&mut host, document, sketch);
    let destination = path("oriented-copy.obk");
    host.save_copy(document, &destination, None, "alpha", &mut source)
        .unwrap();
    let copied = open_file(
        &mut host,
        "oriented-copy.obk",
        &mut source,
        OpenActivation::Background,
    );
    assert_eq!(stored(&host, copied, sketch), Some(definition()));
    assert_eq!(host.workspace().identity(document).unwrap(), &original);
    assert!(host.workspace().needs_save(document).unwrap());
}
