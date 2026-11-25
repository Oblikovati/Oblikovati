// SPDX-License-Identifier: GPL-2.0-only
//! Route compositor pointer input and application-owned frame gestures.
use super::*;
use crate::ViewportButton;
use smithay_client_toolkit::{
    reexports::protocols::xdg::shell::client::xdg_toplevel::ResizeEdge,
    seat::pointer::{
        AxisScroll, BTN_LEFT, BTN_MIDDLE, BTN_RIGHT, PointerEvent, PointerEventKind, PointerHandler,
    },
};
use wayland_client::protocol::wl_pointer;

impl PointerHandler for WindowState {
    fn pointer_frame(
        &mut self,
        connection: &Connection,
        _: &QueueHandle<Self>,
        pointer: &wl_pointer::WlPointer,
        events: &[PointerEvent],
    ) {
        let Some(seat) = pointer.data::<PointerData>().map(|data| data.seat()) else {
            return;
        };
        for event in events {
            if self.is_our_surface(&event.surface) {
                self.copy_pointer(connection, seat, event);
            }
        }
    }
}

impl WindowState {
    fn copy_pointer(
        &mut self,
        connection: &Connection,
        seat: &wl_seat::WlSeat,
        event: &PointerEvent,
    ) {
        let position = self.scale.surface_point(event.position);
        match event.kind {
            PointerEventKind::Enter { .. } => {
                self.input.cursor = None;
                self.point_at(connection, position);
            }
            PointerEventKind::Motion { .. } => self.point_at(connection, position),
            PointerEventKind::Leave { .. } => self.input.scroll_remainder = 0.,
            PointerEventKind::Press { serial, button, .. } => {
                self.pointer_button(seat, serial, button, true, position);
            }
            PointerEventKind::Release { serial, button, .. } => {
                self.pointer_button(seat, serial, button, false, position);
            }
            PointerEventKind::Axis { vertical, .. } => {
                let notches = scroll_steps(vertical, &mut self.input.scroll_remainder);
                if notches != 0 {
                    self.send_input(ViewportInput::Wheel { notches, position });
                }
            }
        }
    }

    fn point_at(&mut self, connection: &Connection, position: [i32; 2]) {
        let cursor = self
            .frame_edge(position)
            .map_or(CursorIcon::Default, |(_, icon)| icon);
        if self.input.cursor != Some(cursor)
            && let Some(pointer) = &self.input.pointer
        {
            let _ = pointer.set_cursor(connection, cursor);
            self.input.cursor = Some(cursor);
        }
        self.send_input(ViewportInput::Motion {
            position,
            modifiers: self.input.modifiers,
        });
    }

    fn pointer_button(
        &mut self,
        seat: &wl_seat::WlSeat,
        serial: u32,
        native: u32,
        pressed: bool,
        position: [i32; 2],
    ) {
        let Some(button) = decode_button(native) else {
            return;
        };
        if pressed {
            self.input.last_serial = Some((seat.clone(), serial));
        }
        if button == ViewportButton::Left && self.frame_press(seat, serial, pressed, position) {
            return;
        }
        self.send_input(ViewportInput::Button {
            button,
            pressed,
            position,
            modifiers: self.input.modifiers,
        });
    }

    fn frame_press(
        &mut self,
        seat: &wl_seat::WlSeat,
        serial: u32,
        pressed: bool,
        position: [i32; 2],
    ) -> bool {
        if !pressed {
            self.input.pointer_press = None;
            return std::mem::take(&mut self.input.resize_grab);
        }
        if let Some((edge, _)) = self.frame_edge(position) {
            self.shell.resize(seat, serial, edge);
            self.input.resize_grab = true;
            return true;
        }
        self.input.pointer_press = Some((seat.clone(), serial));
        false
    }

    fn frame_edge(&self, position: [i32; 2]) -> Option<(ResizeEdge, CursorIcon)> {
        if self.maximized || !self.application_frame {
            return None;
        }
        resize_edge(
            position,
            self.scale.drawable(),
            (6. * self.scale.factor()).round() as i32,
        )
    }
}

fn decode_button(button: u32) -> Option<ViewportButton> {
    match button {
        BTN_LEFT => Some(ViewportButton::Left),
        BTN_RIGHT => Some(ViewportButton::Right),
        BTN_MIDDLE => Some(ViewportButton::Middle),
        _ => None,
    }
}

fn resize_edge(
    [x, y]: [i32; 2],
    [width, height]: [u32; 2],
    border: i32,
) -> Option<(ResizeEdge, CursorIcon)> {
    use CursorIcon as Cursor;
    use ResizeEdge as Edge;
    if x < 0 || y < 0 || x >= width as i32 || y >= height as i32 {
        return None;
    }
    match (
        x < border,
        x >= width as i32 - border,
        y < border,
        y >= height as i32 - border,
    ) {
        (true, _, true, _) => Some((Edge::TopLeft, Cursor::NwResize)),
        (_, true, true, _) => Some((Edge::TopRight, Cursor::NeResize)),
        (true, _, _, true) => Some((Edge::BottomLeft, Cursor::SwResize)),
        (_, true, _, true) => Some((Edge::BottomRight, Cursor::SeResize)),
        (true, _, _, _) => Some((Edge::Left, Cursor::WResize)),
        (_, true, _, _) => Some((Edge::Right, Cursor::EResize)),
        (_, _, true, _) => Some((Edge::Top, Cursor::NResize)),
        (_, _, _, true) => Some((Edge::Bottom, Cursor::SResize)),
        _ => None,
    }
}

fn scroll_steps(scroll: AxisScroll, carry: &mut f64) -> i32 {
    const WHEEL_UNITS: f64 = 120.;
    const CONTINUOUS_STEP: f64 = 12.;
    if scroll.value120 != 0 {
        *carry -= f64::from(scroll.value120) / WHEEL_UNITS;
    } else if scroll.discrete != 0 {
        *carry -= f64::from(scroll.discrete);
    } else {
        *carry -= scroll.absolute / CONTINUOUS_STEP;
    }
    let steps = *carry as i32;
    *carry = if scroll.stop {
        0.
    } else {
        *carry - f64::from(steps)
    };
    steps
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn resize_corners_take_priority_and_inside_title_is_a_control() {
        assert_eq!(
            resize_edge([638, 2], [640, 480], 6),
            Some((ResizeEdge::TopRight, CursorIcon::NeResize))
        );
        assert_eq!(resize_edge([570, 16], [640, 480], 6), None);
    }

    #[test]
    fn high_resolution_scroll_accumulates_fractional_notches() {
        let mut carry = 0.;
        let tick = AxisScroll {
            absolute: 2.,
            value120: -60,
            ..Default::default()
        };
        assert_eq!(scroll_steps(tick, &mut carry), 0);
        assert_eq!(scroll_steps(tick, &mut carry), 1);
    }
}
