// SPDX-License-Identifier: GPL-2.0-only
use super::super::private_server::PrivateX11Server;
use super::super::{NativeWindow, NativeX11};
use super::*;
struct NativeInputFixture {
    window: NativeWindow,
}
impl NativeInputFixture {
    fn new() -> Self {
        Self {
            window: NativeWindow::create(
                NativeX11::open().unwrap(),
                "Oblikovati input acceptance",
                [64, 48],
            )
            .unwrap(),
        }
    }
    fn on_display(name: &str) -> Self {
        let x11 = NativeX11::open().unwrap();
        let display = Handle::new(
            AspectDisplayConnection::with_display_name(
                &rustycad_tkernel::tcollection::AsciiString::from(name),
            )
            .unwrap(),
        );
        Self {
            window: NativeWindow::create_with_display(
                x11,
                display,
                "Oblikovati compose acceptance",
                [64, 48],
            )
            .unwrap(),
        }
    }
    fn event(&self, kind: i32) -> xlib::XEvent {
        let mut event = xlib::XEvent { pad: [0; 24] };
        event.any.type_ = kind;
        event.any.display = self.window.pointer();
        event.any.window = self.window.window.native_handle();
        event
    }
    fn key(&self, symbol: u32, pressed: bool) -> xlib::XEvent {
        let mut event = self.event(if pressed {
            xlib::KeyPress
        } else {
            xlib::KeyRelease
        });
        event.key.keycode = u32::from(unsafe {
            (self.window.x11.api.XKeysymToKeycode)(self.window.pointer(), u64::from(symbol))
        });
        event
    }
    fn send(&self, mut event: xlib::XEvent, mask: i64) {
        assert_ne!(
            unsafe {
                (self.window.x11.api.XSendEvent)(
                    self.window.pointer(),
                    self.window.window.native_handle(),
                    0,
                    mask,
                    &mut event,
                )
            },
            0
        );
        self.window.sync();
    }
}
#[test]
fn mouse_buttons_copy_positions_and_modifiers_and_ignore_auxiliary_releases() {
    let fixture = NativeInputFixture::new();
    for (number, button) in [
        (1, ViewportButton::Left),
        (2, ViewportButton::Middle),
        (3, ViewportButton::Right),
    ] {
        for pressed in [true, false] {
            check_native_button(&fixture, number, button, pressed);
        }
    }
    let mut event = fixture.event(xlib::ButtonRelease);
    for button in [4, 5, 9] {
        event.button.button = button;
        assert!(translate_native_pointer(fixture.window.pointer(), event).is_none());
    }
}
fn check_native_button(
    fixture: &NativeInputFixture,
    number: u32,
    button: ViewportButton,
    pressed: bool,
) {
    let mut event = fixture.event(if pressed {
        xlib::ButtonPress
    } else {
        xlib::ButtonRelease
    });
    event.button.button = number;
    event.button.x = -10;
    event.button.y = 90;
    event.button.state = xlib::ShiftMask | xlib::ControlMask;
    let expected = expected_button_input(button, pressed);
    assert_eq!(
        fixture.window.translate(event),
        Some(ViewportEvent::Input(expected))
    );
}
fn expected_button_input(button: ViewportButton, pressed: bool) -> ViewportInput {
    ViewportInput::Button {
        button,
        pressed,
        position: [-10, 90],
        modifiers: ViewportModifiers {
            shift: true,
            control: true,
            ..Default::default()
        },
    }
}
#[test]
fn wheel_and_motion_events_are_copied_through_the_actual_native_queue() {
    let fixture = NativeInputFixture::new();
    fixture.window.poll(64).unwrap();
    for (button, notches) in [(4, 1), (5, -1)] {
        let mut event = fixture.event(xlib::ButtonPress);
        event.button.button = button;
        fixture.send(event, xlib::ButtonPressMask);
        let expected = ViewportEvent::Input(ViewportInput::Wheel {
            notches,
            position: [0, 0],
        });
        assert!(fixture.window.poll(64).unwrap().contains(&expected));
    }
    check_native_motion(&fixture);
}
fn check_native_motion(fixture: &NativeInputFixture) {
    let mut event = fixture.event(xlib::MotionNotify);
    event.motion.x = 20;
    event.motion.y = 30;
    event.motion.state = xlib::ShiftMask;
    fixture.send(event, xlib::PointerMotionMask);
    let expected = ViewportEvent::Input(ViewportInput::Motion {
        position: [20, 30],
        modifiers: ViewportModifiers {
            shift: true,
            control: false,
            ..Default::default()
        },
    });
    assert!(fixture.window.poll(64).unwrap().contains(&expected));
}
#[test]
fn navigation_keys_use_the_native_keymap_and_preserve_press_release_state() {
    use x11_dl::keysym;
    let fixture = NativeInputFixture::new();
    for (symbol, key) in [
        (keysym::XK_F2, ViewportKey::Pan),
        (keysym::XK_F3, ViewportKey::Zoom),
        (keysym::XK_F4, ViewportKey::Orbit),
        (keysym::XK_F6, ViewportKey::Home),
        (keysym::XK_F5, ViewportKey::Previous),
        (keysym::XK_Escape, ViewportKey::Cancel),
    ] {
        check_mapped_key(&fixture, symbol, key);
    }
    check_unrelated_native_input(&fixture);
}
fn check_unrelated_native_input(fixture: &NativeInputFixture) {
    assert!(
        fixture
            .window
            .translate(fixture.key(x11_dl::keysym::XK_space, true))
            .is_none()
    );
    let event = fixture.event(xlib::Expose);
    assert!(translate_native_pointer(fixture.window.pointer(), event).is_none());
    assert!(copied_button(9).is_none());
}

