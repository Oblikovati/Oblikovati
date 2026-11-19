// SPDX-License-Identifier: GPL-2.0-only
//! Send the window manager close protocol through the real X11 connection.

mod cursor;
pub(super) use cursor::{cursor_sprite, reference_cursor};
use x11rb::connection::Connection;
use x11rb::protocol::xproto::{
    Atom, AtomEnum, BUTTON_PRESS_EVENT, BUTTON_RELEASE_EVENT, ClientMessageEvent, ConnectionExt,
    EventMask, InputFocus, KEY_PRESS_EVENT, KEY_RELEASE_EVENT, MOTION_NOTIFY_EVENT, Window,
};
use x11rb::protocol::xtest::ConnectionExt as _;
use x11rb::rust_connection::RustConnection;

pub(super) fn request_native_close() {
    let (connection, screen) = x11rb::connect(None).expect("X11: expected a display connection");
    let root = connection.setup().roots[screen].root;
    let window = application_window(&connection, root);
    let protocol = atom(&connection, b"WM_PROTOCOLS");
    let delete = atom(&connection, b"WM_DELETE_WINDOW");
    let event = ClientMessageEvent::new(32, window, protocol, [delete, 0, 0, 0, 0]);
    connection
        .send_event(false, window, EventMask::NO_EVENT, event)
        .expect("X11 close: expected a protocol event")
        .check()
        .expect("X11 close: expected the application window to receive the event");
    connection
        .flush()
        .expect("X11 close: expected a flushed request");
}

pub(super) fn click(position: [i32; 2]) {
    let (connection, root, x, y) = pointer_target(position);
    send_click(&connection, root, x, y, 1);
}

/// Change the real left-button state; example: `left_button(at, true)` starts a drag.
pub(super) fn left_button(position: [i32; 2], pressed: bool) {
    let (connection, root, x, y) = pointer_target(position);
    let event = if pressed {
        BUTTON_PRESS_EVENT
    } else {
        BUTTON_RELEASE_EVENT
    };
    connection
        .xtest_fake_input(event, 1, 0, root, x, y, 0)
        .unwrap()
        .check()
        .unwrap();
    connection
        .flush()
        .expect("X11 button: expected a flushed request");
}

pub(super) fn right_click(position: [i32; 2]) {
    let (connection, root, x, y) = pointer_target(position);
    send_click(&connection, root, x, y, 3);
}

pub(super) fn scroll(position: [i32; 2], notches: i32) {
    assert!(
        (1..=20).contains(&notches.unsigned_abs()),
        "wheel notches {notches}: expected 1 to 20 steps in either direction"
    );
    let (connection, root, x, y) = pointer_target(position);
    connection
        .xtest_fake_input(MOTION_NOTIFY_EVENT, 0, 0, root, x, y, 0)
        .unwrap()
        .check()
        .unwrap();
    let button = if notches > 0 { 4 } else { 5 };
    for _ in 0..notches.unsigned_abs() {
        for event in [BUTTON_PRESS_EVENT, BUTTON_RELEASE_EVENT] {
            connection
                .xtest_fake_input(event, button, 0, root, x, y, 0)
                .unwrap()
                .check()
                .unwrap();
        }
    }
    connection
        .flush()
        .expect("X11 wheel: expected a flushed request");
}

pub(super) fn move_pointer(position: [i32; 2]) {
    let (connection, root, x, y) = pointer_target(position);
    connection
        .xtest_fake_input(MOTION_NOTIFY_EVENT, 0, 0, root, x, y, 0)
        .unwrap()
        .check()
        .unwrap();
    connection
        .flush()
        .expect("X11 motion: expected a flushed request");
}

/// Focus the native application window; example: `focus_window()` before pointer input.
pub(super) fn focus_window() {
    let (connection, _) = keyboard_target();
    connection
        .flush()
        .expect("X11 focus: expected a flushed request");
}

/// Send a mapped key; example: `press_keysym(u32::from(b'a'))` types a letter.
pub(super) fn press_keysym(keysym: u32) {
    let (connection, root) = keyboard_target();
    let key = keyboard_keycode(&connection, keysym);
    send_key_sequence(
        &connection,
        root,
        &[(KEY_PRESS_EVENT, key), (KEY_RELEASE_EVENT, key)],
    );
}

