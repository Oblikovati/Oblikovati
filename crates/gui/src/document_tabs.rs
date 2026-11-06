// SPDX-License-Identifier: GPL-2.0-only
use crate::{GuiAuthoringContext, GuiFrameRequest, GuiShellAction};
use imgui::{Condition, TabItemFlags, Ui};
use oblikovati_api::wire::{ActivateDocumentArgs, CloseDocumentArgs, DocumentInfo};

const SETTLING_FRAMES: u8 = 2;
#[derive(Default)]
pub(crate) struct DocumentTabs {
    previous: Option<u64>,
    pub menu: crate::application::document_tabs::TabMenu,
    /// The labels of the last frame; a change needs the frames below to finish its layout.
    drawn_labels: Vec<(u64, String, bool)>,
    settling_frames: u8,
    right_button_last_frame: bool,
}
impl DocumentTabs {
    /// Draw owner-backed tabs; example: `tabs.panel(ui, request, actions)` during a GUI frame.
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        request: &GuiFrameRequest,
        band_height: f32,
        layout: &crate::application::workspace_layout::WorkspaceLayout,
        actions: &mut Vec<GuiShellAction>,
    ) {
        // A sketch edit keeps its document selected: a tab click must not move the bar away from it.
        self.track_labels(&request.commands);
        let force = self.previous != request.commands.document
            || request.commands.authoring == GuiAuthoringContext::Sketch
            || self.right_button_selection(ui);
        self.previous = request.commands.document;
        if !layout.tabs_visible || !request.commands.documents.iter().any(|row| row.visible) {
            return;
        }
        let width = request.extent[0] as f32 / request.layout_scale()[0];
        let before = actions.len();
        let menu = &mut self.menu;
        document_strip_window(ui, layout.tab_span(width), band_height, || {
            document_strip(ui, request, force, menu, actions)
        });
        // #3800: a refused dirty close or activation must restore the owner's tab selection.
        if actions.len() != before {
            self.previous = None;
        }
    }
}
impl DocumentTabs {
    /// Dear ImGui lays a new tab out in its second frame, and GUI frames otherwise wait for an
    /// event: without this, a new document's tab stays invisible until the pointer moves.
    fn track_labels(&mut self, state: &crate::GuiCommandState) {
        self.settling_frames = self.settling_frames.saturating_sub(1);
        let labels: Vec<(u64, String, bool)> = state
            .documents
            .iter()
            .map(|row| (row.id, row.name.clone(), row.dirty))
            .collect();
        if labels != self.drawn_labels {
            self.drawn_labels = labels;
            self.settling_frames = SETTLING_FRAMES;
        }
    }
    /// Dear ImGui selects a tab on a right press and again on its release, a frame before the
    /// selection shows. A menu on an inactive tab must not activate it, so the owner's selection
    /// stays forced through the release and the frame after it.
    fn right_button_selection(&mut self, ui: &Ui) -> bool {
        let right = imgui::MouseButton::Right;
        let active = ui.is_mouse_down(right) || ui.is_mouse_released(right);
        let held = active || self.right_button_last_frame;
        self.right_button_last_frame = active;
        if active {
            self.settling_frames = SETTLING_FRAMES;
        }
        held
    }
    pub(crate) fn needs_frame(&self) -> bool {
        self.settling_frames > 0
    }
}
fn document_strip(
    ui: &Ui,
    request: &GuiFrameRequest,
    force: bool,
    menu: &mut crate::application::document_tabs::TabMenu,
    actions: &mut Vec<GuiShellAction>,
) {
    let Some(_bar) = ui.tab_bar("##visible-documents") else {
        return;
    };
    let state = &request.commands;
    // A sketch edit keeps close marks live: closing asks to leave the sketch first.
    let _disabled = ui.begin_disabled(state.transaction_open);
    for row in state.documents.iter().filter(|row| row.visible) {
        let selected = force && state.document == Some(row.id);
        document_tab(ui, request, row, selected, force, menu, actions);
    }
}
fn document_strip_window(ui: &Ui, span: (f32, f32), band_height: f32, contents: impl FnOnce()) {
    crate::widget_abi::window_geometry(
        ui,
        [span.0, band_height + 2.],
        [span.1, 32.],
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
    menu: &mut crate::application::document_tabs::TabMenu,
    actions: &mut Vec<GuiShellAction>,
) {
    let (active, keep) = draw_document_tab(ui, row, selected);
    menu.popup(ui, &request.commands, row, actions);
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
