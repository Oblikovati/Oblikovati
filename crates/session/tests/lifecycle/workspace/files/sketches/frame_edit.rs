// SPDX-License-Identifier: GPL-2.0-only
use super::recompute::saved_driven_sketch;
use super::*;
use oblikovati_model::document_sketch::{SketchFrameEditDefinition, SketchFrameOrigin};

#[test]
fn authored_frame_survives_save_close_and_reopen() {
    let (mut host, mut source, document, sketch) = saved_driven_sketch();
    host.commands(document)
        .unwrap()
        .run("Edit sketch coordinate system", |snapshot, _| {
            snapshot.sketches.patch_frame_edit(
                sketch,
                SketchFrameEditDefinition {
                    origin: Some(SketchFrameOrigin::Position([2., 3., 0.])),
                    alignment: None,
                },
            )
        })
        .unwrap();
    host.save(document, "alpha", &mut source).unwrap();
    host.close_discarding_changes(document).unwrap();
    let reopened = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    assert_reopened_frame(&host, reopened, sketch);
}

fn assert_reopened_frame(host: &Host, document: u64, sketch: DocumentSketchId) {
    let snapshot = host.workspace().session(document).unwrap().snapshot();
    assert_eq!(
        snapshot
            .resolve_sketch_plane(sketch)
            .unwrap()
            .recipe()
            .origin,
        [2., 3., 0.]
    );
    assert_eq!(
        snapshot
            .sketches
            .get(sketch)
            .unwrap()
            .frame_edit()
            .unwrap()
            .origin,
        Some(SketchFrameOrigin::Position([2., 3., 0.]))
    );
}
