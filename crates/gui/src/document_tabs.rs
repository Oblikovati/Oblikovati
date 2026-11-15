// SPDX-License-Identifier: GPL-2.0-only
use crate::application::document_tabs::windows::{TabRow, activation, is_active_window, tab_rows};
use crate::{GuiFrameRequest, GuiShellAction};
use imgui::{Condition, TabItemFlags, Ui};
use oblikovati_api::wire::{ActivateDocumentArgs, CloseDocumentArgs, DocumentInfo};

const SETTLING_FRAMES: u8 = 2;
/// How far below the ribbon a released tab has left the strip behind.
const DETACH_DISTANCE: f32 = 90.;
const DRAWING_TYPE_NAME: &str = "drawing";
/// The room the document list control takes at the end of the strip.
const DOCUMENT_LIST_WIDTH: f32 = 92.;
/// Room for the two scroll arrows, taken from the strip only when it overflows.
const ARROWS_WIDTH: f32 = 52.;
const TAB_ROOM: f32 = 40.;
#[derive(Default)]
pub(crate) struct DocumentTabs {
    previous: Option<u64>,
    /// The first tab shown; earlier ones wait behind the left arrow.
    first_tab: usize,
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
        band_height: f32,
        layout: &crate::application::workspace_layout::WorkspaceLayout,
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
            return;
        }
        let width = request.extent[0] as f32 / request.layout_scale()[0];
        let span = layout.tab_span(width);
        let strip = (span.0, (span.1 - DOCUMENT_LIST_WIDTH).max(1.));
        let mut strip_owner = StripOwner {
            menu: &mut self.menu,
            detached: &self.detached,
            pressed: &mut self.pressed_tab,
            first_tab: &mut self.first_tab,
            layout_changed: false,
            reveal_active,
            hovered: None,
            home,
        };
        document_strip_window(ui, strip, band_height, || {
            document_strip(ui, request, &mut strip_owner, actions)
        });
        let hovered = strip_owner.hovered.take();
        if strip_owner.layout_changed {
            self.settling_frames = SETTLING_FRAMES;
        }
        let front =
            crate::application::document_tabs::previews::front_key(&request.commands, home.active);
        self.previews.panel(ui, hovered.as_ref(), front);
        self.detach_on_release(ui, request, band_height, actions);
        self.detached_windows(ui, &request.commands, actions);
        crate::application::document_tabs::document_list_control(
            ui,
            [span.0 + strip.1, band_height + 2.],
            DOCUMENT_LIST_WIDTH,
            &request.commands,
            actions,
        );
    }
}
impl DocumentTabs {
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
    menu: &'a mut crate::application::document_tabs::TabMenu,
    detached: &'a [u64],
    pressed: &'a mut Option<u64>,
    first_tab: &'a mut usize,
    layout_changed: bool,
    reveal_active: bool,
    hovered: Option<crate::application::document_tabs::previews::HoveredTab>,
    home: &'a mut crate::application::home_recent_documents::HomeWorkspace,
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
    if tab.is_some()
        && let Some(drawing) = state.drawing_documents.get(&row.id)
    {
        crate::application::document_creation::workspaces::draw_sheet_in_window(
            ui,
            &row.name,
            drawing,
            state.drawing_projections.get(&row.id),
        );
    }
}
fn document_strip(
    ui: &Ui,
    request: &GuiFrameRequest,
    owner: &mut StripOwner<'_>,
    actions: &mut Vec<GuiShellAction>,
) {
    let state = &request.commands;
    let tabs: Vec<TabRow<'_>> = tab_rows(state)
        .into_iter()
        .filter(|tab| !owner.detached.contains(&tab.row.id))
        .collect();
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
    owner.layout_changed = first != *owner.first_tab;
    *owner.first_tab = first;
    let flags = imgui::TabBarFlags::REORDERABLE;
    if let Some(_bar) = ui.tab_bar_with_flags("##visible-documents", flags) {
        let _disabled = ui.begin_disabled(state.transaction_open);
        if first == 0 {
            home_tab(ui, owner, state, actions);
        }
        let shown = first.saturating_sub(1)..(first + count).saturating_sub(1);
        for tab in &tabs[shown.start.min(tabs.len())..shown.end.min(tabs.len())] {
            document_tab(ui, request, tab, owner, actions);
        }
    }
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
/// Home first, then one width per window, as they will be drawn.
fn tab_widths(ui: &Ui, tabs: &[TabRow<'_>]) -> Vec<f32> {
    let spacing = ui.clone_style().item_inner_spacing[0];
    std::iter::once(ui.calc_text_size("Home")[0] + TAB_ROOM / 2.)
        .chain(tabs.iter().map(|tab| {
            let label = tab_label(tab);
            ui.calc_text_size(&label)[0] + TAB_ROOM
        }))
        .map(|width| width + spacing)
        .collect()
}
fn active_tab_index(
    state: &crate::GuiCommandState,
    tabs: &[TabRow<'_>],
    home_active: bool,
) -> Option<usize> {
    if home_active {
        return Some(0);
    }
    tabs.iter()
        .position(|tab| crate::application::document_tabs::windows::is_active_window(state, tab))
        .map(|index| index + 1)
}
/// Which tabs fit from `first`; a tab just brought forward is scrolled into the strip.
fn visible_tabs(
    widths: &[f32],
    room: f32,
    first: usize,
    active: Option<usize>,
    reveal: bool,
) -> (usize, usize) {
    let mut first = first.min(widths.len().saturating_sub(1));
    if let Some(active) = active.filter(|_| reveal) {
        first = first.min(active);
        while first < active && fitting(widths, first, room) + first <= active {
            first += 1;
        }
    }
    (first, fitting(widths, first, room))
}
fn fitting(widths: &[f32], first: usize, room: f32) -> usize {
    let mut used = 0.;
    let mut count = 0;
    for width in &widths[first..] {
        used += width;
        if used > room && count > 0 {
            break;
        }
        count += 1;
    }
    count
}
/// Scrolling only moves the strip: the active document stays what it is.
fn scroll_arrows(
    ui: &Ui,
    available: f32,
    first: &mut usize,
    moved: &mut bool,
    more_to_the_right: bool,
) {
    let start = ui.window_content_region_min()[0];
    ui.set_cursor_pos([
        start + available - ARROWS_WIDTH + 4.,
        ui.window_content_region_min()[1],
    ]);
    {
        let _at_start = ui.begin_disabled(*first == 0);
        if ui.button("<##tabs-left") {
            *first = first.saturating_sub(1);
            *moved = true;
        }
    }
    ui.same_line();
    let _at_end = ui.begin_disabled(!more_to_the_right);
    if ui.button(">##tabs-right") {
        *first += 1;
        *moved = true;
    }
}
/// Home is a workspace, not a document: it has no close control and is chosen only by a click.
fn home_tab(
    ui: &Ui,
    owner: &mut StripOwner<'_>,
    state: &crate::GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
) {
    use crate::application::document_tabs::previews::{HoveredTab, PreviewKey};
    let flags = if owner.home.active {
        TabItemFlags::SET_SELECTED
    } else {
        TabItemFlags::empty()
    };
    let shown = ui.tab_item_with_flags("Home###home-workspace-tab", None, flags);
    let clicked = ui.is_item_clicked();
    if ui.is_item_hovered() {
        owner.hovered = Some(HoveredTab {
            key: PreviewKey::Home,
            bounds: [ui.item_rect_min(), ui.item_rect_max()],
            labels: vec!["Home".to_owned()],
        });
    }
    drop(shown);
    owner.menu.home_popup(ui, state, actions);
    owner.home.active |= clicked;
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
/// A tab is chosen by its own click: the strip may scroll, or Home may stand in front, without
/// the owner's document changing.
fn document_tab(
    ui: &Ui,
    request: &GuiFrameRequest,
    tab: &TabRow<'_>,
    owner: &mut StripOwner<'_>,
    actions: &mut Vec<GuiShellAction>,
) {
    let state = &request.commands;
    let behind = owner.home.active
        || state
            .document
            .is_some_and(|id| owner.detached.contains(&id));
    let selected = !behind && is_active_window(state, tab);
    let keep = draw_document_tab(ui, tab, selected);
    let clicked = ui.is_item_clicked();
    note_hovered_tab(ui, request, tab, owner);
    owner.menu.popup(ui, state, tab.row, actions);
    if ui.is_item_active() {
        *owner.pressed = Some(tab.row.id);
    }
    if !keep {
        actions.push(GuiShellAction::CloseDocument(CloseDocumentArgs {
            id: tab.row.id,
            ..Default::default()
        }));
        return;
    }
    if clicked {
        owner.home.reached_from_home = owner.home.active;
        owner.home.active = false;
        actions.extend(activation(state, tab).filter(|action| action.dispatchable(state)));
    }
}
fn note_hovered_tab(
    ui: &Ui,
    request: &GuiFrameRequest,
    tab: &TabRow<'_>,
    owner: &mut StripOwner<'_>,
) {
    use crate::application::document_tabs::previews::{HoveredTab, PreviewKey};
    if !ui.is_item_hovered() {
        return;
    }
    let saved = request
        .commands
        .document_storage
        .get(&tab.row.id)
        .and_then(|storage| storage.path.clone());
    owner.hovered = Some(HoveredTab {
        key: PreviewKey::Window(tab.window),
        bounds: [ui.item_rect_min(), ui.item_rect_max()],
        labels: saved.into_iter().chain([tab.name.clone()]).collect(),
    });
}
fn tab_label(tab: &TabRow<'_>) -> String {
    format!("{}{}", tab.name, if tab.row.dirty { " *" } else { "" })
}
fn draw_document_tab(ui: &Ui, tab: &TabRow<'_>, selected: bool) -> bool {
    let flags = if selected {
        TabItemFlags::SET_SELECTED
    } else {
        TabItemFlags::empty()
    };
    let label = tab_label(tab);
    ui.set_next_item_width(ui.calc_text_size(&label)[0] + TAB_ROOM);
    let mut keep = true;
    let contents = ui.tab_item_with_flags(
        format!("###open-document-{}", tab.window),
        Some(&mut keep),
        flags,
    );
    draw_literal_tab_label(ui, &label);
    drop(contents);
    keep
}
fn draw_literal_tab_label(ui: &Ui, label: &str) {
    let minimum = ui.item_rect_min();
    crate::widget_abi::literal_text(
        ui,
        [minimum[0] + 8., minimum[1] + 3.],
        ui.style_color(imgui::StyleColor::Text),
        label,
    );
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
