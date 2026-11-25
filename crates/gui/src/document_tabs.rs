// SPDX-License-Identifier: GPL-2.0-only
mod strip_render;
mod tab_order;
mod tab_overflow;

use crate::application::document_tabs::windows::{TabRow, activation, is_active_window, tab_rows};
use crate::{GuiFrameRequest, GuiShellAction};
use imgui::{Condition, TabItemFlags, Ui};
use oblikovati_api::wire::{ActivateDocumentArgs, CloseDocumentArgs, DocumentInfo};
use strip_render::{active_tab_index, document_strip_window, document_tab, home_tab, tab_widths};
use tab_order::{TabKey, TabOrder};
use tab_overflow::{ARROWS_WIDTH, scroll_arrows, visible_tabs};

#[cfg(test)]
type TestTabBounds = std::collections::BTreeMap<u64, ([[f32; 2]; 2], [f32; 2])>;

const SETTLING_FRAMES: u8 = 2;
/// How far above the bottom strip a released tab has left it behind.
const DETACH_DISTANCE: f32 = 90.;
const DRAWING_TYPE_NAME: &str = "drawing";
/// The room the document list control takes at the end of the strip.
const DOCUMENT_LIST_WIDTH: f32 = 92.;
const TAB_ROOM: f32 = 40.;
#[derive(Default)]
pub(crate) struct DocumentTabs {
    #[cfg(test)]
    pub(crate) test_tab_bounds: TestTabBounds,
    previous: Option<u64>,
    /// The first tab shown; earlier ones wait behind the left arrow.
    first_tab: usize,
    order: TabOrder,
    home_press: crate::popup_press::PopupPress<()>,
    pub previews: crate::application::document_tabs::previews::TabPreviews,
    pub menu: crate::application::document_tabs::TabMenu,
    /// The labels of the last frame; a change needs the frames below to finish its layout.
    drawn_labels: Vec<(u64, String, bool)>,
    settling_frames: u8,
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
        layout: &crate::application::workspace_layout::WorkspaceLayout,
        area: crate::application::document_creation::workspaces::WorkspaceArea,
        home: &mut crate::application::home_recent_documents::HomeWorkspace,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.track_labels(&request.commands);
        self.forget_closed_detached(&request.commands);
        self.settle_after_right_button(ui);
        let reveal_active = self.previous != request.commands.document;
        self.previous = request.commands.document;
        let any_document = request.commands.documents.iter().any(|row| row.visible);
        if !layout.tabs_visible || !(any_document || home.active) {
            self.order.cancel_input();
            self.pressed_tab = None;
            self.home_press.finish_frame(ui);
            return;
        }
        let height = request.extent[1] as f32 / request.layout_scale()[1];
        let width = request.extent[0] as f32 / request.layout_scale()[0];
        let strip_top = layout.tab_top(height);
        let span = layout.document_tab_span(width, area);
        let strip = (span.0, (span.1 - DOCUMENT_LIST_WIDTH).max(1.));
        let mut strip_owner = StripOwner {
            #[cfg(test)]
            test_tab_bounds: &mut self.test_tab_bounds,
            menu: &mut self.menu,
            detached: &self.detached,
            pressed: &mut self.pressed_tab,
            first_tab: &mut self.first_tab,
            order: &mut self.order,
            layout_changed: false,
            reveal_active,
            hovered: None,
            home,
            home_press: &mut self.home_press,
        };
        document_strip_window(ui, strip, strip_top, || {
            document_strip(ui, request, &mut strip_owner, actions)
        });
        let hovered = strip_owner.hovered.take();
        strip_owner.home_press.finish_frame(ui);
        if strip_owner.layout_changed {
            self.settling_frames = SETTLING_FRAMES;
        }
        let front =
            crate::application::document_tabs::previews::front_key(&request.commands, home.active);
        self.previews.panel(ui, hovered.as_ref(), front);
        self.detach_on_release(ui, request, strip_top, actions);
        crate::application::document_tabs::document_list_control(
            ui,
            [span.0 + strip.1, strip_top + 2.],
            DOCUMENT_LIST_WIDTH,
            &request.commands,
            actions,
        );
    }
}
impl DocumentTabs {
    pub(crate) fn detached_drawings(&self) -> &[u64] {
        &self.detached
    }
    pub(crate) fn restore_detached_drawing(&mut self, drawing: u64) {
        self.detached.retain(|id| *id != drawing);
        self.settling_frames = SETTLING_FRAMES;
    }
    /// Dear ImGui lays a new tab out in its second frame, and GUI frames otherwise wait for an
    /// event: without this, a new document's tab stays invisible until the pointer moves.
    fn track_labels(&mut self, state: &crate::GuiCommandState) {
        self.settling_frames = self.settling_frames.saturating_sub(1);
        let labels: Vec<(u64, String, bool)> = tab_rows(state)
            .into_iter()
            .map(|tab| (tab.window, tab.name, tab.row.dirty))
            .collect();
        if labels != self.drawn_labels {
            self.drawn_labels = labels;
            self.settling_frames = SETTLING_FRAMES;
        }
    }
    /// A right press on a tab opens its menu on the release, a frame later.
    fn settle_after_right_button(&mut self, ui: &Ui) {
        let right = imgui::MouseButton::Right;
        if ui.is_mouse_down(right) || ui.is_mouse_released(right) {
            self.settling_frames = SETTLING_FRAMES;
        }
    }
    pub(crate) fn needs_frame(&self) -> bool {
        self.settling_frames > 0 || self.previews.needs_frame()
    }
}
/// The strip's borrows of its owner, for the frame's tab bar closure.
struct StripOwner<'a> {
    #[cfg(test)]
    test_tab_bounds: &'a mut TestTabBounds,
    menu: &'a mut crate::application::document_tabs::TabMenu,
    detached: &'a [u64],
    pressed: &'a mut Option<u64>,
    first_tab: &'a mut usize,
    order: &'a mut TabOrder,
    layout_changed: bool,
    reveal_active: bool,
    hovered: Option<crate::application::document_tabs::previews::HoveredTab>,
    home: &'a mut crate::application::home_recent_documents::HomeWorkspace,
    home_press: &'a mut crate::popup_press::PopupPress<()>,
}

