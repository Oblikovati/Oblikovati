// SPDX-License-Identifier: GPL-2.0-only
//! Measure panel widths for the current ribbon appearance.
use super::*;

impl RibbonPanelLayout {
    pub(crate) fn scrollbar_height(&self, ui: &Ui) -> f32 {
        if self.scrollbar_shown {
            ui.clone_style().scrollbar_size
        } else {
            0.
        }
    }
    pub(super) fn use_appearance(
        &mut self,
        appearance: crate::ribbon_command_layout::RibbonAppearance,
    ) {
        if self.measured_appearance != Some(appearance) {
            self.widths.clear();
            self.pending_layout = true;
            self.measured_appearance = Some(appearance);
        }
    }
    pub(super) fn record_width(&mut self, tab: RibbonTab, panel: RibbonPanel, width: f32) {
        let old = self.widths.insert((tab, panel), width);
        self.pending_layout |= old != Some(width);
    }
    pub(super) fn expanded_count(
        &self,
        tab: RibbonTab,
        panels: &[RibbonPanel],
        collapsed: f32,
        available: f32,
    ) -> usize {
        let widths: Vec<f32> = panels
            .iter()
            .map(|panel| self.widths.get(&(tab, *panel)).copied().unwrap_or(0.))
            .collect();
        crate::ribbon_fit::expanded_panel_count(&widths, 8., collapsed, available)
    }
    pub(super) fn observe_scrollbar(&mut self, shown: bool) {
        self.pending_layout |= self.scrollbar_shown != shown;
        self.scrollbar_shown = shown;
    }
}
