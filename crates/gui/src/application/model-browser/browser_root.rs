// SPDX-License-Identifier: GPL-2.0-only
//! The document's own browser row: choosing it, and the commands on the whole document.
use crate::application::workspace_layout::browser_view::{BrowserView, muted_menu_item};
use crate::{GuiCommandState, GuiModelBrowser, GuiShellAction, GuiViewCommand};
use imgui::{MouseButton, Ui};

const ROOT_MENU: &str = "##root-menu";

/// Frame the view on what is chosen; its direction stays as it is.
pub(crate) fn find_in_window(view: &BrowserView, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
    let [width, height] = ui.io().display_size;
    let [across, down] = view.layout_scale;
    actions.push(GuiShellAction::ViewCommand(GuiViewCommand::ZoomSelected {
        extent: [(width * across) as u32, (height * down) as u32],
    }));
}
/// A document with an origin is a thing to choose; any other only names itself.
pub(crate) fn root_row(
    ui: &Ui,
    state: &GuiCommandState,
    browser: &GuiModelBrowser,
    view: &mut BrowserView,
    actions: &mut Vec<GuiShellAction>,
) {
    use crate::GuiBrowserKind;
    let kind = match browser.kind {
        GuiBrowserKind::Part => crate::GuiTemplateKind::Part,
        GuiBrowserKind::SheetMetal => crate::GuiTemplateKind::SheetMetal,
        GuiBrowserKind::Assembly => crate::GuiTemplateKind::Assembly,
        GuiBrowserKind::Drawing => crate::GuiTemplateKind::Drawing,
        GuiBrowserKind::Presentation => crate::GuiTemplateKind::Presentation,
    };
    let glyph = crate::icons::document_glyph(kind);
    if !browser.has_origin {
        crate::browser::browser_art::prefix(ui, view, glyph, ui.cursor_screen_pos());
        ui.dummy([16., ui.text_line_height()]);
        ui.same_line_with_spacing(0., 4.);
        ui.text(&browser.name);
        return;
    }
    let mut caption = crate::application::document_tabs::windows::document_caption(
        &browser.name,
        state
            .document
            .and_then(|id| state.document_model_states.get(&id)),
    );
    if browser.model_states.as_ref().is_some_and(|members| {
        members.edit_scope() == oblikovati_model::model_state::ModelStateEditScope::AllMembers
    }) {
        caption.push_str(" (All)");
    }
    let position = ui.cursor_screen_pos();
    let chosen =
        crate::widget_abi::selectable(ui, "##browser-root-row", browser.root_selected, false);
    crate::browser::browser_art::prefix(ui, view, glyph, position);
    crate::widget_abi::literal_text(
        ui,
        [position[0] + 20., position[1]],
        ui.style_color(imgui::StyleColor::Text),
        &caption,
    );
    let asks_menu = ui.is_item_hovered() && ui.is_mouse_clicked(MouseButton::Right);
    if asks_menu {
        ui.open_popup(ROOT_MENU);
    }
    if chosen || asks_menu {
        view.selected_origin_row = None;
        actions.push(GuiShellAction::BrowserChoice {
            document: state.document.unwrap_or(0),
            choice: crate::GuiBrowserChoice::Root,
        });
    }
    root_menu(ui, state, browser, view, actions);
}
fn root_menu(
    ui: &Ui,
    state: &GuiCommandState,
    browser: &GuiModelBrowser,
    view: &mut BrowserView,
    actions: &mut Vec<GuiShellAction>,
) {
    let Some(_menu) = ui.begin_popup(ROOT_MENU) else {
        return;
    };
    file_location_item(ui, state, actions);
    if ui
        .menu_item_config("Copy File Path")
        .enabled(browser.path.is_some())
        .build()
    {
        ui.set_clipboard_text(browser.path.as_deref().unwrap_or_default());
    }
    ui.separator();
    muted_menu_item(ui, "Create Drawing View");
    muted_menu_item(ui, "Open Drawing");
    ui.separator();
    if ui.menu_item("Expand All Children") {
        view.set_expansion(true);
    }
    if ui.menu_item("Collapse All Children") {
        view.set_expansion(false);
    }
    if ui.menu_item("Find in Window") {
        find_in_window(view, ui, actions);
    }
}

fn file_location_item(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    let request = state.document.map(GuiShellAction::OpenDocumentFolder);
    let enabled = request.as_ref().is_some_and(|action| action.enabled(state));
    if ui
        .menu_item_config("Open File Location")
        .enabled(enabled)
        .build()
        && let Some(action) = request
    {
        actions.push(action);
    }
}
