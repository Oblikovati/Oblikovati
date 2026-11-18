// SPDX-License-Identifier: GPL-2.0-only
//! Working Tools controls retain their actions in icon-only layouts.
use super::*;
use crate::icons::IconGlyph;

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
    if icons.button(ui, "Customize", IconGlyph::Customize, size) {
        owners.customization.open(owners.active_tab);
    }
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
