// SPDX-License-Identifier: GPL-2.0-only
//! Compose the native window title from document and scene state.
use super::NativeHeadWindow;

impl NativeHeadWindow {
    pub(super) fn home_in_front(&self) -> bool {
        self.gui.as_ref().is_some_and(|gui| gui.home_in_front())
    }

    pub(super) fn refresh_title(&mut self) -> Result<(), String> {
        let title = if self.home_in_front() && self.document_title.is_some() {
            "Home - Oblikovati".to_owned()
        } else {
            self.document_scene_title()?
        };
        self.view.set_title(&title)?;
        self.accessibility_title = title;
        Ok(())
    }

    fn document_scene_title(&self) -> Result<String, String> {
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
        Ok(title)
    }
}
