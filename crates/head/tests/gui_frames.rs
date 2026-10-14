// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{HeadOptions, HeadPresentation, HeadWorkStatus};
use std::collections::VecDeque;
#[derive(Default)]
struct FakeGuiWindow {
    packets: VecDeque<HeadGuiUpdate>,
    inputs: Vec<ViewportEvent>,
    states: Vec<oblikovati_gui::GuiCommandState>,
    presentations: usize,
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
    fn gui_frame(
        &mut self,
        _: Option<&ViewportEvent>,
        state: &oblikovati_gui::GuiCommandState,
    ) -> Result<HeadGuiUpdate, String> {
        self.states.push(state.clone());
        Ok(self.packets.pop_front().unwrap_or_default())
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
#[derive(Default)]
struct FakeGuiOwner {
    inputs: Vec<ViewportEvent>,
}
impl HeadWork for FakeGuiOwner {
    fn process_inputs(&mut self, events: &[ViewportEvent]) -> Result<bool, String> {
        self.inputs.extend(events);
        Ok(!events.is_empty())
    }
    fn gui_command_state(&self) -> Result<oblikovati_gui::GuiCommandState, String> {
        Ok(oblikovati_gui::GuiCommandState {
            undo_label: Some("Dimension".into()),
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
fn gui_actions_use_existing_owner_and_camera_command_paths() {
    let actions = vec![
        GuiShellAction::Save,
        GuiShellAction::Undo,
        GuiShellAction::Redo,
        GuiShellAction::Home,
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
    assert_eq!(window.states[0].undo_label.as_deref(), Some("Dimension"));
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
        position: [30, 45],
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
            HeadGuiUpdate::default(),
            HeadGuiUpdate {
                deferred: true,
                actions: vec![GuiShellAction::Save],
                wake_after: Some(Duration::ZERO),
                ..Default::default()
            },
        ]
        .into(),
        ..Default::default()
    };
    let mut owner = FakeGuiOwner::default();
    let summary =
        super::super::run_frames(&mut window, HeadOptions { frames: Some(1) }, &mut owner).unwrap();
    assert_eq!(summary.completed, 1);
    assert_eq!(window.presentations, 1);
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
