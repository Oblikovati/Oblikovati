// SPDX-License-Identifier: GPL-2.0-only
use super::NativeHeadWindow;
impl NativeHeadWindow {
    /// Inspect the authoring source and plane failures; example: `window.sketch_scene_status()` for a health panel.
    pub fn sketch_scene_status(&self) -> Option<&crate::HeadSketchSceneUpdate> {
        self.sketch_scene.as_ref()
    }
    /// Inspect current entity failures, including failures after view resampling.
    /// Example: `window.sketch_display_declines()?` to identify unsupported entities.
    pub fn sketch_display_declines(
        &self,
    ) -> Result<&[oblikovati_viewport::ViewportSketchDecline], String> {
        self.view.sketch_display_declines()
    }
    pub(super) fn sketch_display_title(&self) -> Result<String, String> {
        let Some(scene) = self
            .sketch_scene
            .as_ref()
            .filter(|scene| scene.source.is_some())
        else {
            return Ok(String::new());
        };
        let declines = self.sketch_display_declines()?;
        let mut title = format!(
            " - {} sketches, {} sketch display failures",
            scene.sketches.len(),
            scene.plane_declines.len() + declines.len()
        );
        append_failure_names(&mut title, scene, declines);
        Ok(title)
    }
}

fn append_failure_names(
    title: &mut String,
    scene: &crate::HeadSketchSceneUpdate,
    declines: &[oblikovati_viewport::ViewportSketchDecline],
) {
    if let Some((sketch, cause)) = scene.plane_declines.first() {
        title.push_str(&format!(" - sketch {}: {cause}", sketch.0));
    }
    if let Some(decline) = declines.first() {
        title.push_str(&format!(
            " - sketch {}, entity {}: {}",
            decline.sketch.0, decline.entity.0, decline.cause
        ));
    }
}
