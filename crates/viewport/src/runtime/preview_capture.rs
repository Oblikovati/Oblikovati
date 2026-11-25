// SPDX-License-Identifier: GPL-2.0-only
//! Read packed RGB for tab previews while retaining typed capture failures.
use super::Viewport;
use crate::{ViewportPresentation, ViewportPreviewPixels};
use martensite_tkservice::{
    graphic3d::{Graphic3dBufferType, Graphic3dCView},
    image::ImageFormat,
};

impl Viewport {
    /// Present on X11 and borrow the accepted frame's color data for a preview.
    /// Example: `view.present_with_preview(|extent, generations, pixels| sample(extent, generations, pixels))?`.
    pub fn present_with_preview(
        &mut self,
        receive: impl FnOnce([u32; 2], [u64; 4], ViewportPreviewPixels<'_>),
    ) -> Result<bool, String> {
        if !self.draw_current_frame()? {
            return Ok(false);
        }
        if !self.read_preview_rgb()? {
            return self.present_preview_fallback(receive);
        }
        if self.observe_draw_extent()?.is_some() {
            return Ok(false);
        }
        let generations = self.generations();
        self.completed = Some(generations);
        let rgb = self.preview_rgb.data();
        receive(
            self.extent,
            generations,
            ViewportPreviewPixels::Rgb(rgb.data.data().as_chunks::<3>().0),
        );
        Ok(true)
    }

    fn read_preview_rgb(&self) -> Result<bool, String> {
        let [width, height] = self.extent.map(|side| side as usize);
        let image = &self.preview_rgb;
        if (image.width(), image.height()) != (width, height) {
            if !image.init_trash(ImageFormat::Rgb, width, height, 0) {
                return Ok(false);
            }
            image.set_top_down(true);
        }
        Ok(self
            .live()?
            .renderer
            .buffer_dump(image, &Graphic3dBufferType::Rgb))
    }

    fn present_preview_fallback(
        &mut self,
        receive: impl FnOnce([u32; 2], [u64; 4], ViewportPreviewPixels<'_>),
    ) -> Result<bool, String> {
        // The RGB buffer API returns a boolean. Preserve the typed capture path on refusal.
        let ViewportPresentation::Completed(capture) = self.capture_completed_surface()? else {
            return Ok(false);
        };
        receive(
            capture.extent,
            self.generations(),
            ViewportPreviewPixels::Rgba(capture.rgba.as_chunks::<4>().0),
        );
        Ok(true)
    }
}
