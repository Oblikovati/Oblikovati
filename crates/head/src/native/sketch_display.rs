// SPDX-License-Identifier: GPL-2.0-only
use super::NativeHeadWindow;
impl NativeHeadWindow {
    pub(super) fn update_tool_hover(
        &mut self,
        hover: Option<&oblikovati_viewport::ViewportSketchHover>,
    ) -> Result<bool, String> {
        let next = hover.cloned();
        let changed = self
            .view
            .set_sketch_hover(self.glyph_hover.as_ref().or(next.as_ref()))?;
        self.tool_hover = next;
        Ok(changed)
    }
    pub(super) fn update_sketch_scene(
        &mut self,
        update: crate::HeadSketchSceneUpdate,
    ) -> Result<(), String> {
        if self.sketch_scene.as_ref().and_then(|scene| scene.source) != update.source {
            self.view.set_sketch_hover(None)?;
            self.glyph_hover = None;
            self.tool_hover = None;
        }
        self.view.set_sketch_display(&update.sketches)?;
        self.sketch_scene = Some(update);
        self.refresh_title()
    }
    pub(super) fn update_glyph_hover(
        &mut self,
        hover: Option<&oblikovati_gui::GuiConstraintHover>,
        document: Option<u64>,
    ) -> Result<bool, String> {
        let source = self.sketch_scene.as_ref().and_then(|scene| scene.source);
        let next = hover
            .filter(|_| source.map(|value| value.0) == document)
            .map(|value| oblikovati_viewport::ViewportSketchHover {
                sketch: value.sketch,
                entities: value.related.clone(),
            });
        let changed = self
            .view
            .set_sketch_hover(next.as_ref().or(self.tool_hover.as_ref()))?;
        self.glyph_hover = next;
        Ok(changed)
    }

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
            scene.plane_declines.len() + scene.block_declines.len() + declines.len()
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
    if let Some((sketch, cause)) = scene.block_declines.first() {
        title.push_str(&format!(" - sketch {}: {cause}", sketch.0));
    }
    if let Some(decline) = declines.first() {
        title.push_str(&format!(
            " - sketch {}, entity {}: {}",
            decline.sketch.0, decline.entity.0, decline.cause
        ));
    }
}
