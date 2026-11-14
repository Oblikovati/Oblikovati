// SPDX-License-Identifier: GPL-2.0-only
//! Width decisions follow Go head/ui/ribbon_collapse.go; commands keep authored order.

/// Keep leading panels expanded; example: `expanded_panel_count(&[100.,200.], 10.,50.,160.)`.
pub(crate) fn expanded_panel_count(
    widths: &[f32],
    separator: f32,
    collapsed: f32,
    available: f32,
) -> usize {
    (1..=widths.len())
        .rev()
        .find(|expanded| ribbon_row_width(widths, *expanded, separator, collapsed) <= available)
        .unwrap_or(0)
}

fn ribbon_row_width(widths: &[f32], expanded: usize, separator: f32, collapsed: f32) -> f32 {
    let panels: f32 = widths
        .iter()
        .enumerate()
        .map(|(index, width)| if index < expanded { *width } else { collapsed })
        .sum();
    panels + separator * widths.len().saturating_sub(1) as f32
}

/// Keep a measured popup inside the window; example: `clamp_panel_flyout_x(600.,400.,800.)`.
pub(crate) fn clamp_panel_flyout_x(anchor: f32, width: f32, viewport: f32) -> f32 {
    if width <= 0. {
        return anchor;
    }
    anchor.min(viewport - width).max(0.)
}

#[cfg(test)]
#[path = "../tests/ribbon/fit.rs"]
mod tests;