enum StripTab<'a> {
    Home,
    Document(TabRow<'a>),
}
impl StripTab<'_> {
    fn key(&self) -> TabKey {
        match self {
            Self::Home => TabKey::Home,
            Self::Document(tab) => TabKey::Window(tab.window),
        }
    }
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
    /// A drawing tab released well above the strip leaves it; only drawings have a view that does
    /// not need the single 3D viewport.
    fn detach_on_release(
        &mut self,
        ui: &Ui,
        request: &GuiFrameRequest,
        strip_top: f32,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if !ui.is_mouse_released(imgui::MouseButton::Left) {
            return;
        }
        let Some(id) = self.pressed_tab.take() else {
            return;
        };
        let above_strip = ui.io().mouse_pos[1] < strip_top - DETACH_DISTANCE;
        let state = &request.commands;
        if !above_strip || !is_drawing(state, id) {
            return;
        }
        let workspace = next_attached(state, id, &self.detached)
            .map_or(GuiShellAction::ShowHome, |id| {
                GuiShellAction::ActivateDocument(ActivateDocumentArgs { id })
            });
        self.detached.push(id);
        actions.push(workspace);
        self.settling_frames = SETTLING_FRAMES;
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
fn document_strip(
    ui: &Ui,
    request: &GuiFrameRequest,
    owner: &mut StripOwner<'_>,
    actions: &mut Vec<GuiShellAction>,
) {
    let state = &request.commands;
    owner
        .order
        .begin_input(ui.is_mouse_clicked(imgui::MouseButton::Left));
    let mut tabs: Vec<StripTab<'_>> = std::iter::once(StripTab::Home)
        .chain(tab_rows(state).into_iter().map(StripTab::Document))
        .collect();
    owner.order.synchronize_state(state);
    tabs.retain(|tab| match tab {
        StripTab::Home => true,
        StripTab::Document(tab) => !owner.detached.contains(&tab.row.id),
    });
    tabs.sort_by_key(|tab| owner.order.rank(tab.key()));
    let widths = tab_widths(ui, &tabs);
    let available = ui.content_region_avail()[0];
    let overflow = widths.iter().sum::<f32>() > available;
    let room = if overflow {
        available - ARROWS_WIDTH
    } else {
        available
    };
    let active = active_tab_index(state, &tabs, owner.home.active);
    let (first, count) = visible_tabs(&widths, room, *owner.first_tab, active, owner.reveal_active);
    // A tab reappearing in the strip lays out on its second frame, as a new one does.
    let shown = &tabs[first..first + count];
    let keys: Vec<_> = shown.iter().map(StripTab::key).collect();
    let subset_changed = owner.order.subset(&keys);
    owner.layout_changed = first != *owner.first_tab || subset_changed;
    *owner.first_tab = first;
    let offscreen = active.is_none_or(|index| !(first..first + count).contains(&index));
    let _neutral = offscreen.then(|| tab_overflow::neutral_selection(ui));
    let flags = if owner.order.reorderable() {
        imgui::TabBarFlags::REORDERABLE
    } else {
        imgui::TabBarFlags::empty()
    };
    if let Some(_bar) = ui.tab_bar_with_flags("##visible-documents", flags) {
        let _disabled = ui.begin_disabled(state.transaction_open);
        for tab in shown {
            match tab {
                StripTab::Home => home_tab(ui, owner, state, actions),
                StripTab::Document(tab) => document_tab(ui, request, tab, owner, actions),
            }
        }
    }
    let delta = ui.mouse_drag_delta();
    let dragging =
        ui.is_mouse_dragging(imgui::MouseButton::Left) && delta[0].abs() > delta[1].abs();
    owner.layout_changed |= owner
        .order
        .observe_drag(dragging, ui.is_mouse_released(imgui::MouseButton::Left));
    if overflow {
        let more_to_the_right = first + count < widths.len();
        scroll_arrows(
            ui,
            available,
            owner.first_tab,
            &mut owner.layout_changed,
            more_to_the_right,
        );
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
