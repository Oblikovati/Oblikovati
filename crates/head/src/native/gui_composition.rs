// SPDX-License-Identifier: GPL-2.0-only
use super::NativeHeadWindow;
use oblikovati_viewport::{GuiFrame, GuiFrameCompiler};
impl NativeHeadWindow {
    /// Read the actual target for the next GUI frame; example: `request.extent = window.drawable_extent()?;`.
    pub fn drawable_extent(&mut self) -> Result<[u32; 2], String> {
        self.view.drawable_extent()
    }

    /// Attach owned GUI draws to the existing document view; no UI context is retained by the window.
    /// Example: `window.submit_gui_frame(&compiler, &output.draws)?; window.present()?;`.
    pub fn submit_gui_frame(
        &mut self,
        compiler: &GuiFrameCompiler,
        frame: &GuiFrame,
    ) -> Result<(), String> {
        self.view.submit_gui_frame(compiler, frame)
    }
    /// Remove the GUI without replacing document presentation; example: `window.clear_gui_frame()?;`.
    pub fn clear_gui_frame(&mut self) -> Result<(), String> {
        self.view.clear_gui_frame()
    }
}
