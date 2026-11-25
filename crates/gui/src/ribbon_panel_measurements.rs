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
    pub(super) fn use_available_width(&mut self, available: f32) {
        let previous = self.full_row_available.replace(available);
        if previous.is_some_and(|width| width != available) {
            self.widths
                .retain(|(_, panel), _| *panel != RibbonPanel::PartWorkFeatures);
        }
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
    owners: &RibbonPanelOwners<'_>,
) -> ([f32; 2], [f32; 2]) {
    let factor = owners.icons.layout_factor();
    let floating_notes = display.placement
        == crate::application::ribbon_display::RibbonPlacement::Floating
        && shown == TemporaryPanel::Group(RibbonPanel::PartNotes);
    let content_height = temporary_content_height(display.appearance, tab, floating_notes);
    let panel_height = 24. + content_height * factor;
    let viewport = ui.io().display_size[0];
    let TemporaryPanel::Group(panel) = shown else {
        return ([8., top], [viewport - 16., panel_height]);
    };
    let measured = layout.widths.get(&(tab, panel)).copied().unwrap_or(108.);
    let compact_width = if display.side_anchor.is_some() && panel == RibbonPanel::PartModify {
        crate::ribbon_panel_contents::compact_modify_width(
            display.appearance,
            crate::shell_ribbon::large_in_main(owners.customization),
        ) * factor
    } else if floating_notes {
        crate::ribbon_panel_contents::FLOATING_NOTES_WIDTH * factor
    } else {
        0.
    };
    let options_width = if panel == RibbonPanel::ToolOptions
        && display.placement == crate::application::ribbon_display::RibbonPlacement::Top
    {
        430. * factor
    } else {
        0.
    };
    let width = (measured.max(compact_width).max(options_width) + 32.)
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

pub(super) fn temporary_content_height(
    appearance: crate::ribbon_command_layout::RibbonAppearance,
    tab: RibbonTab,
    floating_notes: bool,
) -> f32 {
    if floating_notes {
        crate::ribbon_panel_contents::FLOATING_NOTES_HEIGHT
    } else {
        20. + appearance.tab_command_height(tab)
    }
}
