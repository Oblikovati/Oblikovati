// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{HeadOptions, HeadPresentation, HeadWorkStatus};
use oblikovati_viewport::ViewportKey;
use std::collections::VecDeque;
#[derive(Default)]
struct FakeGuiWindow {
    notices: Vec<crate::DesktopCommandNotice>,
    packets: VecDeque<HeadGuiUpdate>,
    inputs: Vec<ViewportEvent>,
    states: Vec<oblikovati_gui::GuiCommandState>,
    presentations: usize,
    previous_views: usize,
    parameters_open: bool,
    parameter_views: Vec<Option<oblikovati_gui::GuiParameterSnapshot>>,
    parameter_results: Vec<Result<(), String>>,
    constraint_results: Vec<(GuiShellAction, Option<u64>, Result<(), String>)>,
    constraint_preflight_rejected: bool,
    parameter_view_decline: bool,
    parameter_issues: Vec<String>,
    permit_idle_motion: bool,
    pointer_down: bool,
    camera_drag: bool,
    gui_inputs: Vec<ViewportEvent>,
    projection_resize: bool,
}
#[test]
fn physical_command_dispatch_reuses_the_semantic_command_path() {
    let input = ViewportEvent::Input(ViewportInput::PhysicalKey {
        key: oblikovati_viewport::ViewportPhysicalKey::S,
        layout_key: oblikovati_viewport::ViewportPhysicalKey::S,
        command: Some(ViewportKey::Save),
        pressed: true,
        modifiers: Default::default(),
    });
    let expected = ViewportEvent::Input(ViewportInput::Key {
        key: ViewportKey::Save,
        pressed: true,
    });
    assert_eq!(semantic_command_event(input), expected);
    let text = ViewportEvent::Input(ViewportInput::Text('界'));
    assert_eq!(semantic_command_event(text), text);
}
impl HeadWindow for FakeGuiWindow {
    fn synchronize_projection_extent(&mut self) -> Result<bool, String> {
        Ok(std::mem::take(&mut self.projection_resize))
    }
    fn sketch_pixel(
        &self,
        _: oblikovati_model::sketch::SketchPlane,
        position: [f64; 2],
    ) -> Result<[i32; 2], String> {
        if position[0] < 0. {
            return Err(format!(
                "annotation {position:?}: expected a projectable test point"
            ));
        }
        Ok([
            (position[0] * 2.).round() as i32,
            (position[1] * 3.).round() as i32,
        ])
    }

    fn show_command_notice(&mut self, notice: crate::DesktopCommandNotice) -> Result<(), String> {
        self.notices.push(notice);
        Ok(())
    }
    fn gui_parameters_open(&self) -> bool {
        self.parameters_open
    }
    fn set_gui_parameter_snapshot(
        &mut self,
        snapshot: Option<oblikovati_gui::GuiParameterSnapshot>,
    ) -> Result<bool, String> {
        self.parameter_views.push(snapshot);
        if self.parameter_view_decline {
            return Err("parameter 11: expected display-safe text".into());
        }
        Ok(true)
    }
    fn show_gui_parameter_issue(&mut self, issue: String) -> Result<bool, String> {
        let changed = self.parameter_issues.last() != Some(&issue);
        self.parameter_issues.push(issue);
        Ok(changed)
    }
    fn acknowledge_gui_parameter(
        &mut self,
        _: &oblikovati_gui::GuiParameterAction,
        result: Result<(), String>,
    ) -> Result<(), String> {
        self.parameter_results.push(result);
        Ok(())
    }

    fn validate_gui_constraint_settings(&self, _: &GuiShellAction) -> Result<(), String> {
        if self.constraint_preflight_rejected {
            return Err("parent draft changed".into());
        }
        Ok(())
    }
    fn acknowledge_gui_constraint_settings(
        &mut self,
        action: &GuiShellAction,
        document: Option<u64>,
        result: Result<(), String>,
    ) -> Result<(), String> {
        self.constraint_results
            .push((action.clone(), document, result));
        Ok(())
    }

