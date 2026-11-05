// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{SketchPlacementTool, placement_test_inputs::*};
use oblikovati_gui::GuiSketchTool;
use oblikovati_model::sketch::SketchPlane;
use oblikovati_viewport::ViewportKey;
const TOOL_ENGINE_CASES: [(GuiSketchTool, SketchPlacementTool); 7] = [
    (GuiSketchTool::Rectangle, SketchPlacementTool::Rectangle),
    (GuiSketchTool::Circle, SketchPlacementTool::Circle),
    (GuiSketchTool::Point, SketchPlacementTool::Point),
    (
        GuiSketchTool::CenterPointArc,
        SketchPlacementTool::CenterPointArc,
    ),
    (GuiSketchTool::Ellipse, SketchPlacementTool::Ellipse),
    (
        GuiSketchTool::CenterRectangle,
        SketchPlacementTool::CenterRectangle,
    ),
    (
        GuiSketchTool::ThreePointRectangle,
        SketchPlacementTool::ThreePointRectangle,
    ),
];
fn prepared_gui_tool_owner() -> TestOwner {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    owner
        .create_sketch_for_edit(document, &sketch_edit::request(SketchPlane::xy()))
        .unwrap();
    owner
}
#[test]
fn gui_tool_selection_uses_the_existing_engine_without_a_history_edit() {
    let mut owner = prepared_gui_tool_owner();
    let before = owner.gui_command_state().unwrap();
    for (choice, expected) in TOOL_ENGINE_CASES {
        assert!(owner.place_sketch_tool_from_gui(choice).unwrap());
        assert_eq!(owner.placement.as_ref().unwrap().tool, expected);
        assert_eq!(owner.gui_command_state().unwrap(), before);
        assert!(owner.placement_preview().is_none());
    }
    owner.finish_sketch_edit().unwrap();
    assert!(!owner.placement_active());
    for (choice, _) in TOOL_ENGINE_CASES {
        assert!(
            owner
                .place_sketch_tool_from_gui(choice)
                .unwrap_err()
                .contains("active sketch edit")
        );
    }
}
#[test]
fn gui_tool_switch_and_cancel_retire_the_preview_without_committing_geometry() {
    let mut owner = prepared_gui_tool_owner();
    let before = owner.gui_command_state().unwrap();
    owner
        .place_sketch_tool_from_gui(GuiSketchTool::Circle)
        .unwrap();
    slot_input::deliver(
        &mut owner,
        &[button([0, 0], true), button([0, 0], false), motion([10, 0])],
    );
    assert!(owner.placement_preview().is_some());
    owner
        .place_sketch_tool_from_gui(GuiSketchTool::Ellipse)
        .unwrap();
    assert!(owner.placement_active() && owner.placement_preview().is_none());
    slot_input::deliver(&mut owner, &[key(ViewportKey::Cancel, true)]);
    assert!(!owner.placement_active() && owner.placement_preview().is_none());
    assert_eq!(owner.gui_command_state().unwrap(), before);
}
