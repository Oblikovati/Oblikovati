// SPDX-License-Identifier: GPL-2.0-only
use super::*;

impl Viewport {
    /// Read the displayed Fit bounds for camera framing only, not model decisions.
    /// Example: `view.fit_visible_bounds()?` before centering a view on its selection.
    pub fn fit_visible_bounds(&self) -> Result<[[f64; 3]; 2], String> {
        self.live()?.fit_visible_bounds()
    }

    /// Read visible model and sketch bounds for presentation; an empty scene returns `None`.
    /// Example: `let bounds = view.visible_presentation_bounds()?` for a display overlay.
    pub fn visible_presentation_bounds(&self) -> Result<Option<[[f64; 3]; 2]>, String> {
        self.live()?.visible_presentation_bounds()
    }

    /// Fit or orient displayed geometry inside an unobstructed framebuffer rectangle.
    /// Example: `view.navigate_in_region(ViewportNavigation::Home { margin: 0.1 }, [200, 100, 800, 600])?`.
    pub fn navigate_in_region(
        &mut self,
        command: crate::ViewportNavigation,
        region: [u32; 4],
    ) -> Result<bool, String> {
        let framing = self.prepare_fit_region(command, region)?;
        let next = next_generation(self.camera)?;
        if !self.apply_camera_navigation(command, Some(framing))? {
            return Ok(false);
        }
        self.camera = next;
        self.completed = None;
        Ok(true)
    }
    fn prepare_fit_region(
        &mut self,
        command: crate::ViewportNavigation,
        region: [u32; 4],
    ) -> Result<crate::fit_region::FitFraming, String> {
        self.live()?;
        self.refresh_extent()?;
        command.validate(self.extent)?;
        if !matches!(
            command,
            crate::ViewportNavigation::Fit { .. } | crate::ViewportNavigation::Home { .. }
        ) {
            return Err(format!(
                "navigation {command:?}: expected Fit or Home for region {region:?}"
            ));
        }
        crate::fit_region::fit_region(self.extent, region)
    }
}
