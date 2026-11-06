// SPDX-License-Identifier: GPL-2.0-only
use super::recompute::saved_driven_sketch;
use super::*;

fn assert_hidden_geometry(
    host: &Host,
    document: u64,
    sketch: DocumentSketchId,
    original: &oblikovati_model::sketch::Sketch2D,
) {
    let snapshot = host.workspace().session(document).unwrap().snapshot();
    let placed = snapshot.sketches.get(sketch).unwrap();
    assert!(!placed.visible());
    assert!(placed.geometry().same_recipe(original));
    snapshot.require_resolved_sketch(sketch).unwrap();
}

#[test]
fn saved_hidden_sketch_reopens_and_copies_without_becoming_a_missing_model_source() {
    let (mut host, mut source, document, sketch) = saved_driven_sketch();
    let original = host
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .get(sketch)
        .unwrap()
        .geometry()
        .clone();
    host.commands(document)
        .unwrap()
        .set_sketch_visibility(sketch, false)
        .unwrap();
    host.save(document, "alpha", &mut source).unwrap();
    host.close_discarding_changes(document).unwrap();
    let reopened = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    assert_hidden_geometry(&host, reopened, sketch, &original);
    host.save_copy(
        reopened,
        &path("visibility-copy.obk"),
        None,
        "alpha",
        &mut source,
    )
    .unwrap();
    let copied = open_file(
        &mut host,
        "visibility-copy.obk",
        &mut source,
        OpenActivation::Background,
    );
    assert_hidden_geometry(&host, copied, sketch, &original);
}
