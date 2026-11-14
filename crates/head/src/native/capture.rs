// SPDX-License-Identifier: GPL-2.0-only
use super::*;
impl NativeHeadWindow {
    pub(super) fn publish_window_captures(
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
