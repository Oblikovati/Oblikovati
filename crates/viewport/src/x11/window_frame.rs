// SPDX-License-Identifier: GPL-2.0-only
//! ICCCM and EWMH requests for an application-owned window frame.
use super::NativeWindow;
use crate::ViewportWindowAction;
use martensite_tkservice::aspect::AspectWindow;
use x11rb::connection::Connection;
use x11rb::protocol::xproto::{
    Atom, AtomEnum, ClientMessageEvent, ConnectionExt, EventMask, PropMode,
};
use x11rb::rust_connection::RustConnection;
use x11rb::wrapper::ConnectionExt as _;

fn connection() -> Result<(RustConnection, usize), String> {
    x11rb::connect(None).map_err(|error| format!("window manager connection: {error}"))
}
fn atom(connection: &RustConnection, name: &[u8]) -> Result<Atom, String> {
    connection
        .intern_atom(false, name)
        .map_err(|error| error.to_string())?
        .reply()
        .map(|reply| reply.atom)
        .map_err(|error| error.to_string())
}
fn request(
    connection: &RustConnection,
    root: u32,
    window: u32,
    kind: Atom,
    values: [u32; 5],
) -> Result<(), String> {
    let event = ClientMessageEvent::new(32, window, kind, values);
    connection
        .send_event(
            false,
            root,
            EventMask::SUBSTRUCTURE_REDIRECT | EventMask::SUBSTRUCTURE_NOTIFY,
            event,
        )
        .map_err(|error| error.to_string())?
        .check()
        .map_err(|error| error.to_string())?;
    connection.flush().map_err(|error| error.to_string())
}
impl NativeWindow {
    pub(super) fn set_application_frame(&self) -> Result<(), String> {
        let (connection, _) = connection()?;
        let hints = atom(&connection, b"_MOTIF_WM_HINTS")?;
        // Motif decorations=0 makes the application bar the sole caption.
        connection
            .change_property32(
                PropMode::REPLACE,
                self.window.native_handle() as u32,
                hints,
                hints,
                &[2, 0, 0, 0, 0],
            )
            .map_err(|error| error.to_string())?
            .check()
            .map_err(|error| error.to_string())?;
        let mut size = x11rb::properties::WmSizeHints::new();
        size.min_size = Some((640, 480));
        size.set_normal_hints(&connection, self.window.native_handle() as u32)
            .map_err(|error| error.to_string())?
            .check()
            .map_err(|error| error.to_string())?;
        connection.flush().map_err(|error| error.to_string())
    }
    pub fn window_action(&self, action: ViewportWindowAction) -> Result<(), String> {
        let (connection, screen) = connection()?;
        let root = connection.setup().roots[screen].root;
        let window = self.window.native_handle() as u32;
        match action {
            ViewportWindowAction::Minimize => request(
                &connection,
                root,
                window,
                atom(&connection, b"WM_CHANGE_STATE")?,
                [3, 0, 0, 0, 0],
            ),
            ViewportWindowAction::ToggleMaximize => request(
                &connection,
                root,
                window,
                atom(&connection, b"_NET_WM_STATE")?,
                [
                    2,
                    atom(&connection, b"_NET_WM_STATE_MAXIMIZED_HORZ")?,
                    atom(&connection, b"_NET_WM_STATE_MAXIMIZED_VERT")?,
                    1,
                    0,
                ],
            ),
            ViewportWindowAction::Move => {
                self.release_frame_pointer();
                let pointer = connection
                    .query_pointer(root)
                    .map_err(|error| error.to_string())?
                    .reply()
                    .map_err(|error| error.to_string())?;
                request(
                    &connection,
                    root,
                    window,
                    atom(&connection, b"_NET_WM_MOVERESIZE")?,
                    [
                        pointer.root_x as i32 as u32,
                        pointer.root_y as i32 as u32,
                        8,
                        1,
                        1,
                    ],
                )
            }
        }
    }
    pub fn window_maximized(&self) -> Result<bool, String> {
        Ok(self.maximized.get())
    }
    pub(super) fn refresh_maximized(&self) -> Result<(), String> {
        let (connection, _) = connection()?;
        let state = atom(&connection, b"_NET_WM_STATE")?;
        let horizontal = atom(&connection, b"_NET_WM_STATE_MAXIMIZED_HORZ")?;
        let vertical = atom(&connection, b"_NET_WM_STATE_MAXIMIZED_VERT")?;
        let reply = connection
            .get_property(
                false,
                self.window.native_handle() as u32,
                state,
                AtomEnum::ATOM,
                0,
                64,
            )
            .map_err(|error| error.to_string())?
            .reply()
            .map_err(|error| error.to_string())?;
        let values: Vec<u32> = reply.value32().map_or_else(Vec::new, Iterator::collect);
        self.maximized
            .set(values.contains(&horizontal) && values.contains(&vertical));
        Ok(())
    }
}

