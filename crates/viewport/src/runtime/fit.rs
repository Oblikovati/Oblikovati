// SPDX-License-Identifier: GPL-2.0-only
use super::*;

impl Viewport {
    /// Recenter the current projection within the desktop layout while retaining the view.
    /// Example: `viewport.set_document_projection([1000, 720], [200, 120, 900, 680])?`.
    pub fn set_document_projection(
        &mut self,
        extent: [u32; 2],
        region: [u32; 4],
    ) -> Result<bool, String> {
        self.live()?;
        self.refresh_extent()?;
        // A native resize can supersede the GUI frame while its draws are being prepared.
        if self.extent != extent {
            return Ok(false);
        }
        let framing = crate::fit_region::fit_region(extent, region)?;
        if self.layout_framing == Some(framing) {
            return Ok(false);
        }
        let next = next_generation(self.camera)?;
        let changed = self.live()?.set_layout_projection(framing);
        self.layout_framing = Some(framing);
        if changed {
            self.camera = next;
            self.completed = None;
            self.projection_cache.get_mut().clear();
        }
        Ok(changed)
    }

    pub(crate) fn current_layout_framing(&self) -> Option<crate::fit_region::FitFraming> {
        self.layout_framing
            .filter(|layout| layout.extent == self.extent)
    }

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