    fn gui_frame(
        &mut self,
        event: Option<&ViewportEvent>,
        state: &oblikovati_gui::GuiCommandState,
    ) -> Result<HeadGuiUpdate, String> {
        if let Some(ViewportEvent::Input(ViewportInput::Button { pressed, .. })) = event {
            self.pointer_down = *pressed;
        }
        self.gui_inputs.extend(event.copied());
        self.states.push(state.clone());
        Ok(self.packets.pop_front().unwrap_or_default())
    }
    fn allows_ready_sketch_motion(&self, _: [i32; 2]) -> bool {
        self.permit_idle_motion && !self.pointer_down
    }
    fn batches_navigation_motion(&self) -> bool {
        self.camera_drag
    }
    fn previous_view(&mut self) -> Result<bool, String> {
        self.previous_views += 1;
        Ok(true)
    }
    fn process_view_input(&mut self, event: &ViewportEvent) -> Result<bool, String> {
        self.inputs.push(*event);
        Ok(true)
    }
    fn events(&mut self, _: usize) -> Result<Vec<ViewportEvent>, String> {
        Ok(Vec::new())
    }
    fn present(&mut self) -> Result<HeadPresentation, String> {
        self.presentations += 1;
        Ok(HeadPresentation::Completed)
    }
    fn wait(&mut self, _: Option<Duration>) -> Result<bool, String> {
        panic!("bounded test must not wait")
    }
    fn finish(&mut self) -> Result<(), String> {
        Ok(())
    }
}
#[test]
fn refused_authoring_command_reports_feedback_without_stopping_the_window() {
    let mut window = FakeGuiWindow::default();
    assert!(report_authoring_result(&mut window, Err("missing sketch".into())).unwrap());
    assert_eq!(window.notices.len(), 1);
    assert!(window.notices[0].message().contains("missing sketch"));
    assert!(!report_authoring_result(&mut window, Ok(false)).unwrap());
}
#[test]
fn unavailable_authoring_commands_remain_local_feedback_and_keep_input_routing_alive() {
    let mut window = FakeGuiWindow::default();
    let mut owner = FakeGuiOwner::default();
    for action in [
        GuiShellAction::StartSketchOn(oblikovati_gui::GuiOriginPlaneKind::Xy),
        GuiShellAction::FinishSketch,
        GuiShellAction::PlaceSketchTool(oblikovati_gui::GuiSketchTool::Rectangle),
        GuiShellAction::OpenDocument(oblikovati_api::wire::OpenDocumentArgs {
            full_document_name: "/missing.opd".into(),
            ..Default::default()
        }),
        GuiShellAction::CloseDocument(oblikovati_api::wire::CloseDocumentArgs {
            id: 1,
            force: false,
        }),
        GuiShellAction::SaveDocumentAs(oblikovati_api::wire::SaveDocumentAsArgs {
            document: 1,
            new_full_document_name: "/draft.opd".into(),
        }),
    ] {
        window.packets.push_back(HeadGuiUpdate {
            actions: vec![action],
            ..Default::default()
        });
        assert!(advance_gui(&mut window, &mut owner, None).unwrap().changed);
    }
    assert_eq!(window.notices.len(), 6);
    assert!(owner.inputs.is_empty());
    assert!(window.inputs.is_empty());
    assert!(!advance_gui(&mut window, &mut owner, None).unwrap().changed);
}
#[derive(Default)]
struct FakeGuiOwner {
    inputs: Vec<ViewportEvent>,
    created_parts: usize,
    annotations: crate::HeadDimensionAnnotations,
    shell_actions: Vec<oblikovati_gui::GuiShellAction>,
    parameter_edits: Vec<oblikovati_gui::GuiParameterAction>,
    accept_cube_actions: bool,
    parameter_failure: bool,
    parameter_view_failure: bool,
    ready_sketch: bool,
    ready_part: bool,
    transaction_open: bool,
    settings_document: Option<u64>,
    settings_rejected: bool,
    settings_calls: usize,
    settings_state_rejected: bool,
}
impl HeadWork for FakeGuiOwner {
    fn create_part_document(&mut self) -> Result<bool, String> {
        self.created_parts += 1;
        self.shell_actions.push(GuiShellAction::NewPart);
        Ok(true)
    }
    fn extrude_from_gui(
        &mut self,
        document: u64,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
        profile: i64,
        distance: &str,
        direction: oblikovati_gui::GuiExtrudeDirection,
        second_distance: Option<&str>,
    ) -> Result<bool, String> {
        self.shell_actions.push(GuiShellAction::Extrude {
            document,
            sketch,
            profile,
            distance: distance.into(),
            second_distance: second_distance.map(str::to_owned),
            direction,
        });
        Ok(true)
    }
    fn gui_parameter_snapshot(
        &self,
    ) -> Result<Option<oblikovati_gui::GuiParameterSnapshot>, String> {
        if self.parameter_view_failure {
            return Err("parameter 11: expected a readable parameter view".into());
        }
        Ok(None)
    }

