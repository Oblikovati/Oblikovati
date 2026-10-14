// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::ViewportPhysicalKey;
#[test]
fn invalid_native_keycode_is_refused_before_filtering_or_native_lookup() {
    let fixture = NativeInputFixture::new();
    let mut event = fixture.key(x11_dl::keysym::XK_a, true);
    event.key.keycode = 256;
    assert!(
        fixture
            .window
            .copy_event(event)
            .unwrap_err()
            .contains("256")
    );
}
#[test]
fn shortcut_letter_uses_the_active_xkb_group_instead_of_group_zero() {
    let server = PrivateX11Server::open_compose_fixture().unwrap();
    let fixture = NativeInputFixture::on_display(server.display());
    let mut event = fixture.key(x11_dl::keysym::XK_a, true);
    let mut symbols = [
        xlib::KeySym::from(b'a'),
        xlib::KeySym::from(b'A'),
        xlib::KeySym::from(b's'),
        xlib::KeySym::from(b'S'),
    ];
    install_private_key_symbols(&fixture, unsafe { event.key.keycode } as i32, &mut symbols);
    // XKB.h XkbBuildCoreState encodes group one in bit 13.
    event.key.state = xlib::ControlMask | (1 << 13);
    let events = fixture.window.copy_event(event).unwrap();
    assert_group_one_save(&events);
}
#[test]
fn remapped_letter_keeps_physical_position_and_layout_shortcut_through_release() {
    let server = PrivateX11Server::open_compose_fixture().unwrap();
    let fixture = NativeInputFixture::on_display(server.display());
    let mut press = fixture.key(x11_dl::keysym::XK_a, true);
    remap_private_letter(&fixture, unsafe { press.key.keycode }, x11_dl::keysym::XK_q);
    press.key.state = xlib::ControlMask;
    assert_layout_press(&fixture.window.copy_event(press).unwrap(), true);
    remap_private_letter(&fixture, unsafe { press.key.keycode }, x11_dl::keysym::XK_z);
    press.key.type_ = xlib::KeyRelease;
    assert_layout_press(&fixture.window.copy_event(press).unwrap(), false);
}
fn assert_layout_press(events: &[ViewportEvent], pressed: bool) {
    assert_eq!(
        events,
        [ViewportEvent::Input(ViewportInput::PhysicalKey {
            key: ViewportPhysicalKey::A,
            layout_key: ViewportPhysicalKey::Q,
            command: None,
            pressed,
            modifiers: ViewportModifiers {
                control: true,
                ..Default::default()
            },
        })]
    );
}
fn remap_private_letter(fixture: &NativeInputFixture, code: u32, symbol: u32) {
    let mut symbols = [xlib::KeySym::from(symbol), xlib::KeySym::from(symbol - 32)];
    install_private_key_symbols(fixture, code as i32, &mut symbols);
}
#[test]
fn isolated_native_input_method_commits_dead_key_composition_and_supplementary_utf8() {
    let server = PrivateX11Server::open_compose_fixture().unwrap();
    let fixture = NativeInputFixture::on_display(server.display());
    let dead = typed_native_symbol(&fixture, x11_dl::keysym::XK_dead_acute);
    assert!(committed_characters(&dead).is_empty());
    assert_eq!(
        committed_characters(&typed_native_symbol(&fixture, x11_dl::keysym::XK_e)),
        "é"
    );
    install_supplementary_key(&fixture);
    assert_eq!(
        committed_characters(&typed_native_symbol(&fixture, 0x0101_f642)),
        "🙂"
    );
}
fn typed_native_symbol(fixture: &NativeInputFixture, symbol: u32) -> Vec<ViewportEvent> {
    fixture.window.poll(64).unwrap();
    let press = fixture.key(symbol, true);
    assert_ne!(
        unsafe { press.key.keycode },
        0,
        "keysym {symbol}: expected a mapped fixture key"
    );
    fixture.send(press, xlib::KeyPressMask);
    let events = fixture.window.poll(64).unwrap();
    fixture.send(fixture.key(symbol, false), xlib::KeyReleaseMask);
    fixture.window.poll(64).unwrap();
    events
}
fn committed_characters(events: &[ViewportEvent]) -> String {
    events
        .iter()
        .filter_map(|event| match event {
            ViewportEvent::Input(ViewportInput::Text(character)) => Some(*character),
            _ => None,
        })
        .collect()
}
fn install_supplementary_key(fixture: &NativeInputFixture) {
    let mut minimum = 0;
    let mut maximum = 0;
    unsafe {
        (fixture.window.x11.api.XDisplayKeycodes)(
            fixture.window.pointer(),
            &mut minimum,
            &mut maximum,
        )
    };
    let mut symbol: xlib::KeySym = 0x0101_f642;
    install_private_key_symbols(fixture, maximum, std::slice::from_mut(&mut symbol));
}
fn install_private_key_symbols(
    fixture: &NativeInputFixture,
    code: i32,
    symbols: &mut [xlib::KeySym],
) {
    unsafe {
        (fixture.window.x11.api.XChangeKeyboardMapping)(
            fixture.window.pointer(),
            code,
            symbols.len() as i32,
            symbols.as_mut_ptr(),
            1,
        )
    };
    fixture.window.sync();
    fixture.window.poll(64).unwrap();
}
#[test]
fn real_native_queue_delivers_physical_keys_and_committed_text_across_poll_limits() {
    let fixture = NativeInputFixture::new();
    fixture.window.poll(64).unwrap();
    fixture.send(fixture.key(x11_dl::keysym::XK_a, true), xlib::KeyPressMask);
    assert!(fixture.window.poll(1).unwrap().is_empty());
    assert_copied_input_ready(&fixture);
    assert_native_a_press(&fixture.window.poll(1).unwrap());
    assert_eq!(
        fixture.window.poll(1).unwrap(),
        [ViewportEvent::Input(ViewportInput::Text('a'))]
    );
    assert_native_release_without_text(&fixture);
}
#[test]
fn native_shortcut_repeat_is_suppressed_and_release_keeps_its_press_identity() {
    let fixture = NativeInputFixture::new();
    let mut press = fixture.key(x11_dl::keysym::XK_s, true);
    press.key.state = xlib::ControlMask;
    assert_eq!(
        copied_command(&fixture.window.copy_event(press).unwrap()),
        Some((ViewportKey::Save, true))
    );
    assert_eq!(
        copied_command(&fixture.window.copy_event(press).unwrap()),
        None
    );
    let release = fixture.key(x11_dl::keysym::XK_s, false);
    assert_eq!(
        copied_command(&fixture.window.copy_event(release).unwrap()),
        Some((ViewportKey::Save, false))
    );
    assert_eq!(
        copied_command(&fixture.window.copy_event(press).unwrap()),
        Some((ViewportKey::Save, true))
    );
}
#[test]
fn native_modifier_release_keeps_the_other_side_held() {
    let fixture = NativeInputFixture::new();
    for (symbol, pressed, expected) in [
        (x11_dl::keysym::XK_Control_L, true, true),
        (x11_dl::keysym::XK_Control_R, true, true),
        (x11_dl::keysym::XK_Control_L, false, true),
        (x11_dl::keysym::XK_Control_R, false, false),
    ] {
        let mut event = fixture.key(symbol, pressed);
        event.key.state = xlib::ControlMask;
        let events = fixture.window.copy_event(event).unwrap();
        let [ViewportEvent::Input(ViewportInput::PhysicalKey { modifiers, .. })] =
            events.as_slice()
        else {
            panic!("modifier event {events:?}: expected one physical key");
        };
        assert_eq!(modifiers.control, expected);
    }
}
#[test]
fn altgr_and_super_modifiers_do_not_trigger_plain_control_shortcuts() {
    let fixture = NativeInputFixture::new();
    for extra in [xlib::Mod1Mask, xlib::Mod4Mask, xlib::Mod5Mask] {
        let mut event = fixture.key(x11_dl::keysym::XK_s, true);
        event.key.state = xlib::ControlMask | extra;
        assert_eq!(
            copied_command(&fixture.window.copy_event(event).unwrap()),
            None
        );
        fixture
            .window
            .copy_event(fixture.key(x11_dl::keysym::XK_s, false))
            .unwrap();
    }
}
#[test]
fn focus_or_visibility_loss_retires_pending_shortcut_ownership_without_text() {
    for (kind, expected) in [
        (xlib::FocusOut, ViewportEvent::Focus(false)),
        (xlib::UnmapNotify, ViewportEvent::Mapped(false)),
    ] {
        verify_retired_shortcut(kind, expected);
    }
}
fn verify_retired_shortcut(kind: i32, expected: ViewportEvent) {
    let fixture = NativeInputFixture::new();
    let mut event = fixture.key(x11_dl::keysym::XK_s, true);
    event.key.state = xlib::ControlMask;
    fixture.window.copy_event(event).unwrap();
    let focus_notice = fixture
        .window
        .copy_event(fixture.event(xlib::FocusIn))
        .unwrap();
    assert_eq!(focus_notice, [ViewportEvent::Focus(true)]);
    assert_eq!(
        fixture.window.copy_event(fixture.event(kind)).unwrap(),
        [expected]
    );
    let release = fixture.key(x11_dl::keysym::XK_s, false);
    assert_eq!(
        copied_command(&fixture.window.copy_event(release).unwrap()),
        None
    );
}

