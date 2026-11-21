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
        let required = panels.iter().position(|panel| *panel == RibbonPanel::Exit);
        if required.is_none() {
            return crate::ribbon_fit::expanded_panel_count(&widths, 8., collapsed, available);
        }
        crate::ribbon_fit::expanded_panel_count_reserving(
            &widths, 8., collapsed, available, required,
        )
    }
    pub(super) fn observe_scrollbar(&mut self, shown: bool) {
        self.pending_layout |= self.scrollbar_shown != shown;
        self.scrollbar_shown = shown;
    }
}

/// A group's panel opens under its button at its measured width; a whole tab spans the window.
pub(super) fn temporary_geometry(
    ui: &Ui,
    tab: RibbonTab,
    layout: &RibbonPanelLayout,
    display: &RibbonDisplayState,
    shown: TemporaryPanel,
    top: f32,
    factor: f32,
) -> ([f32; 2], [f32; 2]) {
    let panel_height = 24. + (20. + display.appearance.command_height()) * factor;
    let viewport = ui.io().display_size[0];
    let TemporaryPanel::Group(panel) = shown else {
        return ([8., top], [viewport - 16., panel_height]);
    };
    let measured = layout.widths.get(&(tab, panel)).copied().unwrap_or(108.);
    let compact_width = if display.side_anchor.is_some() && panel == RibbonPanel::PartModify {
        crate::ribbon_panel_contents::compact_modify_width(display.appearance) * factor
    } else {
        0.
    };
    let width = (measured.max(compact_width) + 32.)
        .max(140.)
        .min(viewport - 16.);
    if let Some(anchor) = display.side_anchor {
        let x = if anchor.ends_at_x {
            anchor.x - width
        } else {
            anchor.x
        };
        return ([x, anchor.y], [width, panel_height]);
    }
    let x = display.temporary_anchor.min(viewport - width - 8.).max(8.);
    ([x, top], [width, panel_height])
}
