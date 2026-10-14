// SPDX-License-Identifier: GPL-2.0-only
use super::Viewport;
use crate::{GuiFrame, GuiFrameCompiler};
impl Viewport {
    /// Observe the current target before building a GUI packet; resize invalidates the previous capture.
    /// Example: `let extent = view.drawable_extent()?;` for the next frame request.
    pub fn drawable_extent(&mut self) -> Result<[u32; 2], String> {
        self.live()?;
        self.refresh_extent()?;
        Ok(self.extent)
    }

    /// Atomically replace the GUI on this view; the kernel owns the compiled packet through presentation.
    /// Example: `view.submit_gui_frame(&compiler, &output.draws)?; view.redraw()?;`.
    pub fn submit_gui_frame(
        &mut self,
        compiler: &GuiFrameCompiler,
        frame: &GuiFrame,
    ) -> Result<(), String> {
        self.update_overlay(|render| {
            let layer = compiler.layer(frame)?;
            render
                .renderer
                .attach_gui_layer(frame.extent, layer)
                .map_err(|cause| cause.to_string())?;
            Ok(true)
        })
        .map(|_| ())
    }
    /// Remove GUI content while retaining the model and camera; redraw is required before capture.
    /// Example: `view.clear_gui_frame()?; view.redraw()?;`.
    pub fn clear_gui_frame(&mut self) -> Result<(), String> {
        self.update_overlay(|render| {
            render
                .renderer
                .clear_gui_layer()
                .map_err(|cause| cause.to_string())?;
            Ok(true)
        })
        .map(|_| ())
    }
}