/// Type one ASCII path character through X11; example: `press_path_character('A')`.
pub(super) fn press_path_character(character: char) {
    let shifted = character.is_ascii_uppercase() || character == '_';
    let base = if character == '_' {
        '-'
    } else {
        character.to_ascii_lowercase()
    };
    assert!(
        base.is_ascii_alphanumeric() || matches!(base, '/' | '.' | '-' | ' '),
        "path character {character:?}: expected an ASCII letter, digit, slash, dot, hyphen, underscore or space"
    );
    if shifted {
        press_shifted_keysym(u32::from(base));
    } else {
        press_keysym(u32::from(base));
    }
}

fn press_shifted_keysym(keysym: u32) {
    let (connection, root) = keyboard_target();
    let shift = keyboard_keycode(&connection, 0xffe1);
    let key = keyboard_keycode(&connection, keysym);
    send_key_sequence(
        &connection,
        root,
        &[
            (KEY_PRESS_EVENT, shift),
            (KEY_PRESS_EVENT, key),
            (KEY_RELEASE_EVENT, key),
            (KEY_RELEASE_EVENT, shift),
        ],
    );
}

pub(super) fn press_escape() {
    let (connection, root) = keyboard_target();
    let escape = keyboard_keycode(&connection, 0xff1b);
    send_key_sequence(
        &connection,
        root,
        &[(KEY_PRESS_EVENT, escape), (KEY_RELEASE_EVENT, escape)],
    );
}

pub(super) fn press_alt_f4() {
    let (connection, root) = keyboard_target();
    let alt = keyboard_keycode(&connection, 0xffe9);
    let f4 = keyboard_keycode(&connection, 0xffc1);
    send_key_sequence(
        &connection,
        root,
        &[
            (KEY_PRESS_EVENT, alt),
            (KEY_PRESS_EVENT, f4),
            (KEY_RELEASE_EVENT, f4),
            (KEY_RELEASE_EVENT, alt),
        ],
    );
}

pub(super) fn press_control_o() {
    press_control_letter(b'o');
}

pub(super) fn press_control_s() {
    press_control_letter(b's');
}

/// Switch native tabs; example: `press_control_tab(true)` selects the previous tab.
pub(super) fn press_control_tab(reverse: bool) {
    let (connection, root) = keyboard_target();
    let control = keyboard_keycode(&connection, 0xffe3);
    let shift = keyboard_keycode(&connection, 0xffe1);
    let tab = keyboard_keycode(&connection, 0xff09);
    let mut events = vec![(KEY_PRESS_EVENT, control)];
    if reverse {
        events.push((KEY_PRESS_EVENT, shift));
    }
    events.extend([(KEY_PRESS_EVENT, tab), (KEY_RELEASE_EVENT, tab)]);
    if reverse {
        events.push((KEY_RELEASE_EVENT, shift));
    }
    events.push((KEY_RELEASE_EVENT, control));
    send_key_sequence(&connection, root, &events);
}

fn press_control_letter(key: u8) {
    let (connection, root) = keyboard_target();
    let control = keyboard_keycode(&connection, 0xffe3);
    let letter = keyboard_keycode(&connection, u32::from(key));
    send_key_sequence(
        &connection,
        root,
        &[
            (KEY_PRESS_EVENT, control),
            (KEY_PRESS_EVENT, letter),
            (KEY_RELEASE_EVENT, letter),
            (KEY_RELEASE_EVENT, control),
        ],
    );
}

fn keyboard_target() -> (RustConnection, Window) {
    let (connection, screen) = x11rb::connect(None).expect("X11: expected a display connection");
    let root = connection.setup().roots[screen].root;
    let window = application_window(&connection, root);
    connection
        .set_input_focus(InputFocus::PARENT, window, 0_u32)
        .unwrap()
        .check()
        .unwrap();
    (connection, root)
}

