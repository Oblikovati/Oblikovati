// SPDX-License-Identifier: GPL-2.0-only
//! Check native test input against reparented X11 windows.
use super::*;
use x11rb::protocol::xproto::{CreateWindowAux, WindowClass};
use x11rb::wrapper::ConnectionExt as _;

#[test]
fn finds_application_inside_window_manager_frame() {
    let _lease = crate::test_context::acquire();
    let (connection, screen) = x11rb::connect(None).unwrap();
    let root = connection.setup().roots[screen].root;
    let frame = create_child(&connection, root);
    let application = create_child(&connection, frame);
    connection
        .change_property8(
            x11rb::protocol::xproto::PropMode::REPLACE,
            application,
            AtomEnum::WM_NAME,
            AtomEnum::STRING,
            b"Oblikovati fixture",
        )
        .unwrap()
        .check()
        .unwrap();
    connection.flush().unwrap();
    assert_eq!(application_window(&connection, root), application);
    let second = create_child(&connection, root);
    connection
        .change_property8(
            x11rb::protocol::xproto::PropMode::REPLACE,
            second,
            AtomEnum::WM_NAME,
            AtomEnum::STRING,
            b"Oblikovati second fixture",
        )
        .unwrap()
        .check()
        .unwrap();
    connection.flush().unwrap();
    assert!(std::panic::catch_unwind(|| application_window(&connection, root)).is_err());
}

fn create_child(connection: &RustConnection, parent: Window) -> Window {
    let window = connection.generate_id().unwrap();
    connection
        .create_window(
            0,
            window,
            parent,
            0,
            0,
            100,
            100,
            0,
            WindowClass::INPUT_OUTPUT,
            0,
            &CreateWindowAux::new(),
        )
        .unwrap()
        .check()
        .unwrap();
    window
}
