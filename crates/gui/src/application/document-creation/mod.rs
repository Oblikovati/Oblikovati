// SPDX-License-Identifier: GPL-2.0-only
#[path = "component_dialog.rs"]
pub(crate) mod component_dialog;
#[path = "projects.rs"]
pub(crate) mod projects;
#[path = "templates.rs"]
pub(crate) mod templates;
#[path = "workspaces.rs"]
pub(crate) mod workspaces;
use crate::{GuiCommandState, GuiShellAction};
use imgui::{Condition, Ui};
use templates::{GuiDocumentTemplate, GuiTemplateKind, GuiTemplateUnits, TEMPLATE_GROUPS};

/// A row of the folder tree; the root holds folders and no templates of its own.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum TemplateFolder {
    Root,
    Units(GuiTemplateUnits),
}
impl Default for TemplateFolder {
    fn default() -> Self {
        Self::Units(GuiTemplateUnits::default())
    }
}
impl TemplateFolder {
    const ALL: [Self; 3] = [
        Self::Root,
        Self::Units(GuiTemplateUnits::Metric),
        Self::Units(GuiTemplateUnits::Inch),
    ];
    fn label(self) -> &'static str {
        match self {
            Self::Root => "Templates",
            Self::Units(units) => units.folder(),
        }
    }
}
#[derive(Default)]
pub(crate) struct TemplateChooser {
    requested: bool,
    folder: TemplateFolder,
    /// Folders left behind, most recent last, and those Back has passed through.
    folders_behind: Vec<TemplateFolder>,
    folders_ahead: Vec<TemplateFolder>,
    /// Kept between openings: reopening after Cancel shows the same choice.
    selected: Option<GuiDocumentTemplate>,
    /// A drawing template offers previews, and one must be accepted before it can be created.
    drawing_layout: Option<DrawingLayout>,
    component: component_dialog::ComponentDialog,
    view: TemplateView,
    ungrouped: bool,
    size_sort: Option<SizeOrder>,
    /// A double click on a tile asks for what Create does.
    create_requested: bool,
    projects: projects::ProjectsDialog,
}
/// What a drawing template's previews offer.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum DrawingLayout {
    /// The bare sheet: Create makes the drawing.
    Template,
    /// A layout with a view: Create first asks which component it shows.
    Assembly,
}
/// The direction the Details view's Size column sorts in.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum SizeOrder {
    Increasing,
    Decreasing,
}
/// A click starts increasing, and each later click reverses it.
fn next_size_order(current: Option<SizeOrder>) -> SizeOrder {
    match current {
        Some(SizeOrder::Increasing) => SizeOrder::Decreasing,
        _ => SizeOrder::Increasing,
    }
}
fn size_heading(order: Option<SizeOrder>) -> &'static str {
    match order {
        None => "Size##template-size",
        Some(SizeOrder::Increasing) => "Size ^##template-size",
        Some(SizeOrder::Decreasing) => "Size v##template-size",
    }
}
fn template_size(state: &GuiCommandState, kind: GuiTemplateKind) -> Option<u64> {
    state
        .template_sizes
        .iter()
        .find(|(sized, _)| *sized == kind)
        .map(|(_, bytes)| *bytes)
}
fn size_text(bytes: Option<u64>) -> String {
    bytes.map_or_else(|| "-".to_string(), |bytes| format!("{bytes} B"))
}
/// A group's tiles in the chosen order; without one, in the gallery's own order.
fn ordered_kinds(
    kinds: &[GuiTemplateKind],
    order: Option<SizeOrder>,
    state: &GuiCommandState,
) -> Vec<GuiTemplateKind> {
    let mut ordered = kinds.to_vec();
    ordered.sort_by_key(|kind| template_size(state, *kind));
    match order {
        None => kinds.to_vec(),
        Some(SizeOrder::Increasing) => ordered,
        Some(SizeOrder::Decreasing) => {
            ordered.reverse();
            ordered
        }
    }
}
/// How the gallery shows its tiles; a size is a row height until tiles have icons.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
enum TemplateView {
    ExtraLargeIcons,
    MediumIcons,
    SmallIcons,
    #[default]
    List,
    Details,
}
impl TemplateView {
    const ALL: [Self; 5] = [
        Self::ExtraLargeIcons,
        Self::MediumIcons,
        Self::SmallIcons,
        Self::List,
        Self::Details,
    ];
    fn label(self) -> &'static str {
        match self {
            Self::ExtraLargeIcons => "Extra Large Icons",
            Self::MediumIcons => "Medium Icons",
            Self::SmallIcons => "Small Icons",
            Self::List => "List",
            Self::Details => "Details",
        }
    }
    /// Zero lets the row take its text's own height.
    fn row_height(self) -> f32 {
        match self {
            Self::ExtraLargeIcons => 64.,
            Self::MediumIcons => 40.,
            Self::SmallIcons => 28.,
            Self::List | Self::Details => 0.,
        }
    }
}
impl TemplateChooser {
    pub(crate) fn request(&mut self) {
        self.requested = true;
    }
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if ui.io().key_ctrl && ui.is_key_pressed(imgui::Key::N) && !state.transaction_open {
            self.request();
        }
        if std::mem::take(&mut self.requested) {
            ui.open_popup("New document");
            dialog_geometry(ui);
        }
        if let Some(_dialog) = ui.begin_modal_popup("New document") {
            self.contents(ui, state);
            self.buttons(ui, state, actions);
            self.projects.panel(ui, state);
        }
        self.component.panel(ui);
    }
    /// A frame must follow so a nested dialog opens without waiting for an event.
    pub(crate) fn needs_frame(&self) -> bool {
        self.projects.needs_frame() || self.component.needs_frame()
    }
    fn contents(&mut self, ui: &Ui, state: &GuiCommandState) {
        ui.columns(3, "template-layout", true);
        ui.set_column_width(0, FOLDER_COLUMN_WIDTH);
        ui.set_column_width(1, GALLERY_COLUMN_WIDTH);
        ui.text("Template folders");
        self.folders(ui);
        ui.next_column();
        self.navigation_bar(ui);
        ui.child_window("##template-gallery")
            .size([0., GALLERY_HEIGHT])
            .build(|| self.gallery(ui, state));
        ui.next_column();
        self.details(ui);
        ui.columns(1, "template-layout-end", false);
    }
    fn folders(&mut self, ui: &Ui) {
        for folder in TemplateFolder::ALL {
            if crate::widget_abi::dialog_selectable(ui, folder.label(), self.folder == folder) {
                self.go_to(folder);
            }
        }
    }
    /// Choosing a folder starts a new path: what Back had passed through is gone.
    fn go_to(&mut self, folder: TemplateFolder) {
        if folder != self.folder {
            self.folders_behind.push(self.folder);
            self.folders_ahead.clear();
            self.folder = folder;
        }
    }
    fn back(&mut self) {
        if let Some(folder) = self.folders_behind.pop() {
            self.folders_ahead.push(self.folder);
            self.folder = folder;
        }
    }
    fn forward(&mut self) {
        if let Some(folder) = self.folders_ahead.pop() {
            self.folders_behind.push(self.folder);
            self.folder = folder;
        }
    }
    fn navigation_bar(&mut self, ui: &Ui) {
        if crate::widget_abi::button(ui, "Back", [0.; 2]) {
            self.back();
        }
        ui.same_line();
        if crate::widget_abi::button(ui, "Forward", [0.; 2]) {
            self.forward();
        }
        ui.same_line();
        if crate::widget_abi::button(ui, "Home", [0.; 2]) {
            self.go_to(TemplateFolder::Root);
        }
        ui.same_line();
        self.display_menu(ui);
    }
    fn display_menu(&mut self, ui: &Ui) {
        if crate::widget_abi::button(ui, "Display", [0.; 2]) {
            ui.open_popup(DISPLAY_MENU);
        }
        let Some(_menu) = ui.begin_popup(DISPLAY_MENU) else {
            return;
        };
        for view in TemplateView::ALL {
            let chosen = ui
                .menu_item_config(view.label())
                .selected(self.view == view)
                .build();
            if chosen {
                self.view = view;
            }
        }
        ui.separator();
        if ui
            .menu_item_config("Grouped by Type")
            .selected(!self.ungrouped)
            .build()
        {
            self.ungrouped = !self.ungrouped;
        }
    }
    fn gallery(&mut self, ui: &Ui, state: &GuiCommandState) {
        if self.view == TemplateView::Details {
            self.details_header(ui);
        }
        if self.ungrouped {
            let all: Vec<GuiTemplateKind> = TEMPLATE_GROUPS
                .iter()
                .flat_map(|(_, kinds)| kinds.iter().copied())
                .collect();
            self.tiles(ui, state, &all);
            return;
        }
        // A collapsed group hides its tiles only: the selection and its details stay.
        for (group, kinds) in TEMPLATE_GROUPS {
            if let Some(_tiles) = ui.tree_node_config(group).default_open(true).push() {
                self.tiles(ui, state, kinds);
            }
        }
    }
    fn details_header(&mut self, ui: &Ui) {
        ui.text("Name");
        ui.same_line_with_pos(DETAILS_SIZE_COLUMN);
        let clicked = ui
            .selectable_config(size_heading(self.size_sort))
            .flags(imgui::SelectableFlags::DONT_CLOSE_POPUPS)
            .size([60., 0.])
            .build();
        if clicked {
            self.size_sort = Some(next_size_order(self.size_sort));
        }
        ui.same_line_with_pos(DETAILS_TYPE_COLUMN);
        ui.text("Type");
        ui.separator();
    }
    fn tiles(&mut self, ui: &Ui, state: &GuiCommandState, kinds: &[GuiTemplateKind]) {
        for kind in ordered_kinds(kinds, self.size_sort, state) {
            self.tile(ui, state, kind);
        }
    }
    fn tile(&mut self, ui: &Ui, state: &GuiCommandState, kind: GuiTemplateKind) {
        // The root holds folders only: its gallery shows the headings above nothing.
        let TemplateFolder::Units(units) = self.folder else {
            return;
        };
        let template = GuiDocumentTemplate { kind, units };
        if self.tile_row(ui, state, template) {
            self.selected = Some(template);
            self.drawing_layout = None;
        }
    }
    fn tile_row(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        template: GuiDocumentTemplate,
    ) -> bool {
        let selected = self.selected == Some(template);
        let chosen = ui
            .selectable_config(template.title())
            .flags(
                imgui::SelectableFlags::DONT_CLOSE_POPUPS
                    | imgui::SelectableFlags::ALLOW_DOUBLE_CLICK,
            )
            .selected(selected)
            .size([0., self.view.row_height()])
            .build();
        self.create_requested |=
            ui.is_item_hovered() && ui.is_mouse_double_clicked(imgui::MouseButton::Left);
        if self.view == TemplateView::Details {
            ui.same_line_with_pos(DETAILS_SIZE_COLUMN);
            ui.text_disabled(size_text(template_size(state, template.kind)));
            ui.same_line_with_pos(DETAILS_TYPE_COLUMN);
            ui.text_disabled(template.type_label());
        }
        chosen
    }
    fn details(&mut self, ui: &Ui) {
        ui.text("Template details");
        let Some(selected) = self.selected else {
            ui.text_wrapped("Select a template.");
            return;
        };
        ui.text_wrapped(selected.title());
        ui.text(format!("Document type: {}", selected.type_label()));
        ui.text(selected.unit_label());
        if selected.kind == GuiTemplateKind::Drawing {
            self.drawing_previews(ui);
        }
    }
    fn drawing_previews(&mut self, ui: &Ui) {
        for (label, layout) in [
            ("Use Template", DrawingLayout::Template),
            ("Assembly Layout", DrawingLayout::Assembly),
        ] {
            let chosen = self.drawing_layout == Some(layout);
            if crate::widget_abi::dialog_selectable(ui, label, chosen) {
                self.drawing_layout = Some(layout);
            }
        }
    }
    fn can_create(&self, state: &GuiCommandState) -> bool {
        let accepted = self.selected.is_some_and(|template| {
            template.kind != GuiTemplateKind::Drawing || self.drawing_layout.is_some()
        });
        accepted && !state.transaction_open
    }
    fn buttons(&mut self, ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        ui.separator();
        let can_create = self.can_create(state);
        let disabled = ui.begin_disabled(!can_create);
        let pressed = crate::widget_abi::button(ui, "Create", [0.; 2]);
        drop(disabled);
        // A list or dialog above the chooser keeps its own Enter and Escape.
        let keys_are_ours = !crate::widget_abi::nested_popup_open(ui);
        let entered = keys_are_ours && ui.is_key_pressed_no_repeat(imgui::Key::Enter);
        let doubled = std::mem::take(&mut self.create_requested);
        if can_create
            && (pressed || entered || doubled)
            && let Some(template) = self.selected
        {
            if self.drawing_layout == Some(DrawingLayout::Assembly)
                && template.kind == GuiTemplateKind::Drawing
            {
                self.component.request();
            } else {
                actions.push(GuiShellAction::CreateDocument(template));
            }
            ui.close_current_popup();
        }
        ui.same_line();
        let cancelled = crate::widget_abi::button(ui, "Cancel", [0.; 2])
            || (keys_are_ours && ui.is_key_pressed(imgui::Key::Escape));
        if cancelled {
            ui.close_current_popup();
        }
        ui.same_line();
        if crate::widget_abi::button(ui, "Projects", [0.; 2]) {
            self.projects.request();
        }
    }
}
const GALLERY_HEIGHT: f32 = 280.;
const FOLDER_COLUMN_WIDTH: f32 = 130.;
const GALLERY_COLUMN_WIDTH: f32 = 400.;
const DETAILS_SIZE_COLUMN: f32 = 260.;
const DETAILS_TYPE_COLUMN: f32 = 330.;
const DISPLAY_MENU: &str = "##template-display";
fn dialog_geometry(ui: &Ui) {
    let size = [
        820_f32.min(ui.io().display_size[0] - 24.),
        400_f32.min(ui.io().display_size[1] - 24.),
    ];
    let position = std::array::from_fn(|axis| (ui.io().display_size[axis] - size[axis]) * 0.5);
    crate::widget_abi::window_geometry(ui, position, size, Condition::Appearing);
}
#[cfg(test)]
#[path = "../../../tests/application/document-creation/templates.rs"]
mod tests;
