// SPDX-License-Identifier: GPL-2.0-only
use super::Viewport;
use crate::{ViewportNavigation, renderer::SavedViewCamera};
// Retain Go app/view_history.go's limit without saving native handles or old pixel extents.
const VIEW_HISTORY_LIMIT: usize = 64;
impl Viewport {
    pub(super) fn apply_camera_navigation(
        &mut self,
        command: ViewportNavigation,
    ) -> Result<bool, String> {
        let renderer = self.live()?;
        let previous = renderer.saved_camera();
        renderer.navigate(command)?;
        let changed = renderer.camera_state() != previous.state;
        if matches!(
            command,
            ViewportNavigation::Fit { .. } | ViewportNavigation::Home { .. }
        ) {
            self.record_discrete_view(previous);
        }
        Ok(changed)
    }
    fn record_discrete_view(&mut self, previous: SavedViewCamera) {
        if self.view_history.len() == VIEW_HISTORY_LIMIT {
            self.view_history.pop_front();
        }
        self.view_history.push_back(previous);
    }
    /// Restore the last discrete view without restoring old pixel dimensions.
    /// Example: `viewport.previous_view()?` runs Previous View after Home or Fit.
    pub fn previous_view(&mut self) -> Result<bool, String> {
        self.live()?;
        self.refresh_extent()?;
        let Some(previous) = self.view_history.back() else {
            return Ok(false);
        };
        if self.camera_state()? == previous.state {
            self.view_history.pop_back();
            return Ok(false);
        }
        let next = self.restore_previous_camera(previous)?;
        self.view_history.pop_back();
        self.camera = next;
        self.completed = None;
        Ok(true)
    }
    fn restore_previous_camera(&self, previous: &SavedViewCamera) -> Result<u64, String> {
        let next = super::next_generation(self.camera)?;
        self.live()?.restore_camera(previous, self.extent)?;
        Ok(next)
    }
}
