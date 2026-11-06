// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{SketchPlacementTool, placement_test_inputs::*};
use oblikovati_gui::GuiSketchTool;
use oblikovati_model::sketch::{CircumcircleOptions, SketchPlane};
use oblikovati_viewport::ViewportKey;
const TOOL_ENGINE_CASES: [(GuiSketchTool, SketchPlacementTool); 9] = [
    (
        GuiSketchTool::ThreePointCircle,
        SketchPlacementTool::ThreePointCircle {
            options: CircumcircleOptions {
                maximum_point_residual: 1e-9,
            },
        },
    ),
    (
        GuiSketchTool::ThreePointArc,
        SketchPlacementTool::ThreePointArc {
            options: CircumcircleOptions {
                maximum_point_residual: 1e-9,
            },
        },
    ),
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
        assert_eq!(owner.placement_state().unwrap().tool, expected);
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

fn selected_circular_options(choice: GuiSketchTool, scale: f64) -> CircumcircleOptions {
    use super::super::sketch_tools_gui::gui_placement_tool;
    let resolved = gui_placement_tool(choice, scale).unwrap();
    match resolved {
        SketchPlacementTool::ThreePointCircle { options }
        | SketchPlacementTool::ThreePointArc { options } => options,
        _ => panic!("{resolved:?}: expected a three-point circular tool"),
    }
}
#[test]
fn circular_gui_choices_use_the_versioned_physical_policy_in_working_units() {
    use oblikovati_model::sketch::SketchConstructionPolicy;
    for scale in [0.1, 1., 2.54] {
        let expected = SketchConstructionPolicy::default()
            .in_working_units(scale)
            .unwrap();
        for choice in [
            GuiSketchTool::ThreePointCircle,
            GuiSketchTool::ThreePointArc,
        ] {
            assert_eq!(selected_circular_options(choice, scale), expected);
        }
    }
}
#[test]
fn circular_gui_choices_refuse_invalid_working_unit_scales() {
    use super::super::sketch_tools_gui::gui_placement_tool;
    for choice in [
        GuiSketchTool::ThreePointCircle,
        GuiSketchTool::ThreePointArc,
    ] {
        assert!(
            gui_placement_tool(choice, 0.)
                .unwrap_err()
                .contains("scale 0")
        );
    }
}
