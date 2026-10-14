// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_viewport::ViewportPhysicalKey;
fn captured(pointer: bool) -> GuiInputCapture {
    GuiInputCapture {
        pointer,
        keyboard: true,
        text: true,
    }
}
#[test]
fn physical_keys_and_modifiers_reach_the_gui_as_owned_events() {
    let modifiers = ViewportModifiers {
        shift: true,
        control: true,
        alt: true,
        super_key: true,
    };
    let event = physical_gui_key_event(
        ViewportPhysicalKey::Tab,
        ViewportPhysicalKey::Tab,
        modifiers,
    );
    assert_gui_tab_packet(&event, modifiers);
    let mut routing = PointerOwnership::default();
    assert!(routing.consumed(Some(&event), captured(true)));
}
#[test]
fn physical_command_chords_keep_the_existing_text_focus_policy() {
    let mut routing = PointerOwnership::default();
    for (key, command, consumed) in [
        (ViewportPhysicalKey::S, ViewportKey::Save, false),
        (ViewportPhysicalKey::F6, ViewportKey::Home, true),
    ] {
        let event = ViewportEvent::Input(ViewportInput::PhysicalKey {
            key,
            layout_key: key,
            command: Some(command),
            pressed: true,
            modifiers: Default::default(),
        });
        assert_eq!(routing.consumed(Some(&event), captured(true)), consumed);
    }
}
#[test]
fn gui_shortcuts_follow_layout_letters_without_changing_physical_position() {
    let event = physical_gui_key_event(
        ViewportPhysicalKey::Q,
        ViewportPhysicalKey::A,
        ViewportModifiers {
            control: true,
            ..Default::default()
        },
    );
    let copied = copied_gui_events(Some(&event));
    assert!(matches!(
        copied.as_slice(),
        [
            GuiInputEvent::Modifiers(_),
            GuiInputEvent::Key {
                key: ViewportPhysicalKey::A,
                pressed: true
            }
        ]
    ));
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
            ..Default::default()
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

#[test]
fn committed_unicode_is_owned_text_and_obeys_text_capture() {
    let text = ViewportEvent::Input(ViewportInput::Text('🙂'));
    assert_eq!(
        copied_gui_events(Some(&text)),
        [GuiInputEvent::Text("🙂".into())]
    );
    let mut routing = PointerOwnership::default();
    assert!(routing.consumed(Some(&text), captured(true)));
    assert!(!routing.consumed(Some(&text), Default::default()));
}

fn physical_gui_key_event(
    key: ViewportPhysicalKey,
    layout_key: ViewportPhysicalKey,
    modifiers: ViewportModifiers,
) -> ViewportEvent {
    ViewportEvent::Input(ViewportInput::PhysicalKey {
        key,
        layout_key,
        modifiers,
        command: None,
        pressed: true,
    })
}

fn assert_gui_tab_packet(event: &ViewportEvent, modifiers: ViewportModifiers) {
    assert_eq!(
        copied_gui_events(Some(event)),
        [
            GuiInputEvent::Modifiers(modifiers),
            GuiInputEvent::Key {
                key: ViewportPhysicalKey::Tab,
                pressed: true
            }
        ]
    );
}
