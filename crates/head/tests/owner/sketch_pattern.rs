// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{history::SketchRectangularPatternRequest, sketch::SketchEntityId};
use std::collections::BTreeSet;
fn input(line: SketchEntityId) -> SketchRectangularPatternRequest {
    SketchRectangularPatternRequest {
        seed: vec![line],
        counts: [3, 2],
        directions: [[1., 0.], [0., 1.]],
        spacing: ["10 mm".into(), "5 mm".into()],
        fitted: false,
        suppressed: BTreeSet::new(),
    }
}
#[test]
fn desktop_pattern_preview_cancel_commit_and_undo_use_one_document_owner() {
    let (mut owner, document, sketch) =
        super::geometric_tools::geometric_owner(&[[0., 0.], [100., 20.]], true);
    let session = owner.documents().workspace().session(document).unwrap();
    let before = session.snapshot().clone();
    let line = before.sketches.get(sketch).unwrap().geometry().lines()[0].id;
    let request = input(line);
    owner
        .preview_sketch_pattern_from_gui(Some(request.clone()))
        .unwrap();
    assert!(owner.sketch_preview().is_some());
    let session = owner.documents().workspace().session(document).unwrap();
    assert!(
        before
            .parameters
            .same_recipe(&session.snapshot().parameters)
    );
    assert!(
        before
            .sketches
            .same_evaluated_state(&session.snapshot().sketches)
    );
    owner.preview_sketch_pattern_from_gui(None).unwrap();
    assert!(owner.sketch_preview().is_none());
    owner.create_sketch_pattern_from_gui(request).unwrap();
    let session = owner.documents().workspace().session(document).unwrap();
    assert_eq!(
        session
            .snapshot()
            .sketches
            .get(sketch)
            .unwrap()
            .geometry()
            .rectangular_patterns()
            .len(),
        1
    );
    assert_eq!(
        owner.gui_command_state().unwrap().sketch_pattern_created,
        Some((1, true))
    );
    owner
        .process_inputs(&[oblikovati_viewport::ViewportEvent::Input(
            oblikovati_viewport::ViewportInput::Key {
                key: oblikovati_viewport::ViewportKey::Undo,
                pressed: true,
            },
        )])
        .unwrap();
    let after = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot();
    assert!(before.parameters.same_recipe(&after.parameters));
    assert!(before.sketches.same_evaluated_state(&after.sketches));
}
#[test]
fn invalid_preview_clears_old_geometry_and_failed_creation_reports_retry() {
    let (mut owner, document, sketch) =
        super::geometric_tools::geometric_owner(&[[0., 0.], [100., 20.]], true);
    let line = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .get(sketch)
        .unwrap()
        .geometry()
        .lines()[0]
        .id;
    let request = input(line);
    owner
        .preview_sketch_pattern_from_gui(Some(request.clone()))
        .unwrap();
    let mut invalid = request;
    invalid.spacing[1] = "unknown".into();
    assert!(
        owner
            .preview_sketch_pattern_from_gui(Some(invalid.clone()))
            .is_err()
    );
    assert!(owner.sketch_preview().is_none());
    assert!(owner.create_sketch_pattern_from_gui(invalid).is_err());
    assert_eq!(
        owner.gui_command_state().unwrap().sketch_pattern_created,
        Some((1, false))
    );
}