fn keyboard_keycode(connection: &RustConnection, keysym: u32) -> u8 {
    let setup = connection.setup();
    let first = setup.min_keycode;
    let count = setup.max_keycode - first + 1;
    let mapping = connection
        .get_keyboard_mapping(first, count)
        .unwrap()
        .reply()
        .unwrap();
    mapping
        .keysyms
        .chunks(usize::from(mapping.keysyms_per_keycode))
        .position(|symbols| symbols.contains(&keysym))
        .map(|offset| first + u8::try_from(offset).unwrap())
        .unwrap_or_else(|| panic!("X11 keysym {keysym:#x}: expected a mapped key"))
}

fn send_key_sequence(connection: &RustConnection, root: Window, events: &[(u8, u8)]) {
    for &(event, key) in events {
        connection
            .xtest_fake_input(event, key, 0, root, 0, 0, 0)
            .unwrap()
            .check()
            .unwrap();
    }
    connection
        .flush()
        .expect("X11 keyboard: expected a flushed request");
}

fn pointer_target(position: [i32; 2]) -> (RustConnection, Window, i16, i16) {
    let (connection, screen) = x11rb::connect(None).expect("X11: expected a display connection");
    let root = connection.setup().roots[screen].root;
    let window = application_window(&connection, root);
    let [x, y] = root_position(&connection, root, window, position);
    (connection, root, x, y)
}

fn root_position(
    connection: &RustConnection,
    root: Window,
    window: Window,
    position: [i32; 2],
) -> [i16; 2] {
    let origin = connection
        .translate_coordinates(window, root, 0, 0)
        .unwrap()
        .reply()
        .unwrap();
    [
        coordinate(origin.dst_x, position[0]),
        coordinate(origin.dst_y, position[1]),
    ]
}

fn coordinate(base: i16, offset: i32) -> i16 {
    let value = i32::from(base) + offset;
    i16::try_from(value).unwrap_or_else(|_| panic!("X11 click coordinate {value}: expected i16"))
}

fn send_click(connection: &RustConnection, root: Window, x: i16, y: i16, button: u8) {
    for (event, detail) in [
        (MOTION_NOTIFY_EVENT, 0),
        (BUTTON_PRESS_EVENT, button),
        (BUTTON_RELEASE_EVENT, button),
    ] {
        connection
            .xtest_fake_input(event, detail, 0, root, x, y, 0)
            .unwrap()
            .check()
            .unwrap();
    }
    connection
        .flush()
        .expect("X11 click: expected a flushed request");
}

fn application_window(connection: &RustConnection, root: Window) -> Window {
    let windows = application_descendants(connection, root);
    let [window] = windows.as_slice() else {
        panic!("X11 application windows {windows:?}: expected exactly one Oblikovati window");
    };
    *window
}

fn application_descendants(connection: &RustConnection, root: Window) -> Vec<Window> {
    let mut pending = vec![root];
    let mut matches = Vec::new();
    while let Some(parent) = pending.pop() {
        let Ok(query) = connection.query_tree(parent) else {
            continue;
        };
        let Ok(tree) = query.reply() else { continue };
        for child in tree.children {
            if is_application(connection, child) {
                matches.push(child);
            }
            pending.push(child);
        }
    }
    matches
}

fn is_application(connection: &RustConnection, window: Window) -> bool {
    let reply =
        connection.get_property(false, window, AtomEnum::WM_NAME, AtomEnum::STRING, 0, 1024);
    let Ok(reply) = reply else { return false };
    let Ok(property) = reply.reply() else {
        return false;
    };
    String::from_utf8_lossy(&property.value).contains("Oblikovati")
}

fn atom(connection: &RustConnection, name: &[u8]) -> Atom {
    connection
        .intern_atom(false, name)
        .unwrap()
        .reply()
        .unwrap()
        .atom
}

pub(super) fn press_control_a() {
    let (connection, root) = keyboard_target();
    let control = keyboard_keycode(&connection, 0xffe3);
    let letter = keyboard_keycode(&connection, u32::from(b'a'));
    send_key_sequence(
        &connection,
        root,
        &[
            (KEY_PRESS_EVENT, control),
            (KEY_PRESS_EVENT, letter),
            (KEY_RELEASE_EVENT, letter),
            (KEY_RELEASE_EVENT, control),
        ],
    );
}

