// SPDX-License-Identifier: GPL-2.0-only
use super::Viewport;
use crate::{GuiFrame, GuiFrameCompiler, ViewportGuiSubmission};
use martensite_vulkan::RenderError;
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
        match self.submit_gui_frame_checked(compiler, frame)? {
            ViewportGuiSubmission::Accepted => Ok(()),
            ViewportGuiSubmission::SurfaceChanged(extent) => Err(format!(
                "viewport {} GUI target changed to {extent:?}; expected a fresh packet for the current surface",
                self.id
            )),
        }
    }
    /// Keep target changes distinct from attachment faults; example: `view.submit_gui_frame_checked(&compiler, &frame)?`.
    pub fn submit_gui_frame_checked(
        &mut self,
        compiler: &GuiFrameCompiler,
        frame: &GuiFrame,
    ) -> Result<ViewportGuiSubmission, String> {
        if let Some(error) = self.stage_gui_layer(compiler, frame)? {
            self.checked_gui_result::<()>(Err(error), "attach GUI")?;
            return Ok(ViewportGuiSubmission::SurfaceChanged(self.extent));
        }
        self.gui_surface = Some(self.surface);
        Ok(ViewportGuiSubmission::Accepted)
    }
    fn stage_gui_layer(
        &mut self,
        compiler: &GuiFrameCompiler,
        frame: &GuiFrame,
    ) -> Result<Option<RenderError>, String> {
        let mut fault: Option<RenderError> = None;
        self.update_overlay(|render| {
            let layer = compiler.layer(frame)?;
            match render.renderer.attach_gui_layer(frame.extent, layer) {
                Ok(()) => Ok(true),
                Err(error) => {
                    fault = Some(error);
                    Ok(false)
                }
            }
        })?;
        Ok(fault)
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
        })?;
        self.gui_surface = None;
        Ok(())
    }
}
