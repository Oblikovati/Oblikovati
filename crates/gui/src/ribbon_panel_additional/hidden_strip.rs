// SPDX-License-Identifier: GPL-2.0-only
//! Keep the unpinned Modify title visible while its selector entry is unchecked.
use super::*;

pub(super) fn retains(
    panel: RibbonPanel,
    expansion: &Expansion,
    display: &RibbonDisplayState,
) -> bool {
    panel == RibbonPanel::PartModify
        && !expansion.pinned
        && display.mode == RibbonDisplay::Full
        && display.selector_open
}

pub(super) fn draw(context: &PanelDrawContext<'_>, panel: RibbonPanel, expansion: &mut Expansion) {
    let ui = context.ui;
    let position = expansion.bounds[0];
    let style = ui.clone_style();
    let width = ui.calc_text_size("   Pin")[0]
        + ui.calc_text_size(panel.label())[0]
        + style.item_spacing[0]
        + 4. * style.frame_padding[0]
        + 2. * style.window_padding[0];
    let size = [width, ui.frame_height() + 2. * style.window_padding[1]];
    expansion.bounds = [position, [position[0] + size[0], position[1] + size[1]]];
    crate::widget_abi::window_geometry(ui, position, size, imgui::Condition::Always);
    ui.window(expansion_window(panel))
        .flags(
            imgui::WindowFlags::NO_DECORATION
                | imgui::WindowFlags::NO_DOCKING
                | imgui::WindowFlags::NO_SAVED_SETTINGS
                | imgui::WindowFlags::NO_FOCUS_ON_APPEARING,
        )
        .build(|| {
            let muted = ui.begin_disabled(true);
            ui.small_button("   Pin##additional-pin");
            draw_pin(ui);
            drop(muted);
            ui.same_line();
            // Clicking the strip closes the checklist, which retires the hidden expansion.
            ui.small_button(panel.label());
        });
}
