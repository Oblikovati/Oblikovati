// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn captured(pointer: bool) -> GuiInputCapture {
    GuiInputCapture {
        pointer,
        keyboard: true,
        text: true,
    }
}
fn press(button: ViewportButton, pressed: bool) -> ViewportEvent {
    ViewportEvent::Input(ViewportInput::Button {
        button,
        pressed,
        position: [30, 45],
        modifiers: Default::default(),
    })
}
#[test]
fn pointer_release_retains_each_press_owner_across_panel_edges() {
    for button in [
        ViewportButton::Left,
        ViewportButton::Middle,
        ViewportButton::Right,
    ] {
        for owner in [false, true] {
            let mut routing = PointerOwnership::default();
            assert_eq!(
                routing.consumed(Some(&press(button, true)), captured(owner)),
                owner
            );
            assert_eq!(
                routing.consumed(Some(&press(button, true)), captured(!owner)),
                owner
            );
            let motion = ViewportEvent::Input(ViewportInput::Motion {
                position: [400, 200],
                modifiers: Default::default(),
            });
            assert_eq!(routing.consumed(Some(&motion), captured(!owner)), owner);
            assert_eq!(
                routing.consumed(Some(&press(button, false)), captured(!owner)),
                owner
            );
            assert_eq!(routing.consumed(Some(&motion), captured(!owner)), !owner);
        }
    }
}
#[test]
fn lifecycle_releases_ownership_and_always_reaches_document() {
    for lifecycle in [
        ViewportEvent::Focus(false),
        ViewportEvent::Mapped(false),
        ViewportEvent::CloseRequested,
    ] {
        let mut routing = PointerOwnership::default();
        assert!(routing.consumed(Some(&press(ViewportButton::Left, true)), captured(true)));
        assert!(!routing.consumed(Some(&lifecycle), captured(true)));
        assert!(!routing.consumed(Some(&press(ViewportButton::Left, false)), captured(false)));
    }
}
#[test]
fn command_chords_and_key_releases_survive_text_focus() {
    let mut routing = PointerOwnership::default();
    for key in [
        ViewportKey::Save,
        ViewportKey::Undo,
        ViewportKey::Redo,
        ViewportKey::Cancel,
        ViewportKey::Pan,
        ViewportKey::Zoom,
        ViewportKey::Orbit,
        ViewportKey::Home,
        ViewportKey::Previous,
    ] {
        for pressed in [false, true] {
            let event = ViewportEvent::Input(ViewportInput::Key { key, pressed });
            assert_eq!(
                routing.consumed(Some(&event), captured(true)),
                pressed && !global_chord(key)
            );
            assert!(copied_gui_events(Some(&event)).is_empty());
            assert!(!routing.consumed(Some(&event), Default::default()));
        }
    }
}
#[test]
fn copied_pointer_events_preserve_coordinates_buttons_and_modifiers() {
    for (source, expected) in [
        (ViewportButton::Left, GuiPointerButton::Left),
        (ViewportButton::Middle, GuiPointerButton::Middle),
        (ViewportButton::Right, GuiPointerButton::Right),
    ] {
        assert_eq!(
            copied_gui_events(Some(&press(source, true))),
            vec![
                GuiInputEvent::PointerMoved([30., 45.]),
                GuiInputEvent::Modifiers(Default::default()),
                GuiInputEvent::PointerButton {
                    button: expected,
                    pressed: true
                },
            ]
        );
    }
    let motion = ViewportEvent::Input(ViewportInput::Motion {
        position: [-2, 80],
        modifiers: ViewportModifiers {
            shift: true,
            control: true,
        },
    });
    assert_eq!(
        copied_gui_events(Some(&motion)),
        vec![
            GuiInputEvent::PointerMoved([-2., 80.]),
            GuiInputEvent::Modifiers(GuiModifiers {
                shift: true,
                control: true,
                ..Default::default()
            })
        ]
    );
}
#[test]
fn wheel_and_focus_are_copied_but_mapping_does_not_invent_focus() {
    let wheel = ViewportEvent::Input(ViewportInput::Wheel {
        position: [2, 3],
        notches: -2,
    });
    assert_eq!(
        copied_gui_events(Some(&wheel)),
        vec![
            GuiInputEvent::PointerMoved([2., 3.]),
            GuiInputEvent::Wheel([0., -2.])
        ]
    );
    let mut routing = PointerOwnership::default();
    assert!(routing.consumed(Some(&wheel), captured(true)));
    for event in [
        ViewportEvent::Mapped(false),
        ViewportEvent::CloseRequested,
        ViewportEvent::Focus(false),
    ] {
        assert_eq!(
            copied_gui_events(Some(&event)),
            vec![GuiInputEvent::Focus(false)]
        );
    }
    assert_eq!(
        copied_gui_events(Some(&ViewportEvent::Focus(true))),
        vec![GuiInputEvent::Focus(true)]
    );
    for event in [
        None,
        Some(&ViewportEvent::Mapped(true)),
        Some(&ViewportEvent::Expose),
    ] {
        assert!(copied_gui_events(event).is_empty());
        assert!(!routing.consumed(event, captured(true)));
    }
}
