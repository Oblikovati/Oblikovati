// SPDX-License-Identifier: GPL-2.0-only
//! Working Tools controls retain their actions in icon-only layouts.
use super::*;
use crate::icons::IconGlyph;

/// Draw the main-panel menu and report whether the pointer is on its trigger or popup.
pub(crate) fn highlight_new(
    ui: &Ui,
    origin: [f32; 2],
    icons: &crate::icons::GuiRibbonIcons,
    ribbon_bottom: f32,
) -> bool {
    const MENU: &str = "##highlight-new-menu";
    let clicked = if icons.appearance == crate::ribbon_command_layout::RibbonAppearance::Normal {
        icons.cursor(ui, origin, [156., 30.]);
        crate::widget_abi::button(ui, "Highlight New ▼", icons.cell([108., 26.]))
    } else {
        let (position, size) = icons.appearance.cell(3);
        icons.cursor(ui, origin, position);
        icons.button(ui, "Highlight New", IconGlyph::Preferences, size)
    };
    let trigger_hovered = ui.is_item_hovered();
    if clicked {
        ui.open_popup(MENU);
        // Keep the menu clear of the main ribbon and above its expansion.
        let top = ui.item_rect_max()[1].max(ribbon_bottom);
        crate::widget_abi::next_window_position(ui, [ui.item_rect_min()[0], top]);
    }
    let Some(_popup) = ui.begin_popup(MENU) else {
        return trigger_hovered;
    };
    ui.text("No release highlights available");
    trigger_hovered || ui.is_window_hovered()
}

pub(super) fn options(
    ui: &Ui,
    origin: [f32; 2],
    state: &GuiCommandState,
    owners: &mut crate::ribbon_panels::RibbonPanelOwners<'_>,
) -> bool {
    let icons = owners.icons;
    let (position, size) = icons.appearance.cell(0);
    icons.cursor(ui, origin, position);
    let disabled = ui.begin_disabled(state.ui_scale.is_none());
    let application = icons.button(ui, "Application Options", IconGlyph::Preferences, size);
    if owners.tools_application_tip {
        crate::ribbon_key_tips::badge(ui, "OA");
    }
    drop(disabled);
    if application {
        owners.preferences.request();
    }
    let (position, size) = icons.appearance.cell(1);
    icons.cursor(ui, origin, position);
    let disabled = ui.begin_disabled(
        state.document_units.is_none() || state.transaction_open || owners.home_active,
    );
    let document = icons.button(ui, "Document Settings", IconGlyph::Preferences, size);
    if owners.tools_document_tip {
        crate::ribbon_key_tips::badge(ui, "OD");
    }
    drop(disabled);
    if document && let Some(id) = state.document {
        owners.preferences.request_document(id);
    }
    let (position, size) = icons.appearance.cell(2);
    icons.cursor(ui, origin, position);
    let disabled = ui.begin_disabled(
        state.document_type != Some(oblikovati_api::types::DOCUMENT_PART) || owners.home_active,
    );
    let customize = icons.button(ui, "Customize", IconGlyph::Customize, size);
    owners
        .customization
        .ribbon_button(ui, owners.active_tab, customize);
    drop(disabled);
    application || document
}
pub(super) fn find(
    ui: &Ui,
    state: &GuiCommandState,
    owners: &mut crate::ribbon_panels::RibbonPanelOwners<'_>,
) {
    let disabled = ui.begin_disabled(
        owners.home_active
            || state
                .browser
                .as_ref()
                .is_none_or(|browser| browser.kind == crate::GuiBrowserKind::Drawing),
    );
    let size = owners.icons.appearance.single_cell();
    if owners.icons.button(ui, "Find", IconGlyph::Find, size) {
        owners.workspace.request_find();
    }
    drop(disabled);
}
