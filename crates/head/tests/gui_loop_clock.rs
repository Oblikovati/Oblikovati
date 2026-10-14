// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_gui::GuiShellAction;
use oblikovati_viewport::{ViewportButton, ViewportInput};
fn gui_step(gui: &mut NativeGui, tick: u32, event: Option<&ViewportEvent>) -> HeadGuiUpdate {
    gui.step(
        GUI_TICK * tick,
        [640, 480],
        event,
        &GuiCommandState::default(),
        None,
    )
    .unwrap()
}
fn click_event(pressed: bool) -> ViewportEvent {
    ViewportEvent::Input(ViewportInput::Button {
        button: ViewportButton::Left,
        pressed,
        position: [30, 45],
        modifiers: Default::default(),
    })
}
#[test]
fn first_click_without_motion_is_captured_and_dispatched_once() {
    let mut gui = NativeGui::new(None).unwrap();
    gui_step(&mut gui, 0, None);
    let down = gui_step(&mut gui, 1, Some(&click_event(true)));
    assert!(down.consumed_input);
    let up = gui_step(&mut gui, 2, Some(&click_event(false)));
    assert!(up.consumed_input);
    let mut actions = down.actions;
    actions.extend(up.actions);
    for tick in 3..8 {
        actions.extend(gui_step(&mut gui, tick, None).actions);
    }
    assert_eq!(actions, [GuiShellAction::Save]);
}
#[test]
fn idle_frame_settles_and_metadata_or_extent_requests_a_new_packet() {
    let mut gui = NativeGui::new(None).unwrap();
    for tick in 0..8 {
        gui_step(&mut gui, tick, None);
    }
    assert_eq!(gui.wake_after(GUI_TICK * 8), None);
    gui.dirty = false;
    let before = gui.last_frame;
    gui_step(&mut gui, 9, None);
    assert!(!gui.dirty);
    assert_eq!(gui.last_frame, before);
    gui.step(
        GUI_TICK * 10,
        [800, 600],
        None,
        &GuiCommandState::default(),
        Some("Saved part"),
    )
    .unwrap();
    assert!(gui.dirty);
    assert_eq!(gui.draws.as_ref().unwrap().extent, [800, 600]);
    assert_eq!(gui.notice.as_deref(), Some("Saved part"));
    gui.dirty = false;
    gui.require_publication();
    assert!(gui.dirty);
}
#[test]
fn hover_uses_a_deadline_and_focus_or_unmap_suspends_it() {
    let mut gui = NativeGui::new(None).unwrap();
    gui_step(&mut gui, 0, None);
    let motion = ViewportEvent::Input(ViewportInput::Motion {
        position: [30, 45],
        modifiers: Default::default(),
    });
    assert_eq!(
        gui_step(&mut gui, 1, Some(&motion)).wake_after,
        Some(GUI_TICK)
    );
    assert_eq!(
        gui.wake_after(GUI_TICK + GUI_TICK / 2),
        Some(GUI_TICK - GUI_TICK / 2)
    );
    assert_eq!(
        gui_step(&mut gui, 2, Some(&ViewportEvent::Focus(false))).wake_after,
        None
    );
    gui_step(&mut gui, 3, Some(&ViewportEvent::Mapped(false)));
    gui_step(&mut gui, 4, Some(&ViewportEvent::Mapped(true)));
    assert_eq!(gui.wake_after(GUI_TICK * 4), None);
    gui_step(&mut gui, 5, Some(&ViewportEvent::Focus(true)));
    gui_step(&mut gui, 6, Some(&motion));
    assert_eq!(gui.wake_after(GUI_TICK * 6), Some(GUI_TICK));
}
#[test]
fn repeated_timestamp_is_valid_but_reversed_time_is_refused() {
    let mut gui = NativeGui::new(None).unwrap();
    gui_step(&mut gui, 2, None);
    gui_step(&mut gui, 2, Some(&ViewportEvent::Expose));
    let error = gui
        .step(
            GUI_TICK,
            [640, 480],
            Some(&ViewportEvent::Expose),
            &GuiCommandState::default(),
            None,
        )
        .unwrap_err();
    assert!(error.contains("monotonic"), "{error}");
    assert_eq!(gui.last_frame, Some(GUI_TICK * 2));
}
#[test]
fn parameter_metadata_and_results_schedule_frames_without_replaying_commands() {
    let mut gui = NativeGui::new(None).unwrap();
    for tick in 0..8 {
        gui_step(&mut gui, tick, None);
    }
    assert!(!gui.parameters_open());
    gui.engine.set_parameters_open(true);
    assert!(gui.parameters_open());
    let snapshot = oblikovati_gui::GuiParameterSnapshot {
        linked_tables: Vec::new(),
        link_sources: Vec::new(),
        groups: Vec::new(),
        document: 7,
        rows: Vec::new(),
    };
    assert!(gui.set_parameters(Some(snapshot.clone())).unwrap());
    assert!(!gui.set_parameters(Some(snapshot)).unwrap());
    let output = gui_step(&mut gui, 9, None);
    assert!(output.actions.is_empty());
    assert_eq!(gui.last_frame, Some(GUI_TICK * 9));
    assert!(!gui.parameter_changed);
    let action = oblikovati_gui::GuiParameterAction::Toggle {
        document: 7,
        parameter: 11,
        field: oblikovati_gui::GuiParameterToggleField::Key,
        value: true,
    };
    gui.acknowledge_parameter(
        &action,
        Err("parameter 11: expected an editable row".into()),
    );
    assert_eq!(gui.wake_after(GUI_TICK * 10), Some(Duration::ZERO));
    let output = gui_step(&mut gui, 10, None);
    assert!(output.parameter_actions.is_empty());
    assert!(output.actions.is_empty());
    assert_eq!(gui.last_frame, Some(GUI_TICK * 10));
}
#[test]
fn parameter_view_issues_request_one_frame_and_valid_metadata_recovers() {
    let mut gui = NativeGui::new(None).unwrap();
    gui.engine.set_parameters_open(true);
    for tick in 0..8 {
        gui_step(&mut gui, tick, None);
    }
    assert!(
        gui.report_parameter_issue("parameter 11: expected display-safe text".into())
            .unwrap()
    );
    gui_step(&mut gui, 9, None);
    assert!(!gui.parameter_changed);
    assert!(
        !gui.report_parameter_issue("parameter 11: expected display-safe text".into())
            .unwrap()
    );
    let view = oblikovati_gui::GuiParameterSnapshot {
        linked_tables: Vec::new(),
        link_sources: Vec::new(),
        groups: Vec::new(),
        document: 7,
        rows: Vec::new(),
    };
    assert!(gui.set_parameters(Some(view)).unwrap());
    assert!(gui.parameter_changed);
    gui_step(&mut gui, 10, None);
    assert!(!gui.parameter_changed);
}
