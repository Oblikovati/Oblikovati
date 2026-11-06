// SPDX-License-Identifier: GPL-2.0-only
use crate::{GuiAuthoringContext, GuiFrameRequest, GuiShellAction};
use imgui::{Condition, TabItemFlags, Ui};
use oblikovati_api::wire::{ActivateDocumentArgs, CloseDocumentArgs, DocumentInfo};

#[derive(Default)]
pub(crate) struct DocumentTabs {
    previous: Option<u64>,
}
impl DocumentTabs {
    /// Draw owner-backed tabs; example: `tabs.panel(ui, request, actions)` during a GUI frame.
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        request: &GuiFrameRequest,
        band_height: f32,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let force = self.previous != request.commands.document;
        self.previous = request.commands.document;
        if !request.commands.documents.iter().any(|row| row.visible) {
            return;
        }
        let width = request.extent[0] as f32 / request.framebuffer_scale[0];
        let before = actions.len();
        document_strip_window(ui, width, band_height, || {
            document_strip(ui, request, force, actions)
        });
        // #3800: a refused dirty close or activation must restore the owner's tab selection.
        if actions.len() != before {
            self.previous = None;
        }
    }
}
fn document_strip(
    ui: &Ui,
    request: &GuiFrameRequest,
    force: bool,
    actions: &mut Vec<GuiShellAction>,
) {
    let Some(_bar) = ui.tab_bar("##visible-documents") else {
        return;
    };
    let state = &request.commands;
    let _disabled =
        ui.begin_disabled(state.transaction_open || state.authoring == GuiAuthoringContext::Sketch);
    for row in state.documents.iter().filter(|row| row.visible) {
        let selected = force && state.document == Some(row.id);
        document_tab(ui, request, row, selected, force, actions);
    }
}
fn document_strip_window(ui: &Ui, width: f32, band_height: f32, contents: impl FnOnce()) {
    crate::widget_abi::window_geometry(
        ui,
        [316., band_height + 2.],
        [(width - 316.).max(1.), 32.],
        Condition::Always,
    );
    ui.window("##open-document-tabs")
        .title_bar(false)
        .movable(false)
        .resizable(false)
        .scroll_bar(false)
        .scrollable(false)
        .collapsible(false)
        .build(contents);
}
fn document_tab(
    ui: &Ui,
    request: &GuiFrameRequest,
    row: &DocumentInfo,
    selected: bool,
    force: bool,
    actions: &mut Vec<GuiShellAction>,
) {
    let (active, keep) = draw_document_tab(ui, row, selected);
    let Some(action) = document_tab_action(row.id, active, keep, force, request.commands.document)
    else {
        return;
    };
    if action.dispatchable(&request.commands) {
        actions.push(action);
    }
}
fn draw_document_tab(ui: &Ui, row: &DocumentInfo, selected: bool) -> (bool, bool) {
    let flags = if selected {
        TabItemFlags::SET_SELECTED
    } else {
        TabItemFlags::empty()
    };
    let label = format!("{}{}", row.name, if row.dirty { " *" } else { "" });
    ui.set_next_item_width(ui.calc_text_size(&label)[0] + 40.);
    let mut keep = true;
    let contents = ui.tab_item_with_flags(
        format!("###open-document-{}", row.id),
        Some(&mut keep),
        flags,
    );
    draw_literal_tab_label(ui, &label);
    (contents.is_some(), keep)
}
fn document_tab_action(
    id: u64,
    active: bool,
    keep: bool,
    force: bool,
    current: Option<u64>,
) -> Option<GuiShellAction> {
    if !keep {
        return Some(GuiShellAction::CloseDocument(CloseDocumentArgs {
            id,
            ..Default::default()
        }));
    }
    (active && !force && current != Some(id)).then_some(GuiShellAction::ActivateDocument(
        ActivateDocumentArgs { id },
    ))
}
fn draw_literal_tab_label(ui: &Ui, label: &str) {
    let minimum = ui.item_rect_min();
    crate::widget_abi::literal_text(
        ui,
        [minimum[0] + 8., minimum[1] + 3.],
        ui.style_color(imgui::StyleColor::Text),
        label,
    );
    if ui.is_item_hovered() {
        ui.tooltip_text(label);
    }
}
/// Refuse ambiguous tab identities before ImGui input; example: `validate_documents(&state.documents)?`.
pub(crate) fn validate_documents(rows: &[DocumentInfo]) -> Result<(), String> {
    let mut identities = std::collections::BTreeSet::new();
    for row in rows {
        if row.id == 0 || !identities.insert(row.id) || row.name.contains('\0') {
            return Err(format!(
                "document tab id={}, name={:?}: expected a unique nonzero identity and NUL-free label",
                row.id, row.name
            ));
        }
    }
    Ok(())
}
