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

/// Scroll at the cursor position on the private nested compositor's output.
pub(in super::super) fn scroll_fullscreen_output([x, y]: [i32; 2], notches: i32) {
    let (connection, screen) = x11rb::connect(None).expect("expected the compositor host display");
    let root = connection.setup().roots[screen].root;
    send_scroll(
        &connection,
        root,
        [x.try_into().unwrap(), y.try_into().unwrap()],
        notches,
    );
}

pub(in super::super) fn fullscreen_left_button(position: [i32; 2], pressed: bool) {
    fullscreen_button(position, 1, pressed);
}

pub(in super::super) fn fullscreen_button(position: [i32; 2], button: u8, pressed: bool) {
    let event = if pressed {
        BUTTON_PRESS_EVENT
    } else {
        BUTTON_RELEASE_EVENT
    };
    fullscreen_pointer_event(position, event, button);
}

pub(in super::super) fn switch_fullscreen_wayland_task() {
    let (connection, screen) =
        x11rb::connect(None).expect("X11: expected a private nested display");
    let root = connection.setup().roots[screen].root;
    let modifier = keyboard_keycode(&connection, 0xffeb);
    let tab = keyboard_keycode(&connection, 0xff09);
    send_key_sequence(
        &connection,
        root,
        &[
            (KEY_PRESS_EVENT, modifier),
            (KEY_PRESS_EVENT, tab),
            (KEY_RELEASE_EVENT, tab),
            (KEY_RELEASE_EVENT, modifier),
        ],
    );
}

/// Drive a document shortcut through the fullscreen Weston test output.
pub(in super::super) fn press_weston_control_tab(reverse: bool) {
    let (connection, root) = weston_keyboard_target();
    send_control_tab(&connection, root, reverse);
}

pub(in super::super) fn press_weston_key(keysym: u32) {
    let (connection, root) = weston_keyboard_target();
    press_target_key(&connection, root, keysym);
}

fn weston_keyboard_target() -> (RustConnection, Window) {
    let (connection, screen) = x11rb::connect(None).expect("expected the Weston host display");
    let root = connection.setup().roots[screen].root;
    let property = atom(&connection, b"_NET_WM_NAME");
    let outputs = matching_descendants(&connection, root, property, |name| {
        name == "Weston Compositor - screen0"
    });
    let [output] = outputs.as_slice() else {
        panic!("expected one fullscreen Weston window, found {outputs:?}");
    };
    focus_target(&connection, *output);
    (connection, root)
}

pub(in super::super) fn press_labwc_close_shortcut() {
    let (connection, root, output) = labwc_host_output();
    focus_target(&connection, output);
    send_alt_f4(&connection, root);
}

pub(in super::super) fn labwc_output() -> (RustConnection, Window) {
    let (connection, _, output) = labwc_host_output();
    (connection, output)
}

fn labwc_host_output() -> (RustConnection, Window, Window) {
    assert_eq!(
        std::env::var("OBK_BDD_LABWC_NESTED_X11").as_deref(),
        Ok("1"),
        "run the Wayland frame scenario with the private Labwc runner"
    );
    let (connection, screen) = x11rb::connect(None).expect("expected the labwc host display");
    let root = connection.setup().roots[screen].root;
    let children = connection
        .query_tree(root)
        .unwrap()
        .reply()
        .unwrap()
        .children;
    let [output] = children.as_slice() else {
        panic!("expected one private labwc output window, found {children:?}");
    };
    (connection, root, *output)
}

pub(super) fn send_alt_f4(connection: &RustConnection, root: Window) {
    let alt = keyboard_keycode(connection, 0xffe9);
    let f4 = keyboard_keycode(connection, 0xffc1);
    send_key_sequence(
        connection,
        root,
        &[
            (KEY_PRESS_EVENT, alt),
            (KEY_PRESS_EVENT, f4),
            (KEY_RELEASE_EVENT, f4),
            (KEY_RELEASE_EVENT, alt),
        ],
    );
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
    matching_descendants(connection, root, AtomEnum::WM_NAME.into(), |name| {
        name.contains("Oblikovati")
    })
}

fn named_window(connection: &RustConnection, root: Window, title: &str) -> Window {
    let windows = matching_descendants(connection, root, AtomEnum::WM_NAME.into(), |name| {
        name == title
    });
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
    title_property: Atom,
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
            if window_title(connection, child, title_property)
                .is_some_and(|title| matches_title(&title))
            {
                matches.push(child);
            }
            pending.push(child);
        }
    }
    matches
}

fn window_title(
    connection: &RustConnection,
    window: Window,
    title_property: Atom,
) -> Option<String> {
    let reply = connection
        .get_property(false, window, title_property, AtomEnum::ANY, 0, 1024)
        .ok()?;
    let property = reply.reply().ok()?;
    Some(String::from_utf8_lossy(&property.value).into_owned())
}
