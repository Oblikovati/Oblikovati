// SPDX-License-Identifier: GPL-2.0-only
use super::slot_input::{PlanarInputWindow, deliver};
use super::*;
use crate::{SketchPlacementTool, placement_test_inputs::*};
use oblikovati_viewport::{ViewportButton, ViewportEvent, ViewportInput, ViewportKey};
fn rectangle_owner() -> TestOwner {
    let (mut owner, _, _) = super::rectangle_input::placement_owner(SketchPlacementTool::Rectangle);
    deliver(
        &mut owner,
        &[
            button([0, 0], true),
            button([0, 0], false),
            motion([100, 100]),
            button([100, 100], true),
            key(ViewportKey::Cancel, true),
        ],
    );
    owner
}
fn modified_click(pixel: [i32; 2]) -> ViewportEvent {
    ViewportEvent::Input(ViewportInput::Button {
        button: ViewportButton::Left,
        pressed: true,
        position: pixel,
        modifiers: oblikovati_viewport::ViewportModifiers {
            control: true,
            ..Default::default()
        },
    })
}
fn selection(owner: &mut TestOwner) -> SelectionResult {
    Client::new(owner.document_host())
        .model()
        .selection()
        .unwrap()
}
#[test]
fn viewport_clicks_share_checked_entities_with_the_api_and_scene_without_history() {
    let mut owner = rectangle_owner();
    let history = owner.gui_command_state().unwrap().undo_label;
    deliver(&mut owner, &[button([50, 0], true)]);
    assert_eq!(selection(&mut owner).kinds, Some(vec![6]));
    let first = owner.gui_command_state().unwrap().sketch_selection;
    assert_eq!(first.entities.len(), 1);
    assert!(first.sketches.is_empty());
    deliver(&mut owner, &[modified_click([100, 50])]);
    assert_eq!(selection(&mut owner).count, 2);
    assert_eq!(
        owner
            .gui_command_state()
            .unwrap()
            .sketch_selection
            .entities
            .len(),
        2
    );
    deliver(&mut owner, &[modified_click([100, 50])]);
    assert_eq!(owner.gui_command_state().unwrap().sketch_selection, first);
    deliver(&mut owner, &[modified_click([200, 200])]);
    assert_eq!(selection(&mut owner).count, 1);
    deliver(&mut owner, &[button([200, 200], true)]);
    assert_eq!(selection(&mut owner).count, 0);
    assert_eq!(owner.gui_command_state().unwrap().undo_label, history);
}
#[test]
fn endpoint_point_priority_projection_failures_and_generation_guards_are_visible() {
    let mut owner = rectangle_owner();
    deliver(&mut owner, &[button([0, 0], true)]);
    let chosen = owner.gui_command_state().unwrap().sketch_selection.entities[0];
    let document = owner.documents().workspace().active_document().unwrap();
    owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .get(chosen.0)
        .unwrap()
        .geometry()
        .point(chosen.1)
        .unwrap();
    assert!(
        owner
            .process_window_inputs(
                &[button([50, 0], true)],
                &PlanarInputWindow {
                    navigating: false,
                    fail: true,
                    edge_on: false
                }
            )
            .unwrap()
    );
    assert!(
        matches!(owner.take_command_notice(), Some(DesktopCommandNotice::SelectionFailed { cause }) if cause.contains("named projection failure"))
    );
    assert_eq!(selection(&mut owner).count, 1);
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert_eq!(selection(&mut owner).count, 0);
}
#[test]
fn active_placement_and_releases_do_not_select_geometry() {
    let mut owner = rectangle_owner();
    deliver(&mut owner, &[button([50, 0], false)]);
    assert_eq!(selection(&mut owner).count, 0);
    owner
        .place_sketch_tool_from_gui(oblikovati_gui::GuiSketchTool::Rectangle)
        .unwrap();
    deliver(&mut owner, &[button([50, 0], true)]);
    assert_eq!(selection(&mut owner).count, 0);
}

#[test]
fn camera_gestures_do_not_consume_or_clear_editor_selection() {
    let mut owner = rectangle_owner();
    deliver(&mut owner, &[button([50, 0], true)]);
    let selected = owner.gui_command_state().unwrap().sketch_selection;
    owner
        .process_window_inputs(
            &[button([200, 200], true)],
            &PlanarInputWindow {
                navigating: true,
                fail: true,
                edge_on: false,
            },
        )
        .unwrap();
    assert_eq!(
        owner.gui_command_state().unwrap().sketch_selection,
        selected
    );
    assert_eq!(selection(&mut owner).count, 1);
}

#[test]
fn successful_pick_replaces_a_failure_after_the_window_consumes_it() {
    let mut owner = rectangle_owner();
    owner
        .process_window_inputs(
            &[button([50, 0], true)],
            &PlanarInputWindow {
                fail: true,
                ..Default::default()
            },
        )
        .unwrap();
    assert!(matches!(
        owner.take_command_notice(),
        Some(DesktopCommandNotice::SelectionFailed { .. })
    ));
    deliver(&mut owner, &[motion([50, 0])]);
    assert!(owner.take_command_notice().is_none());
    deliver(&mut owner, &[button([50, 0], true)]);
    assert_eq!(
        owner.take_command_notice(),
        Some(DesktopCommandNotice::SelectionChanged { count: 1 })
    );
    deliver(&mut owner, &[button([200, 200], true)]);
    assert_eq!(
        owner.take_command_notice().unwrap().message(),
        "Sketch selection count: 0."
    );
    assert_eq!(selection(&mut owner).count, 0);
}
