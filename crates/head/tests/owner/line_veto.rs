// SPDX-License-Identifier: GPL-2.0-only
use super::slot_input::PlanarInputWindow;
use super::*;
use crate::placement_test_inputs::button;
use oblikovati_model::sketch::SketchPlane;
use oblikovati_viewport::ViewportEvent;
#[test]
fn a_vetoed_segment_leaves_the_sketch_unchanged_and_the_command_running() {
    let (mut owner, _awake) = super::tool_veto::veto_owner("Create Line");
    owner.create_part_document().unwrap();
    let document = owner.files.workspace().active_document().unwrap();
    let mut request = super::sketch_edit::request(SketchPlane::xy());
    request.project_origin = false;
    let sketch = owner.create_sketch_for_edit(document, &request).unwrap();
    let history = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .history_entries();
    owner
        .place_sketch_tool_from_gui(oblikovati_gui::GuiSketchTool::Line)
        .unwrap();
    owner
        .process_window_inputs(
            &[
                button([0, 0], true),
                button([0, 0], false),
                button([50, 20], true),
            ],
            &PlanarInputWindow::default(),
        )
        .unwrap();
    let session = owner.files.workspace().session(document).unwrap();
    assert!(!session.in_transaction());
    assert_eq!(session.history_entries(), history);
    assert!(
        session
            .snapshot()
            .sketches
            .get(sketch)
            .unwrap()
            .geometry()
            .lines()
            .is_empty()
    );
    assert!(owner.placement_active());
    assert!(matches!(
        owner.take_command_notice(),
        Some(DesktopCommandNotice::PlacementFailed { .. })
    ));
    owner
        .process_window_inputs(
            &[ViewportEvent::Focus(false)],
            &PlanarInputWindow::default(),
        )
        .unwrap();
    assert!(!owner.placement_active());
}
