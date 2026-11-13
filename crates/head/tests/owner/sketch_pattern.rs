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

#[test]
fn recorded_pattern_is_exposed_to_browser_and_edits_keep_its_identity() {
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
    let mut request = input(line);
    owner
        .create_sketch_pattern_from_gui(request.clone())
        .unwrap();
    let state = owner.gui_command_state().unwrap();
    assert_eq!(state.active_sketch, Some(sketch));
    let pattern_id = state.browser.as_ref().unwrap().rectangular_patterns[0]
        .pattern
        .id;
    let action = oblikovati_gui::GuiShellAction::OpenSketchPattern {
        sketch,
        pattern: pattern_id,
    };
    assert!(action.enabled(&state));
    let mut inactive = state.clone();
    inactive.active_sketch = None;
    assert!(!action.enabled(&inactive));
    inactive = state.clone();
    inactive.transaction_open = true;
    assert!(!action.enabled(&inactive));
    let browser = state.browser.unwrap();
    assert_eq!(browser.rectangular_patterns.len(), 1);
    let row = &browser.rectangular_patterns[0];
    assert_eq!(row.sketch, sketch);
    assert_eq!(row.spacing.as_ref().unwrap(), &["10 mm", "5 mm"]);
    let before = row.pattern.clone();
    request.counts[0] = 4;
    request.spacing[0] = "20 mm".into();
    owner
        .preview_sketch_pattern_edit_from_gui(before.id, request.clone())
        .unwrap();
    assert!(owner.sketch_preview().is_some());
    owner.preview_sketch_pattern_from_gui(None).unwrap();
    assert_eq!(
        owner
            .gui_command_state()
            .unwrap()
            .browser
            .unwrap()
            .rectangular_patterns[0]
            .pattern,
        before
    );
    owner
        .edit_sketch_pattern_from_gui(before.id, request)
        .unwrap();
    let row = &owner
        .gui_command_state()
        .unwrap()
        .browser
        .unwrap()
        .rectangular_patterns[0];
    assert_eq!(row.pattern.id, before.id);
    assert_eq!(row.pattern.definition.counts, [4, 2]);
    assert_eq!(row.spacing.as_ref().unwrap(), &["20 mm", "5 mm"]);
    for member in &before.members {
        assert_eq!(
            row.pattern
                .members
                .iter()
                .find(|item| item.index == member.index)
                .unwrap(),
            member
        );
    }
}
