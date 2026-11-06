// SPDX-License-Identifier: GPL-2.0-only
use crate::{GuiFrameRequest, GuiShellAction};
use imgui::{Condition, TabItemFlags, Ui};
use oblikovati_api::wire::{ActivateDocumentArgs, CloseDocumentArgs, DocumentInfo};

const SETTLING_FRAMES: u8 = 2;
/// How far below the ribbon a released tab has left the strip behind.
const DETACH_DISTANCE: f32 = 90.;
const DRAWING_TYPE_NAME: &str = "drawing";
/// The room the document list control takes at the end of the strip.
const DOCUMENT_LIST_WIDTH: f32 = 92.;
#[derive(Default)]
pub(crate) struct DocumentTabs {
    previous: Option<u64>,
    pub menu: crate::application::document_tabs::TabMenu,
    /// The labels of the last frame; a change needs the frames below to finish its layout.
    drawn_labels: Vec<(u64, String, bool)>,
    settling_frames: u8,
    right_button_last_frame: bool,
    /// The drawing tab under the pointer, while a press could still detach it.
    pressed_tab: Option<u64>,
    /// Drawings shown in their own window; they leave this strip until the window closes.
    detached: Vec<u64>,
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
        self.track_labels(&request.commands);
        self.forget_closed_detached(&request.commands);
        let force = self.previous != request.commands.document || self.right_button_selection(ui);
        self.previous = request.commands.document;
        if !layout.tabs_visible || !request.commands.documents.iter().any(|row| row.visible) {
            return;
        }
        let width = request.extent[0] as f32 / request.layout_scale()[0];
        let before = actions.len();
        let span = layout.tab_span(width);
        let strip = (span.0, (span.1 - DOCUMENT_LIST_WIDTH).max(1.));
        let mut strip_owner = StripOwner {
            menu: &mut self.menu,
            detached: &self.detached,
            pressed: &mut self.pressed_tab,
        };
        document_strip_window(ui, strip, band_height, || {
            document_strip(ui, request, force, &mut strip_owner, actions)
        });
        self.detach_on_release(ui, request, band_height, actions);
        self.detached_windows(ui, &request.commands, actions);
        crate::application::document_tabs::document_list_control(
            ui,
            [span.0 + strip.1, band_height + 2.],
            DOCUMENT_LIST_WIDTH,
            &request.commands,
            actions,
        );
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
/// The strip's borrows of its owner, for the frame's tab bar closure.
struct StripOwner<'a> {
    menu: &'a mut crate::application::document_tabs::TabMenu,
    detached: &'a [u64],
    pressed: &'a mut Option<u64>,
}
impl DocumentTabs {
    fn forget_closed_detached(&mut self, state: &crate::GuiCommandState) {
        let open = |id: &u64| state.documents.iter().any(|row| row.id == *id);
        self.detached.retain(open);
    }
    /// The main workspace leaves a detached drawing's sheet to its own window.
    pub(crate) fn is_detached(&self, document: Option<u64>) -> bool {
        document.is_some_and(|id| self.detached.contains(&id))
    }
    /// A drawing tab released well below the strip leaves it; only drawings have a view that does
    /// not need the single 3D viewport.
    fn detach_on_release(
        &mut self,
        ui: &Ui,
        request: &GuiFrameRequest,
        band_height: f32,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if !ui.is_mouse_released(imgui::MouseButton::Left) {
            return;
        }
        let Some(id) = self.pressed_tab.take() else {
            return;
        };
        let below_strip = ui.io().mouse_pos[1] > band_height + DETACH_DISTANCE;
        let state = &request.commands;
        let Some(next) = next_attached(state, id, &self.detached).filter(|_| below_strip) else {
            return;
        };
        if is_drawing(state, id) {
            self.detached.push(id);
            actions.push(GuiShellAction::ActivateDocument(ActivateDocumentArgs {
                id: next,
            }));
            self.settling_frames = SETTLING_FRAMES;
        }
    }
    fn detached_windows(
        &mut self,
        ui: &Ui,
        state: &crate::GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let menu = &mut self.menu;
        self.detached.retain(|id| {
            state
                .documents
                .iter()
                .find(|row| row.id == *id)
                .is_some_and(|row| detached_window(ui, state, row, menu, actions))
        });
    }
}
fn is_drawing(state: &crate::GuiCommandState, id: u64) -> bool {
    state
        .documents
        .iter()
        .any(|row| row.id == id && row.type_ == DRAWING_TYPE_NAME)
}
/// The document the main workspace shows once this one leaves it.
fn next_attached(state: &crate::GuiCommandState, id: u64, detached: &[u64]) -> Option<u64> {
    state
        .documents
        .iter()
        .find(|row| row.visible && row.id != id && !detached.contains(&row.id))
        .map(|row| row.id)
}
/// One floating window for a drawing, with its own tab; closing it returns the tab to the strip.
/// A click in it moves the shared ribbon and browser to the drawing.
fn detached_window(
    ui: &Ui,
    state: &crate::GuiCommandState,
    row: &DocumentInfo,
    menu: &mut crate::application::document_tabs::TabMenu,
    actions: &mut Vec<GuiShellAction>,
) -> bool {
    let mut keep = true;
    crate::widget_abi::window_geometry(ui, [420., 240.], [640., 480.], Condition::FirstUseEver);
    ui.window(format!("{}###detached-document-{}", row.name, row.id))
        .opened(&mut keep)
        .flags(imgui::WindowFlags::NO_DOCKING | imgui::WindowFlags::NO_COLLAPSE)
        .build(|| {
            detached_tab(ui, state, row, menu, actions);
            let clicked = ui.is_window_hovered() && ui.is_mouse_clicked(imgui::MouseButton::Left);
            if clicked && state.document != Some(row.id) {
                actions.push(GuiShellAction::ActivateDocument(ActivateDocumentArgs {
                    id: row.id,
                }));
            }
        });
    keep
}
fn detached_tab(
    ui: &Ui,
    state: &crate::GuiCommandState,
    row: &DocumentInfo,
    menu: &mut crate::application::document_tabs::TabMenu,
    actions: &mut Vec<GuiShellAction>,
) {
    let Some(_bar) = ui.tab_bar("##detached-tabs") else {
        return;
    };
    let tab = ui.tab_item(&row.name);
    menu.popup(ui, state, row, actions);
    if tab.is_some() {
        crate::application::document_creation::workspaces::draw_sheet_in_window(ui, &row.name);
    }
}
fn document_strip(
    ui: &Ui,
    request: &GuiFrameRequest,
    force: bool,
    owner: &mut StripOwner<'_>,
    actions: &mut Vec<GuiShellAction>,
) {
    let flags = imgui::TabBarFlags::REORDERABLE;
    let Some(_bar) = ui.tab_bar_with_flags("##visible-documents", flags) else {
        return;
    };
    let state = &request.commands;
    let _disabled = ui.begin_disabled(state.transaction_open);
    let shown = state
        .documents
        .iter()
        .filter(|row| row.visible && !owner.detached.contains(&row.id));
    for row in shown {
        let selected = force && state.document == Some(row.id);
        document_tab(ui, request, row, selected, force, owner, actions);
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
    owner: &mut StripOwner<'_>,
    actions: &mut Vec<GuiShellAction>,
) {
    let (active, keep) = draw_document_tab(ui, row, selected);
    owner.menu.popup(ui, &request.commands, row, actions);
    if ui.is_item_active() {
        *owner.pressed = Some(row.id);
    }
    // While a detached drawing holds the shared controls, this strip's own selection is stale:
    // only a real click on a tab asks for a change.
    let state = &request.commands;
    let stale = state
        .document
        .is_some_and(|id| owner.detached.contains(&id));
    if stale && keep && ui.is_item_clicked() {
        actions.push(GuiShellAction::ActivateDocument(ActivateDocumentArgs {
            id: row.id,
        }));
    }
    let force = force || stale;
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
