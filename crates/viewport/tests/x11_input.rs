// SPDX-License-Identifier: GPL-2.0-only
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
        assert!(
            translate_pointer_key(&fixture.window.x11.api, fixture.window.pointer(), event)
                .is_none()
        );
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
        },
    }
}
#[test]
fn wheel_and_motion_events_are_copied_through_the_actual_native_queue() {
    let fixture = NativeInputFixture::new();
    fixture.window.poll(64);
    for (button, notches) in [(4, 1), (5, -1)] {
        let mut event = fixture.event(xlib::ButtonPress);
        event.button.button = button;
        fixture.send(event, xlib::ButtonPressMask);
        let expected = ViewportEvent::Input(ViewportInput::Wheel {
            notches,
            position: [0, 0],
        });
        assert!(fixture.window.poll(64).contains(&expected));
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
        },
    });
    assert!(fixture.window.poll(64).contains(&expected));
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
    assert!(
        translate_pointer_key(&fixture.window.x11.api, fixture.window.pointer(), event).is_none()
    );
    assert!(copied_button(9).is_none());
}

fn check_mapped_key(fixture: &NativeInputFixture, symbol: u32, key: ViewportKey) {
    for pressed in [true, false] {
        let expected = ViewportEvent::Input(ViewportInput::Key { key, pressed });
        assert_eq!(
            fixture.window.translate(fixture.key(symbol, pressed)),
            Some(expected)
        );
    }
}

#[test]
fn missing_and_foreign_display_pointers_are_declined_before_key_lookup() {
    let fixture = NativeInputFixture::new();
    let mut event = fixture.key(x11_dl::keysym::XK_F2, true);
    event.key.display = std::ptr::null_mut();
    assert!(fixture.window.translate(event).is_none());
    event.key.display = std::ptr::dangling_mut();
    assert!(fixture.window.translate(event).is_none());
    assert!(translate_pointer_key(&fixture.window.x11.api, std::ptr::null_mut(), event).is_none());
}
