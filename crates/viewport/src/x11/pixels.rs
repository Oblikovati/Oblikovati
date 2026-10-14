// SPDX-License-Identifier: GPL-2.0-only
use x11rb::{connection::Connection, protocol::xproto::*, rust_connection::RustConnection};
mod layout;
pub(super) struct NativePixelReader {
    connection: RustConnection,
}
impl NativePixelReader {
    pub(super) fn open_checked_capture(display: &str) -> Result<Self, String> {
        let (connection, _) = RustConnection::connect(Some(display))
            .map_err(|cause| capture_failure(&format!("display {display:?}"), cause))?;
        Ok(Self { connection })
    }
    pub(super) fn read_drawable_rgba(&self, native_window: u64) -> Result<Vec<u8>, String> {
        let window = u32::try_from(native_window).map_err(|_| {
            format!("native capture window {native_window}: expected a 32-bit X11 resource")
        })?;
        self.require_viewable_drawable(window)?;
        let extent = self.checked_drawable_extent(window)?;
        let image = self.request_drawable_image(window, extent)?;
        let layout =
            layout::NativeImageLayout::from_setup(self.connection.setup(), &image, extent[0])?;
        layout.rgba(&image.data, extent)
    }
    fn require_viewable_drawable(&self, window: Window) -> Result<(), String> {
        let attributes = self
            .connection
            .get_window_attributes(window)
            .map_err(|cause| capture_failure(&format!("window {window} attributes"), cause))?
            .reply()
            .map_err(|cause| {
                capture_failure(&format!("window {window} attributes reply"), cause)
            })?;
        if attributes.map_state != MapState::VIEWABLE {
            return Err(format!(
                "native capture window {window}, state {:?}: expected a mapped viewport",
                attributes.map_state
            ));
        }
        Ok(())
    }
    fn checked_drawable_extent(&self, window: Window) -> Result<[u16; 2], String> {
        let geometry = self
            .connection
            .get_geometry(window)
            .map_err(|cause| capture_failure(&format!("window {window} geometry"), cause))?
            .reply()
            .map_err(|cause| capture_failure(&format!("window {window} geometry reply"), cause))?;
        if geometry.width == 0 || geometry.height == 0 {
            return Err(format!(
                "native capture window {window}, extent {}x{}: expected positive image dimensions",
                geometry.width, geometry.height
            ));
        }
        Ok([geometry.width, geometry.height])
    }
    fn request_drawable_image(
        &self,
        window: Window,
        extent: [u16; 2],
    ) -> Result<GetImageReply, String> {
        let operation = format!("window {window}, extent {extent:?}");
        self.connection
            .get_image(
                ImageFormat::Z_PIXMAP,
                window,
                0,
                0,
                extent[0],
                extent[1],
                u32::MAX,
            )
            .map_err(|cause| capture_failure(&operation, cause))?
            .reply()
            .map_err(|cause| capture_failure(&operation, cause))
    }
}
fn capture_failure(operation: &str, cause: impl std::fmt::Display) -> String {
    format!("native capture {operation}: expected a complete readable drawable: {cause}")
}
