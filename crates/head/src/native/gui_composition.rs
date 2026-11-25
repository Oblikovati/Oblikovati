// SPDX-License-Identifier: GPL-2.0-only
use super::NativeHeadWindow;
use oblikovati_viewport::{GuiFrame, GuiFrameCompiler};
impl NativeHeadWindow {
    pub(super) fn publish_isometric_preview(&mut self, document: u64) -> Result<bool, String> {
        match self.isometric_preview_image(document) {
            Ok(image) => {
                if let Some(gui) = &mut self.gui {
                    gui.receive_isometric_preview(document, image);
                }
            }
            Err(cause) => {
                if let Some(gui) = &mut self.gui {
                    gui.reject_isometric_preview(document, cause.clone());
                }
                crate::HeadWindow::show_command_notice(
                    self,
                    crate::DesktopCommandNotice::PreviewFailed { cause },
                )?;
            }
        }
        Ok(true)
    }
    pub(super) fn isometric_preview_image(
        &self,
        document: u64,
    ) -> Result<oblikovati_gui::GuiPreviewImage, String> {
        let source = self.scene_status.as_ref().and_then(|status| status.source);
        if source.map(|(id, _)| id) != Some(document) {
            return Err(format!(
                "isometric preview document {document}, scene {source:?}: expected the active document scene"
            ));
        }
        let scene = self.current_scene.as_ref().ok_or_else(|| {
            format!("isometric preview document {document}: expected a submitted scene")
        })?;
        let mut preview = self.runtime.open_offscreen([256, 144])?;
        preview.submit(scene)?;
        if let Some(background) = self.preview_background {
            preview.set_background(background)?;
        }
        preview.navigate(oblikovati_viewport::ViewportNavigation::Home { margin: 0.1 })?;
        let capture = preview.redraw()?;
        Ok(downsampled_preview(&capture))
    }
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
    /// Picture the viewport without the GUI: the dialog asking for it must not appear in it.
    /// The GUI returns on the next published frame, so a user sees it blink once.
    pub(super) fn scene_preview_image(
        &mut self,
    ) -> Result<oblikovati_gui::GuiPreviewImage, String> {
        self.clear_gui_frame()?;
        loop {
            match self.view.present()? {
                oblikovati_viewport::ViewportPresentation::Completed(capture) => {
                    return Ok(downsampled_preview(&capture));
                }
                oblikovati_viewport::ViewportPresentation::SurfaceChanged(_) => {
                    self.view.poll_events(PREVIEW_EVENT_LIMIT)?;
                }
            }
        }
    }
    pub(super) fn publish_window_preview(
        &mut self,
        state: &oblikovati_gui::GuiCommandState,
    ) -> Result<bool, String> {
        let image = if state.document_type == Some(oblikovati_api::types::DOCUMENT_DRAWING) {
            self.drawing_preview_image(state)?
        } else {
            self.scene_preview_image()?
        };
        if let Some(gui) = &mut self.gui {
            gui.receive_window_preview(image);
        }
        Ok(true)
    }

    fn drawing_preview_image(
        &self,
        state: &oblikovati_gui::GuiCommandState,
    ) -> Result<oblikovati_gui::GuiPreviewImage, String> {
        let extent = [256, 144];
        let frame = oblikovati_gui::drawing_preview_frame(extent, state)?;
        let compiler = GuiFrameCompiler::with_resources(&frame.resources)?;
        let mut preview = self.runtime.open_offscreen(extent)?;
        preview.submit_gui_frame(&compiler, &frame)?;
        Ok(downsampled_preview(&preview.redraw()?))
    }
    pub fn clear_gui_frame(&mut self) -> Result<(), String> {
        self.view.clear_gui_frame()?;
        if let Some(gui) = &mut self.gui {
            gui.require_publication();
        }
        Ok(())
    }
}

const PREVIEW_EVENT_LIMIT: usize = 64;
const PREVIEW_CELLS: [u32; 2] = [64, 36];
/// Nearest cell centres: a preview shows the window's look, not its detail.
fn downsampled_preview(
    capture: &oblikovati_viewport::ViewportCapture,
) -> oblikovati_gui::GuiPreviewImage {
    let [columns, rows] = PREVIEW_CELLS;
    let width = u64::from(capture.extent[0]);
    let height = u64::from(capture.extent[1]);
    let mut rgb = Vec::with_capacity((columns * rows) as usize);
    for row in 0..u64::from(rows) {
        for column in 0..u64::from(columns) {
            let x = (2 * column + 1) * width / (2 * u64::from(columns));
            let y = (2 * row + 1) * height / (2 * u64::from(rows));
            let at = ((y * width + x) * 4) as usize;
            rgb.push([capture.rgba[at], capture.rgba[at + 1], capture.rgba[at + 2]]);
        }
    }
    oblikovati_gui::GuiPreviewImage {
        extent: PREVIEW_CELLS,
        rgb: std::sync::Arc::new(rgb),
    }
}
#[cfg(test)]
#[path = "../../tests/native_preview/downsample.rs"]
mod preview_tests;
