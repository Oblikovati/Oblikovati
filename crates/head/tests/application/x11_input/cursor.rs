// SPDX-License-Identifier: GPL-2.0-only
//! X11 cursor sprite checks.
use x11rb::connection::Connection;
use x11rb::protocol::xfixes::ConnectionExt as _;
use x11rb::protocol::xproto::ConnectionExt as _;

/// Read the current cursor sprite through XFixes for window-frame checks.
pub(crate) fn cursor_sprite() -> (u16, u16, u16, u16, Vec<u32>) {
    let (connection, _) = x11rb::connect(None).expect("X11: expected a display connection");
    connection
        .xfixes_query_version(5, 0)
        .unwrap()
        .reply()
        .unwrap();
    let cursor = connection
        .xfixes_get_cursor_image()
        .unwrap()
        .reply()
        .unwrap();
    (
        cursor.width,
        cursor.height,
        cursor.xhot,
        cursor.yhot,
        cursor.cursor_image,
    )
}

pub(crate) fn reference_cursor(shape: u16) -> (u16, u16, u16, u16, Vec<u32>) {
    use x11rb::protocol::xproto::{CreateWindowAux, WindowClass};
    let (connection, screen) = x11rb::connect(None).expect("X11: expected a display connection");
    let root = connection.setup().roots[screen].root;
    let xlib = x11_dl::xlib::Xlib::open().expect("X11: expected Xlib");
    let display = unsafe { (xlib.XOpenDisplay)(std::ptr::null()) };
    assert!(
        !display.is_null(),
        "X11: expected a display for the reference cursor"
    );
    let cursor = unsafe { (xlib.XCreateFontCursor)(display, u32::from(shape)) };
    unsafe { (xlib.XSync)(display, 0) };
    let window = connection.generate_id().unwrap();
    let cursor_id = u32::try_from(cursor).unwrap();
    let attributes = CreateWindowAux::new()
        .override_redirect(1)
        .cursor(cursor_id);
    connection
        .create_window(
            0,
            window,
            root,
            0,
            0,
            24,
            24,
            0,
            WindowClass::INPUT_OUTPUT,
            0,
            &attributes,
        )
        .unwrap()
        .check()
        .unwrap();
    connection.map_window(window).unwrap().check().unwrap();
    connection
        .warp_pointer(0u32, window, 0, 0, 0, 0, 12, 12)
        .unwrap()
        .check()
        .unwrap();
    connection
        .xfixes_query_version(5, 0)
        .unwrap()
        .reply()
        .unwrap();
    let image = connection
        .xfixes_get_cursor_image()
        .unwrap()
        .reply()
        .unwrap();
    connection.destroy_window(window).unwrap().check().unwrap();
    unsafe {
        (xlib.XFreeCursor)(display, cursor);
        (xlib.XCloseDisplay)(display);
    }
    (
        image.width,
        image.height,
        image.xhot,
        image.yhot,
        image.cursor_image,
    )
}
