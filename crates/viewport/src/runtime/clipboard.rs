// SPDX-License-Identifier: GPL-2.0-only
use super::*;
impl Viewport {
    /// Join injected clipboard readiness with window and worker events without rendering.
    /// Example: `view.wait_events_with_clipboard(&wake, &clipboard, None)?` while idle.
    pub fn wait_events_with_clipboard(
        &self,
        wake: &crate::ViewportWake,
        clipboard: &crate::NativeClipboardEndpoint,
        timeout: Option<std::time::Duration>,
    ) -> Result<crate::ViewportReady, String> {
        self.live()?;
        match self.target.as_ref().expect("live target") {
            ViewTarget::Native(native) => {
                native.wait_native_sources(wake, timeout, Some(clipboard))
            }
            ViewTarget::Offscreen(_) => Err(format!(
                "viewport {}: expected a native event source",
                self.id
            )),
        }
    }
}
