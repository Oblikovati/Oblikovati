// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::native::quick_access_icons;
use oblikovati_gui::GuiShellAction;
use oblikovati_viewport::{ViewportButton, ViewportInput};
impl NativeGui {
    fn step(
        &mut self,
        now: Duration,
        extent: [u32; 2],
        event: Option<&ViewportEvent>,
        commands: &GuiCommandState,
        notice: Option<&str>,
    ) -> Result<HeadGuiUpdate, String> {
        self.step_scaled(now, extent, 1.0, event, commands, notice)
    }
}
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
fn gui_step_for_document(gui: &mut NativeGui, tick: u32, document: u64) -> HeadGuiUpdate {
    let commands = GuiCommandState {
        document: Some(document),
        ..Default::default()
    };
    gui.step(GUI_TICK * tick, [640, 480], None, &commands, None)
        .unwrap()
}
fn click_event(position: [i32; 2], pressed: bool) -> ViewportEvent {
    ViewportEvent::Input(ViewportInput::Button {
        button: ViewportButton::Left,
        pressed,
        position,
        modifiers: Default::default(),
    })
}
#[test]
fn monitor_density_change_rebuilds_the_gui_font_atlas() {
    let mut gui = NativeGui::new(None).unwrap();
    let commands = GuiCommandState::default();
    gui.step_scaled(Duration::ZERO, [1200, 900], 1.0, None, &commands, None)
        .unwrap();
    let original = gui.engine.atlas().clone();
    gui.step_scaled(
        GUI_TICK,
        [1200, 900],
        1.5,
        Some(&ViewportEvent::Resized([1200, 900])),
        &commands,
        None,
    )
    .unwrap();
    assert_ne!(original, *gui.engine.atlas());
    assert!(gui.engine.atlas().rgba.len() > original.rgba.len());
}
#[test]
fn fit_area_requires_a_current_nonempty_layout() {
    let mut gui = NativeGui::new(None).unwrap();
    assert!(gui.document_area([640, 480]).is_err());
    gui_step(&mut gui, 0, None);
    let [left, top, right, bottom] = gui.document_area([640, 480]).unwrap();
    assert!(left < right && top < bottom);
    assert!(gui.document_area([800, 600]).is_err());
    gui.document_area = Some(([640, 480], None));
    assert!(gui.document_area([640, 480]).is_err());
}
#[test]
fn first_click_without_motion_is_captured_and_dispatched_once() {
    let mut gui = NativeGui::new(None).unwrap();
    let commands = GuiCommandState {
        document: Some(7),
        document_type: Some(oblikovati_api::types::DOCUMENT_PART),
        authoring: oblikovati_gui::GuiAuthoringContext::Part,
        documents: vec![oblikovati_api::wire::DocumentInfo {
            id: 7,
            name: "Part1".into(),
            visible: true,
            ..Default::default()
        }],
        ..Default::default()
    };
    let mut step = |tick, event: Option<&ViewportEvent>| {
        gui.step(GUI_TICK * tick, [640, 480], event, &commands, None)
            .unwrap()
    };
    step(0, None);
    let save = quick_access_icons::standard_icon("Save").expect("expected a rendered Save icon");
    assert_eq!(save.alpha, 255);
    let position = save.center.map(|axis| axis.round() as i32);
    let down = step(1, Some(&click_event(position, true)));
    assert!(down.consumed_input);
    let up = step(2, Some(&click_event(position, false)));
    assert!(up.consumed_input);
    let mut actions = down.actions;
    actions.extend(up.actions);
    for tick in 3..8 {
        actions.extend(step(tick, None).actions);
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
        position: [111, 45],
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
fn an_unfocused_mapped_window_can_finish_layout_after_a_document_switch() {
    let mut gui = NativeGui::new(None).unwrap();
    gui.focused = false;
    gui.deadline = Some(GUI_TICK);
    gui.activity.pending_layout = true;
    assert_eq!(gui.wake_after(Duration::ZERO), Some(GUI_TICK));
    gui.activity.pending_layout = false;
    assert_eq!(gui.wake_after(Duration::ZERO), None);
    gui.mapped = false;
    gui.activity.pending_layout = true;
    assert_eq!(gui.wake_after(Duration::ZERO), None);
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
fn changed_content_scale_replaces_the_compiler_atlas_before_publication() {
    let mut gui = NativeGui::new(None).unwrap();
    gui_step(&mut gui, 0, None);
    let previous = gui.textures.clone();
    let commands = GuiCommandState {
        ui_scale: Some(oblikovati_model::ui_scale::UiScaleOptions::new(2., 1.).unwrap()),
        ..Default::default()
    };
    gui.step(GUI_TICK, [1440, 900], None, &commands, None)
        .unwrap();
    assert!(previous != gui.textures);
    assert_eq!(&gui.textures, gui.engine.resources());
    assert!(gui.dirty);
    gui.compiler.compile(gui.draws.as_ref().unwrap()).unwrap();
}
#[test]
fn parameter_metadata_and_results_schedule_frames_without_replaying_commands() {
    let mut gui = NativeGui::new(None).unwrap();
    for tick in 0..8 {
        gui_step_for_document(&mut gui, tick, 7);
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
    let output = gui_step_for_document(&mut gui, 9, 7);
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
    let output = gui_step_for_document(&mut gui, 10, 7);
    assert!(output.parameter_actions.is_empty());
    assert!(output.actions.is_empty());
    assert_eq!(gui.last_frame, Some(GUI_TICK * 10));
}
#[test]
fn parameter_view_issues_request_one_frame_and_valid_metadata_recovers() {
    let mut gui = NativeGui::new(None).unwrap();
    gui.engine.set_parameters_open(true);
    for tick in 0..8 {
        gui_step_for_document(&mut gui, tick, 7);
    }
    assert!(
        gui.report_parameter_issue("parameter 11: expected display-safe text".into())
            .unwrap()
    );
    gui_step_for_document(&mut gui, 9, 7);
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
    gui_step_for_document(&mut gui, 10, 7);
    assert!(!gui.parameter_changed);
}

#[test]
fn idle_sketch_motion_gate_rejects_pointer_and_gui_interactions() {
    let mut gui = NativeGui::new(None).unwrap();
    gui.document_area = Some(([640, 480], Some([100, 100, 600, 400])));
    assert!(gui.allows_ready_sketch_motion([300, 200]));
    assert!(!gui.allows_ready_sketch_motion([99, 200]));
    let down = click_event([111, 45], true);
    gui.pointer.consumed(Some(&down), Default::default());
    assert!(!gui.allows_ready_sketch_motion([300, 200]));
    let up = click_event([111, 45], false);
    gui.pointer.consumed(Some(&up), Default::default());
    gui.last_capture.popup = true;
    assert!(!gui.allows_ready_sketch_motion([300, 200]));
    gui.last_capture.popup = false;
    gui.last_capture.pointer = true;
    assert!(!gui.allows_ready_sketch_motion([300, 200]));
    gui.last_capture.pointer = false;
    gui.last_capture.keyboard = true;
    assert!(!gui.allows_ready_sketch_motion([300, 200]));
    gui.last_capture.keyboard = false;
    gui.activity.pending_animation = true;
    assert!(!gui.allows_ready_sketch_motion([300, 200]));
    gui.activity.pending_animation = false;
    gui.dirty = true;
    assert!(!gui.allows_ready_sketch_motion([300, 200]));
    gui.dirty = false;
    gui.focused = false;
    assert!(!gui.allows_ready_sketch_motion([300, 200]));
}
