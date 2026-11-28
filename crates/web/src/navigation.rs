// SPDX-License-Identifier: GPL-2.0-only
//! Browser pointer/wheel camera interaction controller and standard view presets.

use crate::WebViewOrientation;
use oblikovati_viewport::{
    ViewportButton, ViewportCamera, ViewportEvent, ViewportInput, ViewportKey, ViewportNavigation,
};

const ORBIT_RADIANS_PER_PIXEL: f64 = 0.008;
const ZOOM_STEP_FACTOR: f64 = 1.15;

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum DragMode {
    Orbit,
    Pan,
}

/// Stateful browser pointer/wheel gesture translator producing `ViewportNavigation` steps.
///
/// ```
/// use oblikovati_web::WebNavigationController;
/// let mut nav = WebNavigationController::default();
/// assert!(!nav.is_dragging());
/// ```
#[derive(Default)]
pub struct WebNavigationController {
    drag: Option<(DragMode, [i32; 2])>,
}

impl WebNavigationController {
    /// Report whether a pointer drag gesture is currently active.
    /// Example: `if nav.is_dragging() { /* active */ }`.
    pub fn is_dragging(&self) -> bool {
        self.drag.is_some()
    }

    /// Translate one viewport event into an optional camera navigation action.
    /// Example: `if let Some(step) = nav.consume(event) { viewport.navigate(step)?; }`.
    pub fn consume(&mut self, event: ViewportEvent) -> Option<ViewportNavigation> {
        let ViewportEvent::Input(input) = event else {
            if matches!(event, ViewportEvent::Focus(false)) {
                self.drag = None;
            }
            return None;
        };
        match input {
            ViewportInput::Button {
                button,
                pressed,
                position,
                modifiers,
            } => {
                if !pressed {
                    self.drag = None;
                    return None;
                }
                let mode = match button {
                    ViewportButton::Left if !modifiers.shift => DragMode::Orbit,
                    ViewportButton::Middle if modifiers.shift => DragMode::Orbit,
                    ViewportButton::Middle | ViewportButton::Right | ViewportButton::Left => {
                        DragMode::Pan
                    }
                };
                self.drag = Some((mode, position));
                None
            }
            ViewportInput::Motion { position, .. } => {
                let (mode, last) = self.drag?;
                self.drag = Some((mode, position));
                let dx = position[0].saturating_sub(last[0]);
                let dy = position[1].saturating_sub(last[1]);
                if dx == 0 && dy == 0 {
                    return None;
                }
                match mode {
                    DragMode::Orbit => Some(ViewportNavigation::Orbit {
                        yaw: -f64::from(dx) * ORBIT_RADIANS_PER_PIXEL,
                        pitch: -f64::from(dy) * ORBIT_RADIANS_PER_PIXEL,
                    }),
                    DragMode::Pan => Some(ViewportNavigation::Pan([dx, dy])),
                }
            }
            ViewportInput::Wheel { notches, position } => {
                if notches == 0 {
                    return None;
                }
                let factor = ZOOM_STEP_FACTOR.powi(notches);
                Some(ViewportNavigation::Zoom {
                    factor,
                    anchor: Some(position),
                })
            }
            ViewportInput::PhysicalKey {
                command: Some(ViewportKey::Home),
                pressed: true,
                ..
            } => Some(ViewportNavigation::Home { margin: 0.1 }),
            _ => None,
        }
    }
}

/// Build an oriented camera around the current target at the current eye distance.
/// Example: `let front = oriented_camera(current, WebViewOrientation::Front);`.
pub fn oriented_camera(current: ViewportCamera, view: WebViewOrientation) -> ViewportCamera {
    let delta: [f64; 3] = std::array::from_fn(|axis| current.eye[axis] - current.target[axis]);
    let distance = delta
        .iter()
        .map(|value| value * value)
        .sum::<f64>()
        .sqrt()
        .max(1.0);
    let (direction, up) = match view {
        WebViewOrientation::Iso => {
            let s = 1.0 / 3.0_f64.sqrt();
            ([s, s, s], [0.0, 1.0, 0.0])
        }
        WebViewOrientation::Front => ([0.0, 0.0, 1.0], [0.0, 1.0, 0.0]),
        WebViewOrientation::Back => ([0.0, 0.0, -1.0], [0.0, 1.0, 0.0]),
        WebViewOrientation::Right => ([1.0, 0.0, 0.0], [0.0, 1.0, 0.0]),
        WebViewOrientation::Left => ([-1.0, 0.0, 0.0], [0.0, 1.0, 0.0]),
        WebViewOrientation::Top => ([0.0, 1.0, 0.0], [0.0, 0.0, -1.0]),
        WebViewOrientation::Bottom => ([0.0, -1.0, 0.0], [0.0, 0.0, 1.0]),
    };
    let eye = std::array::from_fn(|axis| current.target[axis] + direction[axis] * distance);
    ViewportCamera { eye, up, ..current }
}

#[cfg(test)]
mod tests {
    use super::*;
    use oblikovati_viewport::{ViewportModifiers, ViewportPhysicalKey};

    #[test]
    fn drag_wheel_and_orientation_presets_translate_cleanly() {
        let mut nav = WebNavigationController::default();
        assert!(
            nav.consume(ViewportEvent::Input(ViewportInput::Button {
                button: ViewportButton::Left,
                pressed: true,
                position: [100, 100],
                modifiers: ViewportModifiers::default(),
            }))
            .is_none()
        );
        assert!(nav.is_dragging());
        let orbit = nav
            .consume(ViewportEvent::Input(ViewportInput::Motion {
                position: [110, 90],
                modifiers: ViewportModifiers::default(),
            }))
            .unwrap();
        assert!(matches!(orbit, ViewportNavigation::Orbit { .. }));

        nav.consume(ViewportEvent::Focus(false));
        assert!(!nav.is_dragging());

        nav.consume(ViewportEvent::Input(ViewportInput::Button {
            button: ViewportButton::Right,
            pressed: true,
            position: [50, 50],
            modifiers: ViewportModifiers::default(),
        }));
        let pan = nav
            .consume(ViewportEvent::Input(ViewportInput::Motion {
                position: [65, 55],
                modifiers: ViewportModifiers::default(),
            }))
            .unwrap();
        assert_eq!(pan, ViewportNavigation::Pan([15, 5]));

        let zoom = nav
            .consume(ViewportEvent::Input(ViewportInput::Wheel {
                notches: 1,
                position: [320, 240],
            }))
            .unwrap();
        assert!(matches!(zoom, ViewportNavigation::Zoom { .. }));

        let home = nav
            .consume(ViewportEvent::Input(ViewportInput::PhysicalKey {
                key: ViewportPhysicalKey::Home,
                layout_key: ViewportPhysicalKey::Home,
                command: Some(ViewportKey::Home),
                pressed: true,
                modifiers: ViewportModifiers::default(),
            }))
            .unwrap();
        assert_eq!(home, ViewportNavigation::Home { margin: 0.1 });

        let base = ViewportCamera::default();
        for view in [
            WebViewOrientation::Iso,
            WebViewOrientation::Front,
            WebViewOrientation::Back,
            WebViewOrientation::Left,
            WebViewOrientation::Right,
            WebViewOrientation::Top,
            WebViewOrientation::Bottom,
        ] {
            let next = oriented_camera(base, view);
            assert_eq!(next.target, base.target);
        }
    }
}
