// SPDX-License-Identifier: GPL-2.0-only
//! Model Browser child work-feature rows and context menu for User Coordinate System nodes (#3918).
use crate::application::model_browser::browser_root::find_in_window;
use crate::application::workspace_layout::browser_view::{BrowserBranch, BrowserView};
use crate::browser::ORIGIN_ROWS;
use crate::icons::IconGlyph;
use crate::{GuiBrowserFeature, GuiCommandState, GuiModelBrowser, GuiShellAction, GuiUcsAction};
use imgui::Ui;

/// Render the seven standard child work features under an expanded UCS node.
pub(crate) fn ucs_child_rows(
    ui: &Ui,
    browser: &GuiModelBrowser,
    feature: &GuiBrowserFeature,
    view: &BrowserView,
) {
    if !browser.ucs_features.contains_key(&feature.id) {
        return;
    }
    for (row, label) in ORIGIN_ROWS.into_iter().enumerate() {
        let _id = ui.push_id(format!("ucs-{}-child-{row}", feature.id.0));
        let glyph = match row {
            0..=2 => IconGlyph::WorkPlane,
            3..=5 => IconGlyph::WorkAxis,
            _ => IconGlyph::WorkPoint,
        };
        crate::browser::browser_art::prefix(ui, view, glyph, ui.cursor_screen_pos());
        crate::widget_abi::dummy(ui, [16., ui.text_line_height()]);
        ui.same_line_with_spacing(0., 4.);
        let _ = ui
            .selectable_config(format!("{label}###ucs-{}-{row}", feature.id.0))
            .build();
    }
}

/// Dispatch custom feature menus (Imported Reference or UCS); returns true when handled.
pub(crate) fn custom_feature_menu(
    ui: &Ui,
    state: &GuiCommandState,
    browser: &GuiModelBrowser,
    feature: &GuiBrowserFeature,
    view: &mut BrowserView,
    actions: &mut Vec<GuiShellAction>,
) -> bool {
    if crate::browser::browser_import::reference_menu(ui, state, browser, feature, actions) {
        return true;
    }
    let Some(&visible) = browser.ucs_features.get(&feature.id) else {
        return false;
    };
    let redefine = GuiShellAction::Ucs(GuiUcsAction::Redefine(feature.id));
    if ui
        .menu_item_config("Redefine Feature")
        .enabled(redefine.enabled(state))
        .build()
    {
        actions.push(redefine);
    }
    let toggle_visibility = GuiShellAction::Ucs(GuiUcsAction::SetVisibility {
        feature: feature.id,
        visible: !visible,
    });
    if ui
        .menu_item_config("Visibility")
        .selected(visible)
        .enabled(toggle_visibility.enabled(state))
        .build()
    {
        actions.push(toggle_visibility);
    }
    ui.separator();
    if ui.menu_item("Properties") {
        view.feature_properties.request(feature);
    }
    if ui.menu_item("Expand All Children") {
        view.open_branch(BrowserBranch::Feature(feature.id));
    }
    ui.menu_item("Collapse All Children");
    if ui.menu_item("Find in Window") {
        find_in_window(view, ui, actions);
    }
    true
}
