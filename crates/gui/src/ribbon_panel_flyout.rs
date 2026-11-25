// SPDX-License-Identifier: GPL-2.0-only
//! Measured expanded groups share the same icon-aware bounds in a flyout.
use super::*;
pub(super) fn draw_panel_popup(
    context: &mut PanelDrawContext<'_>,
    tab: RibbonTab,
    panel: RibbonPanel,
    identity: &str,
    anchor: f32,
    layout: &mut RibbonPanelLayout,
) {
    let ui = context.ui;
    let width = layout.widths.get(&(tab, panel)).copied().unwrap_or(108.);
    let factor = context.owners.icons.layout_factor();
    let command_height = context.owners.icons.appearance.tab_command_height(tab);
    let size = popup_geometry(
        ui,
        anchor,
        width,
        78. + layout.scrollbar_height(ui) + (20. + command_height) * factor,
        factor,
        command_height,
    );
    let Some(_popup) = ui.begin_popup(identity) else {
        return;
    };
    let region = crate::widget_abi::scroll_region(ui, "##panel-contents", size);
    if !region.visible {
        return;
    }
    let before = context.actions.len();
    let (measured, local_action) = draw_expanded(context, tab, panel, [0.; 2], layout, None);
    // Retain the compact Work Features position after regrouping its Plane choices.
    // A change in the ribbon width will measure the group again.
    if panel != RibbonPanel::PartWorkFeatures || measured >= width {
        layout.record_width(tab, panel, measured);
    }
    if context.actions.len() != before || local_action {
        ui.close_current_popup();
    }
}
fn popup_geometry(
    ui: &Ui,
    anchor: f32,
    width: f32,
    band_height: f32,
    factor: f32,
    command_height: f32,
) -> [f32; 2] {
    let viewport = ui.io().display_size[0];
    let content_width = width.min((viewport - 32.).max(1.));
    let padding = ui.clone_style().window_padding;
    let outer_width = content_width + 2. * padding[0];
    let height = (20. + command_height) * factor
        + if content_width < width {
            ui.clone_style().scrollbar_size
        } else {
            0.
        };
    let x = crate::ribbon_fit::clamp_panel_flyout_x(anchor, outer_width, viewport);
    crate::widget_abi::window_geometry(
        ui,
        [x, band_height],
        [outer_width, height + 2. * padding[1]],
        imgui::Condition::Always,
    );
    [content_width, height]
}
