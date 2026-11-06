// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::placement_test_inputs::button;
use oblikovati_model::sketch::{SketchPlacementRecipe, SketchPlane};

#[test]
fn a_fix_commit_veto_restores_both_endpoints_and_keeps_the_tool() {
    let (mut owner, _awake) = super::tool_veto::veto_owner("Fix");
    owner.create_part_document().unwrap();
    let document = owner.files.workspace().active_document().unwrap();
    let mut request = super::sketch_edit::request(SketchPlane::xy());
    request.project_origin = false;
    let sketch = owner.create_sketch_for_edit(document, &request).unwrap();
    let line = SketchPlacementRecipe::line_chain(&[[1., 2.], [101., 42.]], false).unwrap();
    owner
        .commit_sketch_placement(document, sketch, &line, &[])
        .unwrap();
    let before = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .clone();
    let history = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .history_entries();
    owner
        .constrain_sketch_from_gui(oblikovati_gui::GuiSketchConstraint::Fix)
        .unwrap();
    owner
        .process_window_inputs(
            &[button([51, 22], true)],
            &super::slot_input::PlanarInputWindow::default(),
        )
        .unwrap();
    let session = owner.files.workspace().session(document).unwrap();
    assert!(!session.in_transaction());
    assert!(session.snapshot().sketches.same_recipe(&before));
    assert_eq!(session.history_entries(), history);
    assert!(owner.sketch_tool.is_some());
    assert!(!owner.check_sketch_tool_context());
    assert!(
        matches!(owner.take_command_notice(), Some(DesktopCommandNotice::AuthoringFailed { cause }) if cause.contains("unlocked"))
    );
}
