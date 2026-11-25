// SPDX-License-Identifier: GPL-2.0-only
//! Keep surface-local input and drawable pixels in the same coordinate system.
use super::shell::WindowState;
use crate::ViewportEvent;
use smithay_client_toolkit::{reexports::protocols, shell::WaylandSurface};
use wayland_client::{
    Connection, Dispatch, Proxy, QueueHandle, globals::GlobalList, protocol::wl_surface::WlSurface,
};

use protocols::wp::{
    fractional_scale::v1::client::{wp_fractional_scale_manager_v1, wp_fractional_scale_v1},
    viewporter::client::{wp_viewport, wp_viewporter},
};

const SCALE_UNIT: u32 = 120;

pub(super) struct DrawableScale {
    pub(super) logical: [u32; 2],
    integer: u32,
    preferred: Option<u32>,
    fractional: Option<FractionalSurface>,
}

struct FractionalSurface {
    viewport: wp_viewport::WpViewport,
    scale: wp_fractional_scale_v1::WpFractionalScaleV1,
}

impl Drop for FractionalSurface {
    fn drop(&mut self) {
        self.scale.destroy();
        self.viewport.destroy();
    }
}

impl DrawableScale {
    pub(super) fn new(
        globals: &GlobalList,
        queue: &QueueHandle<WindowState>,
        surface: &WlSurface,
        logical: [u32; 2],
    ) -> Self {
        Self {
            logical,
            integer: 1,
            preferred: None,
            fractional: fractional_surface(globals, queue, surface),
        }
    }
    pub(super) fn factor(&self) -> f32 {
        self.numerator() as f32 / SCALE_UNIT as f32
    }
    fn numerator(&self) -> u32 {
        self.preferred
            .unwrap_or(self.integer.saturating_mul(SCALE_UNIT))
    }
    pub(super) fn drawable(&self) -> [u32; 2] {
        self.logical
            .map(|side| scale_dimension(side, self.numerator()))
    }
    pub(super) fn to_logical(&self, extent: [u32; 2]) -> [u32; 2] {
        let factor = f64::from(self.numerator()) / f64::from(SCALE_UNIT);
        extent.map(|side| (f64::from(side) / factor).round().max(1.) as u32)
    }
    pub(super) fn surface_point(&self, position: (f64, f64)) -> [i32; 2] {
        let [width, height] = self.drawable();
        let [logical_width, logical_height] = self.logical;
        [
            (position.0 * f64::from(width) / f64::from(logical_width)).round() as i32,
            (position.1 * f64::from(height) / f64::from(logical_height)).round() as i32,
        ]
    }
    pub(super) fn apply(&self, surface: &WlSurface) {
        if let Some(fractional) = self
            .fractional
            .as_ref()
            .filter(|_| self.preferred.is_some())
        {
            surface.set_buffer_scale(1);
            let [width, height] = self.logical;
            fractional
                .viewport
                .set_destination(width as i32, height as i32);
        } else if surface.version() >= 3 {
            surface.set_buffer_scale(self.integer as i32);
        }
    }
}

fn scale_dimension(side: u32, numerator: u32) -> u32 {
    let divisor = u64::from(SCALE_UNIT);
    let pixels = (u64::from(side) * u64::from(numerator) + divisor / 2) / divisor;
    pixels.clamp(1, i32::MAX as u64) as u32
}

fn fractional_surface(
    globals: &GlobalList,
    queue: &QueueHandle<WindowState>,
    surface: &WlSurface,
) -> Option<FractionalSurface> {
    let viewporter: wp_viewporter::WpViewporter = globals.bind(queue, 1..=1, ()).ok()?;
    let manager: wp_fractional_scale_manager_v1::WpFractionalScaleManagerV1 =
        globals.bind(queue, 1..=1, ()).ok()?;
    let viewport = viewporter.get_viewport(surface, queue, ());
    let scale = manager.get_fractional_scale(surface, queue, ());
    manager.destroy();
    viewporter.destroy();
    Some(FractionalSurface { viewport, scale })
}

impl WindowState {
    pub(super) fn change_integer_scale(&mut self, factor: i32) {
        let previous = self.scale.numerator();
        self.scale.integer = factor.max(1) as u32;
        self.publish_scale(previous);
    }

    fn publish_scale(&mut self, previous: u32) {
        if previous == self.scale.numerator() {
            return;
        }
        self.scale.apply(self.shell.wl_surface());
        self.pending
            .push_back(ViewportEvent::Resized(self.scale.drawable()));
    }
}

impl Dispatch<wp_fractional_scale_v1::WpFractionalScaleV1, ()> for WindowState {
    fn event(
        state: &mut Self,
        _: &wp_fractional_scale_v1::WpFractionalScaleV1,
        event: wp_fractional_scale_v1::Event,
        _: &(),
        _: &Connection,
        _: &QueueHandle<Self>,
    ) {
        if let wp_fractional_scale_v1::Event::PreferredScale { scale } = event
            && scale > 0
        {
            let previous = state.scale.numerator();
            state.scale.preferred = Some(scale);
            state.publish_scale(previous);
        }
    }
}

wayland_client::delegate_noop!(WindowState: ignore wp_fractional_scale_manager_v1::WpFractionalScaleManagerV1);
wayland_client::delegate_noop!(WindowState: ignore wp_viewporter::WpViewporter);
wayland_client::delegate_noop!(WindowState: ignore wp_viewport::WpViewport);

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn fractional_scale_rounds_drawable_pixels_on_half_boundaries() {
        assert_eq!(scale_dimension(641, 180), 962);
        assert_eq!(scale_dimension(479, 150), 599);
        assert_eq!(scale_dimension(u32::MAX, u32::MAX), i32::MAX as u32);
    }
}