// EWMH specifies eight edge directions clockwise from the top-left corner.
const RESIZE_BORDER: i32 = 6;
fn resize_edge(position: [i32; 2], extent: [i32; 2]) -> Option<(u32, u32)> {
    let [x, y] = position;
    let [width, height] = extent;
    if x < 0 || y < 0 || x >= width || y >= height {
        return None;
    }
    let (left, right, top, bottom) = (
        x < RESIZE_BORDER,
        x >= width - RESIZE_BORDER,
        y < RESIZE_BORDER,
        y >= height - RESIZE_BORDER,
    );
    match (left, right, top, bottom) {
        (true, _, true, _) => Some((0, 134)),
        (_, true, true, _) => Some((2, 136)),
        (_, true, _, true) => Some((4, 14)),
        (true, _, _, true) => Some((6, 12)),
        (_, _, true, _) => Some((1, 138)),
        (_, true, _, _) => Some((3, 96)),
        (_, _, _, true) => Some((5, 16)),
        (true, _, _, _) => Some((7, 70)),
        _ => None,
    }
}
impl NativeWindow {
    pub(super) fn handle_frame_pointer(&self, event: x11_dl::xlib::XEvent) -> Result<bool, String> {
        use x11_dl::xlib;
        if !matches!(event.get_type(), xlib::MotionNotify | xlib::ButtonPress) {
            return Ok(false);
        }
        let (position, button) = if event.get_type() == xlib::MotionNotify {
            let motion = unsafe { event.motion };
            ([motion.x, motion.y], None)
        } else {
            let press = unsafe { event.button };
            ([press.x, press.y], Some(press.button))
        };
        let (width, height) = self.window.size();
        let edge = (!self.maximized.get())
            .then(|| resize_edge(position, [width, height]))
            .flatten();
        self.set_frame_cursor(edge.map_or(0, |(_, cursor)| cursor));
        if button != Some(1) {
            return Ok(false);
        }
        let Some((direction, _)) = edge else {
            return Ok(false);
        };
        self.begin_resize(direction)?;
        Ok(true)
    }
    fn set_frame_cursor(&self, shape: u32) {
        if self.frame_cursor.replace(shape) == shape {
            return;
        }
        unsafe {
            if shape == 0 {
                (self.x11.api.XUndefineCursor)(self.pointer(), self.window.native_handle());
            } else {
                let cursor = (self.x11.api.XCreateFontCursor)(self.pointer(), shape);
                (self.x11.api.XDefineCursor)(self.pointer(), self.window.native_handle(), cursor);
                (self.x11.api.XFreeCursor)(self.pointer(), cursor);
            }
            (self.x11.api.XFlush)(self.pointer());
        }
    }
    fn release_frame_pointer(&self) {
        // The window manager cannot take over an implicit grab held by the application.
        unsafe {
            (self.x11.api.XUngrabPointer)(self.pointer(), x11_dl::xlib::CurrentTime);
            (self.x11.api.XSync)(self.pointer(), 0);
        }
    }
    fn begin_resize(&self, direction: u32) -> Result<(), String> {
        self.release_frame_pointer();
        let (connection, screen) = connection()?;
        let root = connection.setup().roots[screen].root;
        let pointer = connection
            .query_pointer(root)
            .map_err(|error| error.to_string())?
            .reply()
            .map_err(|error| error.to_string())?;
        request(
            &connection,
            root,
            self.window.native_handle() as u32,
            atom(&connection, b"_NET_WM_MOVERESIZE")?,
            [
                pointer.root_x as i32 as u32,
                pointer.root_y as i32 as u32,
                direction,
                1,
                1,
            ],
        )
    }
}
#[cfg(test)]
mod tests {
    use super::resize_edge;
    #[test]
    fn edges_and_corners_have_distinct_ewmh_directions() {
        for (point, direction) in [
            ([0, 0], 0),
            ([40, 0], 1),
            ([99, 0], 2),
            ([99, 40], 3),
            ([99, 79], 4),
            ([40, 79], 5),
            ([0, 79], 6),
            ([0, 40], 7),
        ] {
            assert_eq!(
                resize_edge(point, [100, 80]).map(|edge| edge.0),
                Some(direction)
            );
        }
        assert_eq!(resize_edge([50, 40], [100, 80]), None);
        assert_eq!(resize_edge([-1, 40], [100, 80]), None);
    }
}