pub(super) fn type_digits(digits: &str) {
    assert!(
        digits.bytes().all(|digit| digit.is_ascii_digit()),
        "digits {digits:?}: expected ASCII digits"
    );
    let (connection, root) = keyboard_target();
    for digit in digits.bytes() {
        let key = keyboard_keycode(&connection, u32::from(digit));
        send_key_sequence(
            &connection,
            root,
            &[(KEY_PRESS_EVENT, key), (KEY_RELEASE_EVENT, key)],
        );
    }
}

#[cfg(test)]
#[path = "x11_input_tests.rs"]
mod tests;

/// Read application bounds in root pixels; example: `application_bounds()` before a resize.
pub(super) fn application_bounds() -> [i32; 4] {
    let (connection, screen) = x11rb::connect(None).unwrap();
    let root = connection.setup().roots[screen].root;
    let window = application_window(&connection, root);
    let geometry = connection.get_geometry(window).unwrap().reply().unwrap();
    let origin = connection
        .translate_coordinates(window, root, 0, 0)
        .unwrap()
        .reply()
        .unwrap();
    [
        i32::from(origin.dst_x),
        i32::from(origin.dst_y),
        i32::from(geometry.width),
        i32::from(geometry.height),
    ]
}

pub(super) fn window_has_state(name: &[u8]) -> bool {
    let (connection, screen) = x11rb::connect(None).expect("X11: expected a display connection");
    let window = application_window(&connection, connection.setup().roots[screen].root);
    let state = atom(&connection, b"_NET_WM_STATE");
    let expected = atom(&connection, name);
    let property = connection
        .get_property(false, window, state, AtomEnum::ATOM, 0, 64)
        .unwrap()
        .reply()
        .unwrap();
    property
        .value32()
        .is_some_and(|mut values| values.any(|value| value == expected))
}

pub(super) fn activate_from_taskbar() {
    let (connection, screen) = x11rb::connect(None).expect("X11: expected a display connection");
    let root = connection.setup().roots[screen].root;
    let window = application_window(&connection, root);
    let event = ClientMessageEvent::new(
        32,
        window,
        atom(&connection, b"_NET_ACTIVE_WINDOW"),
        [2, 0, 0, 0, 0],
    );
    connection
        .send_event(
            false,
            root,
            EventMask::SUBSTRUCTURE_REDIRECT | EventMask::SUBSTRUCTURE_NOTIFY,
            event,
        )
        .unwrap()
        .check()
        .unwrap();
    connection.flush().unwrap();
}

pub(super) fn click_test_taskbar() {
    let (connection, screen) = x11rb::connect(None).expect("X11: expected a display connection");
    let root = &connection.setup().roots[screen];
    let y = i16::try_from(root.height_in_pixels - 16).unwrap();
    send_click(&connection, root.root, 100, y, 1);
}

pub(super) fn desktop_work_area() -> [i32; 4] {
    let (connection, screen) = x11rb::connect(None).expect("X11: expected a display connection");
    let root = connection.setup().roots[screen].root;
    let property = connection
        .get_property(
            false,
            root,
            atom(&connection, b"_NET_WORKAREA"),
            AtomEnum::CARDINAL,
            0,
            4,
        )
        .unwrap()
        .reply()
        .unwrap();
    let values: Vec<i32> = property
        .value32()
        .unwrap()
        .map(|value| value as i32)
        .collect();
    values
        .try_into()
        .expect("_NET_WORKAREA: expected the current desktop rectangle")
}

pub(super) fn frame_extents() -> [u32; 4] {
    let (connection, screen) = x11rb::connect(None).expect("X11: expected a display connection");
    let root = connection.setup().roots[screen].root;
    let window = application_window(&connection, root);
    let property = connection
        .get_property(
            false,
            window,
            atom(&connection, b"_NET_FRAME_EXTENTS"),
            AtomEnum::CARDINAL,
            0,
            4,
        )
        .unwrap()
        .reply()
        .unwrap();
    let values: Vec<u32> = property.value32().unwrap().collect();
    values
        .try_into()
        .expect("_NET_FRAME_EXTENTS: expected four decoration widths")
}
