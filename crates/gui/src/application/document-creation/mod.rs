// SPDX-License-Identifier: GPL-2.0-only
#[path = "component_dialog.rs"]
pub(crate) mod component_dialog;
mod kind_selection;
mod template_catalog;
mod template_material;
mod template_navigation;
mod template_sort;
mod template_tiles;
#[path = "templates.rs"]
pub(crate) mod templates;
#[path = "workspaces.rs"]
pub(crate) mod workspaces;
use crate::{GuiCommandState, GuiShellAction};
use imgui::{Condition, Ui};
use template_catalog::TemplateChoice;
use template_sort::{TemplateSort, size_heading};
use templates::{
    GuiDocumentTemplate, GuiTemplateFolder as TemplateFolder, GuiTemplateKind, GuiTemplateUnits,
    TEMPLATE_GROUPS,
};

#[derive(Default)]
pub(crate) struct TemplateChooser {
    requested: bool,
    folder: TemplateFolder,
    /// Folders left behind, most recent last, and those Back has passed through.
    folders_behind: Vec<TemplateFolder>,
    folders_ahead: Vec<TemplateFolder>,
    /// Kept between openings: reopening after Cancel shows the same choice.
    selected: Option<TemplateChoice>,
    remembered_project_template: Option<std::path::PathBuf>,
    /// A drawing template offers previews, and one must be accepted before it can be created.
    drawing_layout: Option<DrawingLayout>,
    component: component_dialog::ComponentDialog,
    view: TemplateView,
    ungrouped: bool,
    sort: TemplateSort,
    gallery_keyboard: bool,
    gallery_scroll_target: Option<template_navigation::GalleryScrollTarget>,
    gallery_scroll_frames: u8,
    gallery_scroll_y: Option<f32>,
    /// A double click on a tile asks for what Create does.
    create_requested: bool,
    projects: crate::application::project_dialog::ProjectsDialog,
}
/// What a drawing template's previews offer.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum DrawingLayout {
    /// The bare sheet: Create makes the drawing.
    Template,
    /// A layout with a view: Create first asks which component it shows.
    Assembly,
}
fn size_text(bytes: Option<u64>) -> String {
    bytes.map_or_else(|| "-".to_string(), |bytes| format!("{bytes} B"))
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
}
impl TemplateChooser {
    pub(crate) fn register_dockables(
        &mut self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        self.projects
            .register_dockable(registry, "template-projects")
    }
    fn details_layout(&self, ui: &Ui, state: &GuiCommandState) -> DetailsLayout {
        let mut widths = [
            ui.calc_text_size("Name v")[0],
            ui.calc_text_size("Size v")[0],
            ui.calc_text_size("Type")[0],
        ];
        let kinds = TEMPLATE_GROUPS
            .iter()
            .flat_map(|(_, kinds)| kinds.iter().copied())
            .collect::<Vec<_>>();
        for entry in self.catalog(state, &kinds) {
            widths[0] = widths[0].max(ui.calc_text_size(entry.title())[0]);
            widths[1] = widths[1].max(ui.calc_text_size(size_text(entry.bytes(state)))[0]);
            widths[2] = widths[2].max(ui.calc_text_size(entry.type_label())[0]);
        }
        let style = ui.clone_style();
        DetailsLayout::new(widths, style.indent_spacing, style.item_spacing[0])
    }
    pub(crate) fn request(&mut self) {
        self.clear_gallery_navigation();
        self.requested = true;
    }
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
        icons: &crate::icons::GuiRibbonIcons,
    ) {
        if ui.io().key_ctrl && ui.is_key_pressed(imgui::Key::N) && !state.transaction_open {
            self.request();
        }
        if std::mem::take(&mut self.requested) {
            self.reveal_project_root(state);
            self.restore_project_selection(state);
            ui.open_popup("New document");
            dialog_geometry(ui);
        }
        if let Some(_dialog) = ui.begin_modal_popup("New document") {
            self.contents(ui, state, icons);
            self.buttons(ui, state, actions);
            self.projects.panel(ui, state);
        } else {
            self.projects.close();
            self.clear_gallery_navigation();
        }
        self.component.panel(ui);
    }
    /// A frame must follow so a nested dialog opens without waiting for an event.
    pub(crate) fn needs_frame(&self) -> bool {
        self.projects.needs_frame()
            || self.component.needs_frame()
            || self.gallery_scroll_frames > 0
    }
    fn contents(&mut self, ui: &Ui, state: &GuiCommandState, icons: &crate::icons::GuiRibbonIcons) {
        let gallery_width = gallery_column_width(ui.content_region_avail()[0]);
        ui.columns(3, "template-layout", true);
        ui.set_column_width(0, FOLDER_COLUMN_WIDTH);
        ui.set_column_width(1, gallery_width);
        ui.text("Template folders");
        self.folders(ui);
        ui.next_column();
        self.navigation_bar(ui);
        let layout = self.details_layout(ui, state);
        ui.child_window("##template-gallery")
            .size([0., GALLERY_HEIGHT])
            .horizontal_scrollbar(self.view != TemplateView::List)
            .content_size([self.gallery_width(ui, state, icons, layout), 0.])
            .build(|| self.gallery(ui, state, layout, icons));
        ui.next_column();
        self.details(ui, state);
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
    fn gallery(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        layout: DetailsLayout,
        icons: &crate::icons::GuiRibbonIcons,
    ) {
        self.navigate_gallery(ui, state);
        if self.view == TemplateView::Details {
            self.details_header(ui, layout);
        }
        if self.ungrouped {
            let all: Vec<GuiTemplateKind> = TEMPLATE_GROUPS
                .iter()
                .flat_map(|(_, kinds)| kinds.iter().copied())
                .collect();
            self.tiles(ui, state, &all, layout, icons);
            return;
        }
        // A collapsed group hides its tiles only: the selection and its details stay.
        for (group, kinds) in TEMPLATE_GROUPS {
            if let Some(_tiles) = ui.tree_node_config(group).default_open(true).push() {
                self.tiles(ui, state, kinds, layout, icons);
            }
        }
    }
    fn details_header(&mut self, ui: &Ui, layout: DetailsLayout) {
        let name = ui
            .selectable_config(self.sort.name_heading())
            .flags(imgui::SelectableFlags::DONT_CLOSE_POPUPS)
            .size([ui.calc_text_size("Name v")[0], 0.])
            .build();
        if name {
            self.sort = self.sort.toggle_name();
        }
        ui.same_line_with_pos(layout.size);
        let clicked = ui
            .selectable_config(size_heading(self.sort))
            .flags(imgui::SelectableFlags::DONT_CLOSE_POPUPS)
            .size([ui.calc_text_size("Size v")[0], 0.])
            .build();
        if clicked {
            self.sort = self.sort.toggle_size();
        }
        ui.same_line_with_pos(layout.type_column);
        ui.text("Type");
        ui.separator();
    }
    fn tiles(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        kinds: &[GuiTemplateKind],
        layout: DetailsLayout,
        icons: &crate::icons::GuiRibbonIcons,
    ) {
        let entries = self.catalog(state, kinds);
        let width = self.tile_width(ui, icons, &entries);
        for (index, entry) in entries.into_iter().enumerate() {
            if index > 0 && self.view.is_icon_view() {
                ui.same_line();
            }
            self.tile(ui, state, entry, layout, icons, width);
        }
    }
    fn tile(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        template: TemplateChoice,
        layout: DetailsLayout,
        icons: &crate::icons::GuiRibbonIcons,
        width: f32,
    ) {
        if self.selected.as_ref() == Some(&template) && self.gallery_scroll_target.is_some() {
            ui.set_keyboard_focus_here();
        }
        if self.tile_row(ui, state, &template, layout, icons, width) {
            if self.selected.as_ref() != Some(&template) {
                self.selected = Some(template);
                self.drawing_layout = None;
            }
            self.gallery_keyboard = true;
        }
    }
    fn tile_row(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        template: &TemplateChoice,
        layout: DetailsLayout,
        icons: &crate::icons::GuiRibbonIcons,
        width: f32,
    ) -> bool {
        let _file = match template {
            TemplateChoice::Project(entry) => Some(ui.push_id(entry.path.to_string_lossy())),
            TemplateChoice::BuiltIn(_) => None,
        };
        let selected = self.selected.as_ref() == Some(template);
        let chosen = if self.view.is_icon_view() {
            self.icon_tile(ui, icons, template, width, selected)
        } else {
            let _selected_color = selected.then(|| {
                ui.push_style_color(
                    imgui::StyleColor::Header,
                    ui.style_color(imgui::StyleColor::HeaderActive),
                )
            });
            ui.selectable_config(template.title())
                .flags(
                    imgui::SelectableFlags::DONT_CLOSE_POPUPS
                        | imgui::SelectableFlags::ALLOW_DOUBLE_CLICK,
                )
                .selected(selected)
                .size([0., 0.])
                .build()
        };
        let secondary = ui.is_item_clicked_with_button(imgui::MouseButton::Right);
        self.remember_selected_tile_y(ui, template);
        self.scroll_selected_tile(ui, template);
        self.create_requested |=
            ui.is_item_hovered() && ui.is_mouse_double_clicked(imgui::MouseButton::Left);
        if self.view == TemplateView::Details {
            ui.same_line_with_pos(layout.size);
            ui.text_disabled(size_text(template.bytes(state)));
            ui.same_line_with_pos(layout.type_column);
            ui.text_disabled(template.type_label());
        }
        chosen || secondary
    }
    fn details(&mut self, ui: &Ui, state: &GuiCommandState) {
        ui.text("Template details");
        if let Some(cause) = &state.template_size_issue {
            ui.text_wrapped(format!("Template sizes unavailable: {cause}"));
        }
        if let Some(cause) = &state.project_template_issue {
            ui.text_wrapped(cause);
        }
        let Some(selected) = self.selected.as_ref() else {
            ui.text_wrapped("Select a template.");
            return;
        };
        ui.text_wrapped(selected.title());
        ui.text(format!("Document type: {}", selected.type_label()));
        ui.text(selected.unit_label());
        template_material::material_detail(ui, state, selected);
        if selected.kind() == GuiTemplateKind::Drawing {
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
        let accepted = self.selected.as_ref().is_some_and(|template| {
            template.kind() != GuiTemplateKind::Drawing || self.drawing_layout.is_some()
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
            && let Some(template) = self.selected.as_ref()
        {
            if self.drawing_layout == Some(DrawingLayout::Assembly)
                && template.kind() == GuiTemplateKind::Drawing
            {
                self.component.request(active_document_name(state));
            } else {
                actions.push(template.create_action());
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
        if let Some(active) = state.projects.iter().find(|project| project.active) {
            ui.same_line();
            ui.text(format!("Project: {}", active.name));
        }
    }
}
fn active_document_name(state: &GuiCommandState) -> &str {
    state
        .documents
        .iter()
        .find(|row| Some(row.id) == state.document)
        .map_or("", |row| row.name.as_str())
}
const GALLERY_HEIGHT: f32 = 280.;
const FOLDER_COLUMN_WIDTH: f32 = 130.;
const GALLERY_COLUMN_WIDTH: f32 = 400.;
fn gallery_column_width(available: f32) -> f32 {
    ((available - FOLDER_COLUMN_WIDTH).max(0.) * 0.65).min(GALLERY_COLUMN_WIDTH)
}
#[derive(Clone, Copy)]
struct DetailsLayout {
    size: f32,
    type_column: f32,
    width: f32,
}
impl DetailsLayout {
    fn new(widths: [f32; 3], indent: f32, gap: f32) -> Self {
        let size = indent + widths[0] + gap;
        let type_column = size + widths[1] + gap;
        Self {
            size,
            type_column,
            width: type_column + widths[2] + gap,
        }
    }
}
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
