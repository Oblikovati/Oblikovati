// SPDX-License-Identifier: GPL-2.0-only
//! Send the window manager close protocol through the real X11 connection.

use x11rb::connection::Connection;
use x11rb::protocol::xproto::{
    Atom, AtomEnum, BUTTON_PRESS_EVENT, BUTTON_RELEASE_EVENT, ClientMessageEvent, ConnectionExt,
    EventMask, MOTION_NOTIFY_EVENT, Window,
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
    let (connection, screen) = x11rb::connect(None).expect("X11: expected a display connection");
    let root = connection.setup().roots[screen].root;
    let window = application_window(&connection, root);
    let [x, y] = root_position(&connection, root, window, position);
    send_click(&connection, root, x, y);
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

fn send_click(connection: &RustConnection, root: Window, x: i16, y: i16) {
    for (event, detail) in [
        (MOTION_NOTIFY_EVENT, 0),
        (BUTTON_PRESS_EVENT, 1),
        (BUTTON_RELEASE_EVENT, 1),
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
    let tree = connection.query_tree(root).unwrap().reply().unwrap();
    let windows: Vec<_> = tree
        .children
        .into_iter()
        .filter(|id| is_application(connection, *id))
        .collect();
    let [window] = windows.as_slice() else {
        panic!("X11 application windows {windows:?}: expected exactly one Oblikovati window");
    };
    *window
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
