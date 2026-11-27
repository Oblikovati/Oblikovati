// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use super::{
    geometric_tools::{geometric_owner, inspection},
    slot_input::deliver,
};
use crate::placement_test_inputs::*;
use oblikovati_gui::GuiSketchTool;
use oblikovati_model::document_sketch::DocumentSketchId;

#[test]
fn browser_visibility_updates_api_scene_picking_and_history() {
    let (mut owner, document, sketch) = geometric_owner(&[[0., 0.], [100., 40.]], false);
    assert_eq!(
        owner.take_sketch_scene().unwrap().unwrap().sketches.len(),
        1
    );
    deliver(&mut owner, &[button([0, 0], true)]);
    assert_eq!(
        owner
            .gui_command_state()
            .unwrap()
            .sketch_selection
            .entities
            .len(),
        1
    );
    assert!(
        owner
            .set_sketch_visibility_from_gui(document, sketch, false)
            .unwrap()
    );
    assert!(!inspection(&mut owner).visible);
    assert!(!owner.gui_browser_snapshot(document).unwrap().sketches[0].visible);
    assert!(
        owner
            .take_sketch_scene()
            .unwrap()
            .unwrap()
            .sketches
            .is_empty()
    );
    deliver(&mut owner, &[button([0, 0], true)]);
    assert!(
        owner
            .gui_command_state()
            .unwrap()
            .sketch_selection
            .entities
            .is_empty()
    );
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert!(inspection(&mut owner).visible);
    assert_eq!(
        owner.take_sketch_scene().unwrap().unwrap().sketches.len(),
        1
    );
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    assert!(!inspection(&mut owner).visible);
    assert!(
        owner
            .take_sketch_scene()
            .unwrap()
            .unwrap()
            .sketches
            .is_empty()
    );
}

#[test]
fn visibility_noop_and_invalid_identity_preserve_the_live_tool() {
    let (mut owner, document, sketch) = geometric_owner(&[[0., 0.]], false);
    owner
        .place_sketch_tool_from_gui(GuiSketchTool::Point)
        .unwrap();
    assert!(owner.sketch_tool.is_some());
    assert!(
        !owner
            .set_sketch_visibility_from_gui(document, sketch, true)
            .unwrap()
    );
    assert!(
        owner
            .set_sketch_visibility_from_gui(document + 1, sketch, false)
            .is_err()
    );
    assert!(
        owner
            .set_sketch_visibility_from_gui(document, DocumentSketchId(999), false)
            .is_err()
    );
    assert!(owner.sketch_tool.is_some());
    assert!(
        owner
            .set_sketch_visibility_from_gui(document, sketch, false)
            .unwrap()
    );
    assert!(owner.sketch_tool.is_none());
}
