// SPDX-License-Identifier: GPL-2.0-only
//! Ready-state motion routing keeps input boundaries (#4035).
use super::*;

#[test]
fn part_hover_and_camera_drag_batch_only_unseen_gui_input() {
    let motions = [1, 2, 3].map(|x| {
        ViewportEvent::Input(ViewportInput::Motion {
            position: [x, 40],
            modifiers: Default::default(),
        })
    });
    let mut window = FakeGuiWindow {
        permit_idle_motion: true,
        ..Default::default()
    };
    let mut owner = FakeGuiOwner {
        ready_part: true,
        ..Default::default()
    };
    crate::frames::process_frame_inputs(&mut window, &mut owner, &motions).unwrap();
    assert_eq!(window.gui_inputs, motions[2..]);
    assert_eq!(window.inputs, motions[2..]);
    assert_eq!(owner.inputs, motions[2..]);

    window.camera_drag = true;
    window.gui_inputs.clear();
    window.inputs.clear();
    owner.inputs.clear();
    crate::frames::process_frame_inputs(&mut window, &mut owner, &motions).unwrap();
    assert_eq!(window.gui_inputs, motions[2..]);
    assert_eq!(window.inputs, motions);
    assert_eq!(owner.inputs, motions);
}

#[test]
fn ready_motion_bursts_keep_last_position_without_skipping_drags_or_modifier_changes() {
    use oblikovati_viewport::{ViewportButton, ViewportModifiers};
    let motion = |x, shift| {
        ViewportEvent::Input(ViewportInput::Motion {
            position: [x, 40],
            modifiers: ViewportModifiers {
                shift,
                ..Default::default()
            },
        })
    };
    let button = |pressed| {
        ViewportEvent::Input(ViewportInput::Button {
            button: ViewportButton::Left,
            pressed,
            position: [3, 40],
            modifiers: Default::default(),
        })
    };
    let input = [
        motion(1, false),
        motion(2, false),
        button(true),
        motion(3, false),
        motion(4, false),
        button(false),
        motion(5, false),
        motion(6, true),
        motion(7, true),
    ];
    let mut window = FakeGuiWindow {
        permit_idle_motion: true,
        ..Default::default()
    };
    let mut owner = FakeGuiOwner {
        ready_sketch: true,
        ..Default::default()
    };
    crate::frames::process_frame_inputs(&mut window, &mut owner, &input).unwrap();
    assert_eq!(
        owner.inputs,
        [
            motion(2, false),
            button(true),
            motion(3, false),
            motion(4, false),
            button(false),
            motion(5, false),
            motion(7, true)
        ]
    );
    assert_eq!(window.inputs, owner.inputs);
}
#[test]
fn motion_bursts_are_preserved_when_the_sketch_or_window_is_not_ready() {
    let events: Vec<_> = (0..3)
        .map(|x| {
            ViewportEvent::Input(ViewportInput::Motion {
                position: [x, 40],
                modifiers: Default::default(),
            })
        })
        .collect();
    for (ready, permit, transaction_open) in [
        (false, true, false),
        (true, false, false),
        (true, true, true),
    ] {
        let mut window = FakeGuiWindow {
            permit_idle_motion: permit,
            ..Default::default()
        };
        let mut owner = FakeGuiOwner {
            ready_sketch: ready,
            transaction_open,
            ..Default::default()
        };
        crate::frames::process_frame_inputs(&mut window, &mut owner, &events).unwrap();
        assert_eq!(owner.inputs, events);
    }
}
