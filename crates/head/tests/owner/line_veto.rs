// SPDX-License-Identifier: GPL-2.0-only
use super::slot_input::PlanarInputWindow;
use super::*;
use crate::placement_test_inputs::button;
use oblikovati_model::sketch::SketchPlane;
use oblikovati_viewport::{ViewportEvent, ViewportPhysicalKey};
#[test]
fn line_commit_veto_rearms_only_after_rollback_and_focus_loss_releases_the_scope() {
    let (mut owner, _awake) = super::tool_veto::veto_owner("Line");
    owner.create_part_document().unwrap();
    let document = owner.files.workspace().active_document().unwrap();
    let mut request = super::sketch_edit::request(SketchPlane::xy());
    request.project_origin = false;
    let sketch = owner.create_sketch_for_edit(document, &request).unwrap();
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
        .place_sketch_tool_from_gui(oblikovati_gui::GuiSketchTool::Line)
        .unwrap();
    let token = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .transaction_identity();
    owner
        .process_window_inputs(
            &[
                button([0, 0], true),
                button([50, 20], true),
                super::line_tools::line_key(ViewportPhysicalKey::Enter),
            ],
            &PlanarInputWindow::default(),
        )
        .unwrap();
    let session = owner.files.workspace().session(document).unwrap();
    assert!(session.in_transaction());
    assert_ne!(session.transaction_identity(), token);
    assert!(session.snapshot().sketches.same_recipe(&before));
    assert_eq!(session.history_entries(), history);
    assert!(owner.placement_active());
    assert!(!owner.check_sketch_tool_context());
    assert!(
        matches!(owner.take_command_notice(), Some(DesktopCommandNotice::PlacementFailed { cause }) if cause.contains("unlocked"))
    );
    owner
        .process_window_inputs(
            &[ViewportEvent::Focus(false)],
            &PlanarInputWindow::default(),
        )
        .unwrap();
    assert!(
        !owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .in_transaction()
    );
    assert!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .snapshot()
            .sketches
            .get(sketch)
            .unwrap()
            .geometry()
            .lines()
            .is_empty()
    );
}
