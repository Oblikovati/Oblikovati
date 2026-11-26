// SPDX-License-Identifier: GPL-2.0-only
//! Find and drive named native windows independently of application controls.
use super::*;

/// Send a native close to the selected drawing window. Example: `request_named_window_close("Drawing1")`.
pub(in super::super) fn request_named_window_close(title: &str) {
    let (connection, screen) = x11rb::connect(None).unwrap();
    let root = connection.setup().roots[screen].root;
    let window = named_window(&connection, root, title);
    send_window_close(&connection, window);
}

/// Send a key to a specific top-level window. Example: `press_window_key("Drawing1", 0xff1b)`.
pub(in super::super) fn press_window_key(title: &str, keysym: u32) {
    let (connection, screen) = x11rb::connect(None).unwrap();
    let root = connection.setup().roots[screen].root;
    let window = named_window(&connection, root, title);
    focus_target(&connection, window);
    press_target_key(&connection, root, keysym);
}

/// Click a fullscreen nested Wayland compositor at its X11 output coordinates.
pub(in super::super) fn click_fullscreen_output([x, y]: [i32; 2]) {
    let (connection, screen) = x11rb::connect(None).expect("X11: expected a display");
    let root = connection.setup().roots[screen].root;
    send_click(
        &connection,
        root,
        x.try_into().unwrap(),
        y.try_into().unwrap(),
        1,
    );
}

pub(in super::super) fn move_fullscreen_pointer(position: [i32; 2]) {
    fullscreen_pointer_event(position, MOTION_NOTIFY_EVENT, 0);
}

pub(in super::super) fn fullscreen_left_button(position: [i32; 2], pressed: bool) {
    let event = if pressed {
        BUTTON_PRESS_EVENT
    } else {
        BUTTON_RELEASE_EVENT
    };
    fullscreen_pointer_event(position, event, 1);
}

fn fullscreen_pointer_event([x, y]: [i32; 2], event: u8, detail: u8) {
    let (connection, screen) =
        x11rb::connect(None).expect("X11: expected a private nested display");
    let root = connection.setup().roots[screen].root;
    connection
        .xtest_fake_input(
            event,
            detail,
            0,
            root,
            x.try_into().unwrap(),
            y.try_into().unwrap(),
            0,
        )
        .unwrap()
        .check()
        .unwrap();
}

pub(super) fn application_window(connection: &RustConnection, root: Window) -> Window {
    let windows = application_descendants(connection, root);
    let [window] = windows.as_slice() else {
        panic!("X11 application windows {windows:?}: expected exactly one Oblikovati window");
    };
    *window
}

fn application_descendants(connection: &RustConnection, root: Window) -> Vec<Window> {
    matching_descendants(connection, root, |name| name.contains("Oblikovati"))
}

fn named_window(connection: &RustConnection, root: Window, title: &str) -> Window {
    let windows = matching_descendants(connection, root, |name| name == title);
    let [one] = windows.as_slice() else {
        panic!("X11 windows {windows:?}: expected exactly one separate window named {title:?}");
    };
    *one
}

/// Identify a real native window independently of GUI draw data. Example: `window_is_separate("Drawing1")`.
pub(in super::super) fn window_is_separate(title: &str) -> bool {
    let (connection, screen) = x11rb::connect(None).unwrap();
    let root = connection.setup().roots[screen].root;
    let drawing = named_window(&connection, root, title);
    let main = application_window(&connection, root);
    top_level_frame(&connection, root, drawing) != top_level_frame(&connection, root, main)
}

fn top_level_frame(connection: &RustConnection, root: Window, mut window: Window) -> Window {
    loop {
        let parent = connection
            .query_tree(window)
            .unwrap()
            .reply()
            .unwrap()
            .parent;
        if parent == root {
            return window;
        }
        assert_ne!(
            parent, 0,
            "window {window}: expected to descend from root {root}"
        );
        window = parent;
    }
}

fn matching_descendants(
    connection: &RustConnection,
    root: Window,
    matches_title: impl Fn(&str) -> bool,
) -> Vec<Window> {
    let mut pending = vec![root];
    let mut matches = Vec::new();
    while let Some(parent) = pending.pop() {
        let Ok(query) = connection.query_tree(parent) else {
            continue;
        };
        let Ok(tree) = query.reply() else { continue };
        for child in tree.children {
            if window_title(connection, child).is_some_and(|title| matches_title(&title)) {
                matches.push(child);
            }
            pending.push(child);
        }
    }
    matches
}

fn window_title(connection: &RustConnection, window: Window) -> Option<String> {
    let reply = connection
        .get_property(false, window, AtomEnum::WM_NAME, AtomEnum::STRING, 0, 1024)
        .ok()?;
    let property = reply.reply().ok()?;
    Some(String::from_utf8_lossy(&property.value).into_owned())
}