fn assert_native_a_press(pressed: &[ViewportEvent]) {
    assert!(matches!(
        pressed,
        [ViewportEvent::Input(ViewportInput::PhysicalKey {
            key: ViewportPhysicalKey::A,
            pressed: true,
            command: None,
            ..
        })]
    ));
}

fn assert_copied_input_ready(fixture: &NativeInputFixture) {
    let wake = crate::ViewportWake::new().unwrap();
    assert!(
        fixture
            .window
            .wait_events(&wake, Some(std::time::Duration::ZERO))
            .unwrap()
            .native_events
    );
}

fn assert_native_release_without_text(fixture: &NativeInputFixture) {
    fixture.send(
        fixture.key(x11_dl::keysym::XK_a, false),
        xlib::KeyReleaseMask,
    );
    assert!(
        fixture
            .window
            .poll(64)
            .unwrap()
            .iter()
            .all(|event| !matches!(event, ViewportEvent::Input(ViewportInput::Text(_))))
    );
}

fn assert_group_one_save(events: &[ViewportEvent]) {
    assert!(matches!(
        events,
        [ViewportEvent::Input(ViewportInput::PhysicalKey {
            key: ViewportPhysicalKey::A,
            layout_key: ViewportPhysicalKey::S,
            command: Some(ViewportKey::Save),
            ..
        })]
    ));
}
