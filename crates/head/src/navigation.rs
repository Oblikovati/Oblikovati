// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_viewport::{
    ViewportButton, ViewportEvent, ViewportInput, ViewportKey, ViewportModifiers,
    ViewportNavigation,
};
#[derive(Clone, Copy)]
enum DragGesture {
    Pan,
    Zoom,
    Orbit,
}
#[derive(Clone, Copy)]
struct NavigationDrag {
    button: ViewportButton,
    last: [i32; 2],
}
#[derive(Default)]
pub(crate) struct DesktopNavigation {
    held_modes: [bool; 3],
    drag: Option<NavigationDrag>,
}
impl DesktopNavigation {
    pub(crate) fn command(
        &mut self,
        event: &ViewportEvent,
    ) -> Result<Option<ViewportNavigation>, String> {
        match event {
            ViewportEvent::Input(input) => self.input_command(*input),
            ViewportEvent::Focus(false)
            | ViewportEvent::Mapped(false)
            | ViewportEvent::CloseRequested => {
                self.clear_gesture();
                Ok(None)
            }
            _ => Ok(None),
        }
    }
    fn input_command(
        &mut self,
        input: ViewportInput,
    ) -> Result<Option<ViewportNavigation>, String> {
        match input {
            ViewportInput::Motion {
                position,
                modifiers,
            } => return self.motion_command(position, modifiers),
            ViewportInput::Wheel { notches, position } => {
                return Ok(Some(wheel_navigation(notches, position)));
            }
            ViewportInput::Button {
                button,
                pressed,
                position,
                ..
            } => self.button_command(button, pressed, position),
            ViewportInput::Key {
                key: ViewportKey::Home,
                pressed: true,
            } => {
                self.clear_gesture();
                return Ok(Some(ViewportNavigation::Home { margin: 0.1 }));
            }
            ViewportInput::Key { key, pressed } => self.key_mode(key, pressed),
        }
        Ok(None)
    }
    fn clear_gesture(&mut self) {
        self.drag = None;
        self.held_modes = [false; 3];
    }
    fn key_mode(&mut self, key: ViewportKey, pressed: bool) {
        match key {
            ViewportKey::Pan => self.held_modes[0] = pressed,
            ViewportKey::Zoom => self.held_modes[1] = pressed,
            ViewportKey::Orbit => self.held_modes[2] = pressed,
            ViewportKey::Cancel if pressed => self.clear_gesture(),
            _ => {}
        }
    }
    fn modal_gesture(&self) -> Option<DragGesture> {
        match self.held_modes {
            [true, _, _] => Some(DragGesture::Pan),
            [false, true, _] => Some(DragGesture::Zoom),
            [false, false, true] => Some(DragGesture::Orbit),
            _ => None,
        }
    }
    fn button_command(&mut self, button: ViewportButton, pressed: bool, position: [i32; 2]) {
        if !pressed {
            if self.drag.is_some_and(|drag| drag.button == button) {
                self.drag = None;
            }
            return;
        }
        let admitted = button == ViewportButton::Middle
            || (button == ViewportButton::Left && self.modal_gesture().is_some());
        if admitted {
            self.drag = Some(NavigationDrag {
                button,
                last: position,
            });
        }
    }
    fn motion_command(
        &mut self,
        position: [i32; 2],
        modifiers: ViewportModifiers,
    ) -> Result<Option<ViewportNavigation>, String> {
        let Some(drag) = self.drag else {
            return Ok(None);
        };
        let delta = pointer_delta(drag.last, position)?;
        self.drag = Some(NavigationDrag {
            last: position,
            ..drag
        });
        let gesture = match drag.button {
            ViewportButton::Left => self.modal_gesture(),
            ViewportButton::Middle if modifiers.shift => Some(DragGesture::Orbit),
            ViewportButton::Middle => Some(DragGesture::Pan),
            ViewportButton::Right => None,
        };
        Ok(gesture.map(|gesture| drag_navigation(gesture, delta)))
    }
}
fn wheel_navigation(notches: i32, position: [i32; 2]) -> ViewportNavigation {
    ViewportNavigation::Zoom {
        factor: 0.9_f64.powi(notches),
        anchor: Some(position),
    }
}
fn pointer_delta(start: [i32; 2], end: [i32; 2]) -> Result<[i32; 2], String> {
    let delta = std::array::from_fn::<_, 2, _>(|axis| end[axis].checked_sub(start[axis]));
    match delta {
        [Some(x), Some(y)] => Ok([x, y]),
        _ => Err(format!(
            "navigation pointer {start:?}->{end:?}: expected representable signed 32-bit pixel deltas"
        )),
    }
}
fn drag_navigation(gesture: DragGesture, [x, y]: [i32; 2]) -> ViewportNavigation {
    // These sensitivities retain Go head/ui/navigate.go at the fixed migration revision.
    match gesture {
        DragGesture::Pan => ViewportNavigation::Pan([x, y]),
        DragGesture::Orbit => ViewportNavigation::Orbit {
            yaw: -f64::from(x) * 0.01,
            pitch: -f64::from(y) * 0.01,
        },
        DragGesture::Zoom => ViewportNavigation::Zoom {
            factor: 1.01_f64.powi(y),
            anchor: None,
        },
    }
}
#[cfg(test)]
#[path = "../tests/navigation/controller.rs"]
mod tests;
