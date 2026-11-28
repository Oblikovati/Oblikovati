// SPDX-License-Identifier: GPL-2.0-only
use super::*;
impl NativeHeadWindow {
    /// Write the current composed image using the window's capture destination.
    /// Example: `window.capture_window_png(path)?`.
    pub fn capture_window_png(
        &mut self,
        path: &std::path::Path,
    ) -> Result<oblikovati_viewport::ViewportCapture, String> {
        let capture = self.view.capture()?;
        self.capture_files.write_capture(path, &capture)?;
        Ok(capture)
    }
}
impl NativeHeadWindow {
    pub(super) fn present_native_frame(&mut self) -> Result<HeadPresentation, String> {
        #[cfg(target_arch = "wasm32")]
        {
            let completed = self.view.present_frame()?;
            return self.finish_native_presentation(completed);
        }
        #[cfg(target_os = "linux")]
        if self.capture_requests.is_empty() {
            let last = self.view.completed_generations();
            let completed = match self.gui.as_mut().filter(|gui| gui.needs_frame_pixels(last)) {
                Some(gui) => self
                    .view
                    .present_with_preview(|extent, generations, pixels| {
                        gui.note_window_preview_pixels(extent, generations, pixels);
                    }),
                None => self.view.present_without_capture(),
            }?;
            return self.finish_native_presentation(completed);
        }
        #[cfg(not(target_arch = "wasm32"))]
        {
            let ViewportPresentation::Completed(capture) = self.view.present()? else {
                return self.finish_native_presentation(false);
            };
            let completed = self.finish_native_presentation(true)?;
            if let Some(gui) = &mut self.gui {
                gui.note_presented_frame(&capture);
            }
            self.publish_window_captures(&capture)?;
            Ok(completed)
        }
    }

    fn finish_native_presentation(&mut self, completed: bool) -> Result<HeadPresentation, String> {
        self.refresh_title()?;
        if completed {
            self.note_orientation_presented();
            return Ok(HeadPresentation::Completed);
        }
        if let Some(gui) = &mut self.gui {
            gui.require_publication();
        }
        Ok(HeadPresentation::SurfaceChanged)
    }

    fn publish_window_captures(
        &mut self,
        capture: &oblikovati_viewport::ViewportCapture,
    ) -> Result<(), String> {
        for request in std::mem::take(&mut self.capture_requests) {
            let outcome = matching_capture_extent(&request, capture)
                .and_then(|()| self.capture_files.write_capture(&request.path, capture));
            if let Err(cause) = outcome {
                self.show_command_notice(crate::DesktopCommandNotice::WindowCaptureFailed {
                    path: request.path,
                    cause,
                })?;
            }
        }
        Ok(())
    }
}
fn matching_capture_extent(
    request: &crate::WindowCaptureJob,
    capture: &oblikovati_viewport::ViewportCapture,
) -> Result<(), String> {
    if request.extent != capture.extent {
        return Err(format!(
            "window capture path {:?}, extent {:?}: expected the requested drawable size {:?}; request a new capture after resize",
            request.path, capture.extent, request.extent
        ));
    }
    Ok(())
}

#[cfg(test)]
#[path = "../../tests/native_window_capture.rs"]
mod tests;