    fn apply_gui_parameter(
        &mut self,
        action: &oblikovati_gui::GuiParameterAction,
    ) -> Result<bool, String> {
        self.parameter_edits.push(action.clone());
        if self.parameter_failure {
            return Err("formula (: expected a valid expression".into());
        }
        Ok(true)
    }

    fn apply_gui_shell_action(
        &mut self,
        action: oblikovati_gui::GuiShellAction,
    ) -> Result<bool, String> {
        self.shell_actions.push(action);
        Ok(true)
    }

    fn process_inputs(&mut self, events: &[ViewportEvent]) -> Result<bool, String> {
        self.inputs.extend(events);
        Ok(!events.is_empty())
    }
    fn gui_command_state(&mut self) -> Result<oblikovati_gui::GuiCommandState, String> {
        if self.settings_state_rejected {
            return Err("state unavailable".into());
        }
        Ok(oblikovati_gui::GuiCommandState {
            undo_label: Some("Dimension".into()),
            authoring: if self.ready_sketch {
                oblikovati_gui::GuiAuthoringContext::Sketch
            } else if self.ready_part {
                oblikovati_gui::GuiAuthoringContext::Part
            } else {
                Default::default()
            },
            active_sketch: self
                .ready_sketch
                .then_some(oblikovati_model::document_sketch::DocumentSketchId(1)),
            sketch_at_ready: self.ready_sketch,
            transaction_open: self.transaction_open,
            document: self.settings_document.or(self.ready_part.then_some(1)),
            ..Default::default()
        })
    }
    fn poll(&mut self, _: usize) -> Result<HeadWorkStatus, String> {
        Ok(HeadWorkStatus::default())
    }
    fn close_allowed(&mut self) -> Result<bool, String> {
        Ok(true)
    }
    fn finish(&mut self) -> Result<(), String> {
        Ok(())
    }
}
#[test]
fn new_part_gui_action_reaches_the_owner_without_camera_input() {
    let mut window = FakeGuiWindow::default();
    window.packets.push_back(HeadGuiUpdate {
        actions: vec![GuiShellAction::NewPart],
        ..Default::default()
    });
    let mut owner = FakeGuiOwner::default();
    assert!(advance_gui(&mut window, &mut owner, None).unwrap().changed);
    assert_eq!(owner.created_parts, 1);
    assert!(owner.inputs.is_empty());
    assert!(window.inputs.is_empty());
}
#[test]
fn gui_actions_use_existing_owner_and_camera_command_paths() {
    let actions = vec![
        GuiShellAction::Save,
        GuiShellAction::Undo,
        GuiShellAction::Redo,
        GuiShellAction::Home,
        GuiShellAction::ViewCommand(oblikovati_gui::GuiViewCommand::Previous),
    ];
    let mut window = FakeGuiWindow {
        packets: [HeadGuiUpdate {
            actions,
            ..Default::default()
        }]
        .into(),
        ..Default::default()
    };
    let mut owner = FakeGuiOwner::default();
    assert!(advance_gui(&mut window, &mut owner, None).unwrap().changed);
    let expected: Vec<_> = [
        ViewportKey::Save,
        ViewportKey::Undo,
        ViewportKey::Redo,
        ViewportKey::Home,
    ]
    .map(|key| ViewportEvent::Input(ViewportInput::Key { key, pressed: true }))
    .into();
    assert_eq!(window.inputs, expected);
    assert_eq!(owner.inputs, expected);
    assert_eq!(window.previous_views, 1);
    assert_eq!(window.states[0].undo_label.as_deref(), Some("Dimension"));
}
#[test]
fn cube_actions_reach_the_owner_without_viewport_commands() {
    let actions = vec![
        GuiShellAction::NewPart,
        GuiShellAction::StartSketchOn(oblikovati_gui::GuiOriginPlaneKind::Xy),
        GuiShellAction::PlaceSketchTool(oblikovati_gui::GuiSketchTool::Rectangle),
        GuiShellAction::Extrude {
            document: 1,
            sketch: oblikovati_model::document_sketch::DocumentSketchId(1),
            profile: 0,
            distance: "10 mm".into(),
            second_distance: None,
            direction: Default::default(),
        },
    ];
    let mut window = FakeGuiWindow {
        packets: [HeadGuiUpdate {
            actions: actions.clone(),
            ..Default::default()
        }]
        .into(),
        ..Default::default()
    };
    let mut owner = FakeGuiOwner {
        accept_cube_actions: true,
        ..Default::default()
    };
    assert!(advance_gui(&mut window, &mut owner, None).unwrap().changed);
    assert_eq!(owner.shell_actions, actions);
    assert!(window.inputs.is_empty() && owner.inputs.is_empty());
}

