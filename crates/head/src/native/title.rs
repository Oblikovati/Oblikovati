// SPDX-License-Identifier: GPL-2.0-only
//! Compose the native window title from document and scene state.
use super::NativeHeadWindow;

impl NativeHeadWindow {
    pub(super) fn refresh_title(&mut self) -> Result<(), String> {
        let mut title = self
            .scene_status
            .as_ref()
            .map_or_else(|| "Oblikovati".into(), crate::HeadSceneStatus::title);
        if let Some(document) = &self.document_title {
            title = format!("{document} - {title}");
        }
        title.push_str(&self.sketch_display_title()?);
        if let Some((feature, cause)) = self.datum_declines.first() {
            title.push_str(&format!(" - work plane {}: {cause}", feature.0));
        }
        if let Some(notice) = &self.command_notice {
            title.push_str(&format!(" - {}", notice.message()));
        }
        self.view.set_title(&title)?;
        #[cfg(target_os = "linux")]
        {
            self.accessibility_title = title;
        }
        Ok(())
    }
}