fn check_mapped_key(fixture: &NativeInputFixture, symbol: u32, key: ViewportKey) {
    for pressed in [true, false] {
        assert_eq!(
            copied_command(
                &fixture
                    .window
                    .copy_event(fixture.key(symbol, pressed))
                    .unwrap()
            ),
            Some((key, pressed))
        );
    }
}

#[test]
fn missing_and_foreign_display_pointers_are_declined_before_key_lookup() {
    let fixture = NativeInputFixture::new();
    let mut event = fixture.key(x11_dl::keysym::XK_F2, true);
    event.key.display = std::ptr::null_mut();
    assert!(fixture.window.copy_event(event).unwrap().is_empty());
    event.key.display = std::ptr::dangling_mut();
    assert!(fixture.window.copy_event(event).unwrap().is_empty());
    assert!(translate_native_pointer(std::ptr::null_mut(), event).is_none());
}

#[test]
fn native_save_chord_requires_control_without_shift_or_alt() {
    let fixture = NativeInputFixture::new();
    let cases = [
        (0, false),
        (xlib::ControlMask, true),
        (xlib::ControlMask | xlib::ShiftMask, false),
        (xlib::ControlMask | xlib::Mod1Mask, false),
    ];
    for (state, accepted) in cases {
        assert_eq!(control_only_chord(state), accepted);
        check_native_save_key(&fixture, state, accepted);
    }
}
fn check_native_save_key(fixture: &NativeInputFixture, state: u32, accepted: bool) {
    for pressed in [true, false] {
        let mut event = fixture.key(x11_dl::keysym::XK_s, pressed);
        event.key.state = state;
        let expected = accepted.then_some((ViewportKey::Save, pressed));
        assert_eq!(
            copied_command(&fixture.window.copy_event(event).unwrap()),
            expected
        );
    }
}

#[test]
fn native_history_chords_use_control_z_control_y_and_control_shift_z() {
    use x11_dl::keysym::{XK_y, XK_z};
    let fixture = NativeInputFixture::new();
    let control = xlib::ControlMask;
    let cases = [
        (XK_z, control, Some(ViewportKey::Undo)),
        (XK_y, control, Some(ViewportKey::Redo)),
        (XK_z, control | xlib::ShiftMask, Some(ViewportKey::Redo)),
        (XK_y, control | xlib::ShiftMask, None),
        (XK_z, 0, None),
        (XK_z, control | xlib::ShiftMask | xlib::Mod1Mask, None),
    ];
    for (symbol, state, key) in cases {
        verify_native_history_key(&fixture, symbol, state, key);
    }
}
fn verify_native_history_key(
    fixture: &NativeInputFixture,
    symbol: u32,
    state: u32,
    key: Option<ViewportKey>,
) {
    for pressed in [true, false] {
        let mut event = fixture.key(symbol, pressed);
        event.key.state = state;
        let expected = key.map(|key| (key, pressed));
        assert_eq!(
            copied_command(&fixture.window.copy_event(event).unwrap()),
            expected
        );
    }
}

fn copied_command(events: &[ViewportEvent]) -> Option<(ViewportKey, bool)> {
    events.iter().find_map(|event| match event {
        ViewportEvent::Input(ViewportInput::PhysicalKey {
            command: Some(key),
            pressed,
            ..
        })
        | ViewportEvent::Input(ViewportInput::Key { key, pressed }) => Some((*key, *pressed)),
        _ => None,
    })
}
#[path = "x11_keyboard.rs"]
mod keyboard_tests;