#[test]
fn captured_pointer_does_not_reach_document_or_camera() {
    let mut window = FakeGuiWindow {
        packets: [
            HeadGuiUpdate::default(),
            HeadGuiUpdate {
                consumed_input: true,
                ..Default::default()
            },
            HeadGuiUpdate::default(),
        ]
        .into(),
        ..Default::default()
    };
    let mut owner = FakeGuiOwner::default();
    let pointer = ViewportEvent::Input(ViewportInput::Motion {
        position: [111, 45],
        modifiers: Default::default(),
    });
    super::super::process_frame_inputs(
        &mut window,
        &mut owner,
        &[pointer, ViewportEvent::Focus(false)],
    )
    .unwrap();
    assert_eq!(owner.inputs, [ViewportEvent::Focus(false)]);
    assert_eq!(window.inputs, owner.inputs);
}
#[test]
fn deferred_publication_does_not_present_or_replay_actions() {
    let mut window = FakeGuiWindow {
        packets: [
            HeadGuiUpdate {
                deferred: true,
                actions: vec![GuiShellAction::Save],
                wake_after: Some(Duration::ZERO),
                ..Default::default()
            },
            HeadGuiUpdate::default(),
        ]
        .into(),
        ..Default::default()
    };
    let mut owner = FakeGuiOwner::default();
    let summary =
        super::super::run_frames(&mut window, HeadOptions { frames: Some(1) }, &mut owner).unwrap();
    assert_eq!(summary.completed, 1);
    assert_eq!(window.presentations, 1);
    assert_eq!(window.states.len(), 2);
    assert_eq!(owner.inputs.len(), 1);
}
#[test]
fn owner_and_gui_wake_use_the_earliest_deadline() {
    let short = Some(Duration::from_millis(3));
    let long = Some(Duration::from_secs(1));
    for (left, right, expected) in [
        (short, long, short),
        (long, short, short),
        (None, long, long),
        (short, None, short),
        (None, None, None),
    ] {
        assert_eq!(earliest_wake(left, right), expected);
    }
}

#[path = "gui_frames/annotations.rs"]
mod annotations;

impl crate::HeadSketchWork for FakeGuiOwner {
    fn set_document_sketch_settings_from_gui(
        &mut self,
        _: u64,
        _: &oblikovati_api::types::SketchSettings,
        _: &oblikovati_api::types::SketchSettings,
    ) -> Result<bool, String> {
        self.settings_calls += 1;
        if self.settings_rejected {
            return Err("settings rejected".into());
        }
        Ok(false)
    }
    fn start_sketch_on_plane_from_gui(
        &mut self,
        plane: oblikovati_gui::GuiOriginPlaneKind,
    ) -> Result<bool, String> {
        if !self.accept_cube_actions {
            return Err(format!("plane {plane:?}: expected a sketch owner"));
        }
        self.shell_actions
            .push(GuiShellAction::StartSketchOn(plane));
        Ok(true)
    }
    fn place_sketch_tool_from_gui(
        &mut self,
        tool: oblikovati_gui::GuiSketchTool,
    ) -> Result<bool, String> {
        if !self.accept_cube_actions {
            return Err(format!("tool {tool:?}: expected an active sketch"));
        }
        self.shell_actions
            .push(GuiShellAction::PlaceSketchTool(tool));
        Ok(true)
    }
    fn sketch_dimension_annotations(&self) -> Result<crate::HeadDimensionAnnotations, String> {
        Ok(self.annotations.clone())
    }
}

#[path = "gui_frames/parameter.rs"]
mod parameter;

#[path = "gui_frames/sketch_scene_order.rs"]
mod sketch_scene_order;

#[path = "gui_frames/ready_motion.rs"]
mod ready_motion;

#[path = "gui_frames/document_settings.rs"]
mod document_settings;
