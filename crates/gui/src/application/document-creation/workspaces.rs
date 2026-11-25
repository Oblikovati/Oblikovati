// SPDX-License-Identifier: GPL-2.0-only
//! What a drawing or a presentation shows around the viewport once it is created.
use crate::application::document_saving::folder_browser::{
    DialogEscape, FileTypeList, FolderBrowser, dialog_buttons, file_name_field, file_type_field,
};
use crate::application::file_system::NativeFolders;
use crate::{GuiCommandState, GuiShellAction};
use imgui::{Condition, Ui};
use oblikovati_api::types::{DOCUMENT_DRAWING, DOCUMENT_PRESENTATION};
use std::rc::Rc;
#[path = "assembly_place_dialog.rs"]
mod assembly_place_dialog;
#[path = "base_view_dialog.rs"]
mod base_view_dialog;
#[path = "drawing_lines.rs"]
mod drawing_lines;

const INSERT: &str = "Insert";
/// An A3 landscape sheet: the built-in drawing template's only format.
#[cfg(test)]
const SHEET_RATIO: f32 = 420. / 297.;
const SHEET_MARGIN: f32 = 12.;
const TITLE_BLOCK_SIZE: [f32; 2] = [280., 70.];
pub(super) const SHEET_COLOR: [f32; 4] = [0.93, 0.93, 0.91, 1.];
pub(super) const LINE_COLOR: [f32; 4] = [0.1, 0.1, 0.12, 1.];
const SNAPSHOT_WIDTH: f32 = 220.;
const STORYBOARD_HEIGHT: f32 = 128.;

/// The screen area around the viewport that a workspace may use.
#[derive(Clone, Copy, Debug, PartialEq)]
pub(crate) struct WorkspaceArea {
    pub left: f32,
    pub top: f32,
    pub right: f32,
    pub bottom: f32,
}
impl WorkspaceArea {
    /// Convert the available logical area to a contained physical framebuffer rectangle.
    /// Example: `area.framebuffer_region([2., 2.], [1440, 900])` for fit navigation.
    pub(crate) fn framebuffer_region(self, scale: [f32; 2], extent: [u32; 2]) -> Option<[u32; 4]> {
        let x = |value: f32| (value * scale[0]).clamp(0., extent[0] as f32);
        let y = |value: f32| (value * scale[1]).clamp(0., extent[1] as f32);
        let region = [
            x(self.left).ceil() as u32,
            y(self.top).ceil() as u32,
            x(self.right).floor() as u32,
            y(self.bottom).floor() as u32,
        ];
        (region[0] < region[2] && region[1] < region[3]).then_some(region)
    }
}
pub(crate) struct DocumentWorkspaces {
    insert: InsertDialog,
    base_view: base_view_dialog::BaseViewDialog,
    assembly_place: assembly_place_dialog::AssemblyPlaceDialog,
    expecting_presentation: bool,
    last_document: Option<u64>,
}
impl Default for DocumentWorkspaces {
    fn default() -> Self {
        Self {
            insert: InsertDialog::new(),
            base_view: Default::default(),
            assembly_place: Default::default(),
            expecting_presentation: false,
            last_document: None,
        }
    }
}
impl DocumentWorkspaces {
    pub(crate) fn register_dockables(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        self.base_view.register_dockable(registry)?;
        self.assembly_place.register_dockable(registry)
    }
    pub(crate) fn sync_before_docking(
        &mut self,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.base_view.sync_before_docking(state, actions);
        self.assembly_place.sync_before_docking(state, actions);
    }
    pub(crate) fn request_assembly_place(&mut self) {
        self.assembly_place.request();
    }
    pub(crate) fn request_base_view(&mut self) {
        self.base_view.request();
    }
    /// Open the assembly chooser; example: the ribbon's Insert Model button.
    pub(crate) fn request_insert(&mut self) {
        self.insert.request();
    }
    /// A new presentation asks for its model at once, so the chooser's Create is remembered.
    pub(crate) fn note_created(&mut self, actions: &[GuiShellAction]) {
        self.expecting_presentation |= actions.iter().any(|action| {
            let kind = match action {
                GuiShellAction::CreateDocument(template) => template.kind,
                GuiShellAction::CreateProjectTemplate(template) => template.kind,
                _ => return false,
            };
            kind == super::templates::GuiTemplateKind::Presentation
        });
    }
    pub(crate) fn needs_frame(&self) -> bool {
        self.expecting_presentation
            || self.insert.opening
            || self.base_view.needs_frame()
            || self.assembly_place.needs_frame()
    }
    /// Draw the active document's surroundings; example: `workspaces.panel(ui, &state, area)`.
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        area: WorkspaceArea,
        detached: bool,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let created = state.document != self.last_document;
        self.last_document = state.document;
        let presentation = state.document_type == Some(DOCUMENT_PRESENTATION);
        if created && presentation && std::mem::take(&mut self.expecting_presentation) {
            self.insert.request();
        }
        match state.document_type {
            Some(DOCUMENT_DRAWING) if !detached => {
                draw_active_sheet(ui, state, area);
            }
            Some(DOCUMENT_PRESENTATION) => presentation_panels(ui, area),
            _ => {}
        }
        self.insert.panel(ui);
        self.base_view.panel(ui, state, actions);
        self.assembly_place.panel(ui, state, actions);
    }
}
fn active_name(state: &GuiCommandState) -> &str {
    state
        .documents
        .iter()
        .find(|row| Some(row.id) == state.document)
        .map_or("", |row| row.name.as_str())
}

pub(crate) fn draw_active_sheet(ui: &Ui, state: &GuiCommandState, area: WorkspaceArea) -> bool {
    let Some(id) = state.document else {
        return false;
    };
    let Some(drawing) = state.drawing_documents.get(&id) else {
        return false;
    };
    sheet_on_list(
        &ui.get_background_draw_list(),
        active_name(state),
        area,
        drawing,
        state.drawing_projections.get(&id),
    );
    true
}
/// The largest sheet of the template's ratio inside the area, centred.
fn sheet_rectangle(area: WorkspaceArea, dimensions: [f64; 2]) -> ([f32; 2], [f32; 2]) {
    let ratio = dimensions[0] / dimensions[1];
    let available = [
        (area.right - area.left - 2. * SHEET_MARGIN).max(1.),
        (area.bottom - area.top - 2. * SHEET_MARGIN).max(1.),
    ];
    let width = f64::from(available[0]).min(f64::from(available[1]) * ratio);
    let height = width / ratio;
    let width = (width as f32).max(f32::MIN_POSITIVE);
    let height = (height as f32).max(f32::MIN_POSITIVE);
    let minimum = [
        area.left + (area.right - area.left - width) * 0.5,
        area.top + (area.bottom - area.top - height) * 0.5,
    ];
    (minimum, [minimum[0] + width, minimum[1] + height])
}
/// Draw a sheet with room for its bottom tab. Example: when a drawing is detached.
pub(crate) fn draw_sheet_in_window(
    ui: &Ui,
    name: &str,
    drawing: &oblikovati_model::drawing::DrawingDocument,
    projection: Option<&crate::GuiDrawingProjection>,
) {
    let minimum = ui.cursor_screen_pos();
    let [width, available_height] = ui.content_region_avail();
    let tab_height = ui.frame_height_with_spacing();
    let height = (available_height - tab_height).max(0.);
    let area = WorkspaceArea {
        left: minimum[0],
        top: minimum[1],
        right: minimum[0] + width,
        bottom: minimum[1] + height,
    };
    if height > 0. {
        sheet_on_list(&ui.get_window_draw_list(), name, area, drawing, projection);
    }
    ui.dummy([0., height]);
}
fn sheet_on_list(
    list: &imgui::DrawListMut<'_>,
    name: &str,
    area: WorkspaceArea,
    drawing: &oblikovati_model::drawing::DrawingDocument,
    projection: Option<&crate::GuiDrawingProjection>,
) {
    let sheet = &drawing.sheets()[drawing.active_sheet()];
    let Ok(dimensions) = sheet.dimensions_mm() else {
        return;
    };
    let (minimum, maximum) = sheet_rectangle(area, dimensions);
    list.add_rect(minimum, maximum, SHEET_COLOR)
        .filled(true)
        .build();
    if let Some(projection) = projection {
        drawing_lines::draw(list, projection, minimum, maximum, dimensions);
    }
    let (border_minimum, border_maximum) =
        sheet_inner_bounds(minimum, maximum, dimensions, sheet.border_margins_mm());
    if sheet.border {
        list.add_rect(border_minimum, border_maximum, LINE_COLOR)
            .build();
    }
    if sheet.title_block.is_empty() {
        return;
    }
    let (title_minimum, title_maximum) =
        title_bounds(border_minimum, border_maximum, sheet.title_block_at);
    list.add_rect(title_minimum, title_maximum, LINE_COLOR)
        .build();
    let divider = (title_minimum[1] + title_maximum[1]) * 0.5;
    list.add_line(
        [title_minimum[0], divider],
        [title_maximum[0], divider],
        LINE_COLOR,
    )
    .build();
    let inset = title_minimum[0] + 8.;
    list.add_text(
        [inset, title_minimum[1] + 10.],
        LINE_COLOR,
        format!("Title: {name}"),
    );
    list.add_text(
        [inset, divider + 10.],
        LINE_COLOR,
        format!(
            "Scale 1:1   Sheet {} of {}",
            drawing.active_sheet() + 1,
            drawing.sheets().len()
        ),
    );
}
fn title_bounds(
    minimum: [f32; 2],
    maximum: [f32; 2],
    location: oblikovati_api::types::TitleBlockLocation,
) -> ([f32; 2], [f32; 2]) {
    use oblikovati_api::types::{
        BOTTOM_LEFT_TITLE_BLOCK, TOP_LEFT_TITLE_BLOCK, TOP_RIGHT_TITLE_BLOCK,
    };
    let width = TITLE_BLOCK_SIZE[0].min(maximum[0] - minimum[0]);
    let height = TITLE_BLOCK_SIZE[1].min(maximum[1] - minimum[1]);
    let left = matches!(location, BOTTOM_LEFT_TITLE_BLOCK | TOP_LEFT_TITLE_BLOCK);
    let top = matches!(location, TOP_LEFT_TITLE_BLOCK | TOP_RIGHT_TITLE_BLOCK);
    let x = if left { minimum[0] } else { maximum[0] - width };
    let y = if top { minimum[1] } else { maximum[1] - height };
    ([x, y], [x + width, y + height])
}
fn sheet_inner_bounds(
    minimum: [f32; 2],
    maximum: [f32; 2],
    dimensions: [f64; 2],
    margins: Option<[f64; 4]>,
) -> ([f32; 2], [f32; 2]) {
    let Some([left, right, top, bottom]) = margins else {
        return (minimum, maximum);
    };
    let scale_x = f64::from(maximum[0] - minimum[0]) / dimensions[0];
    let scale_y = f64::from(maximum[1] - minimum[1]) / dimensions[1];
    (
        [
            minimum[0] + (left * scale_x) as f32,
            minimum[1] + (top * scale_y) as f32,
        ],
        [
            maximum[0] - (right * scale_x) as f32,
            maximum[1] - (bottom * scale_y) as f32,
        ],
    )
}
fn presentation_panels(ui: &Ui, area: WorkspaceArea) {
    let snapshot_x = area.right - SNAPSHOT_WIDTH;
    fixed_panel(
        ui,
        "Snapshot Views",
        [snapshot_x, area.top],
        [SNAPSHOT_WIDTH, area.bottom - area.top],
        "No snapshots",
    );
    fixed_panel(
        ui,
        "Storyboard",
        [area.left, area.bottom - STORYBOARD_HEIGHT],
        [snapshot_x - area.left - 8., STORYBOARD_HEIGHT],
        "Storyboard1: no actions",
    );
}
fn fixed_panel(ui: &Ui, title: &str, position: [f32; 2], size: [f32; 2], empty: &str) {
    crate::widget_abi::window_geometry(ui, position, size, Condition::Always);
    ui.window(title)
        // `flags` replaces the flags set so far, so it comes first.
        .flags(imgui::WindowFlags::NO_DOCKING | imgui::WindowFlags::NO_FOCUS_ON_APPEARING)
        .movable(false)
        .resizable(false)
        .collapsible(false)
        .build(|| ui.text_disabled(empty));
}
/// Choosing an assembly is real; placing it is not, and the dialog says so instead of pretending.
struct InsertDialog {
    browser: FolderBrowser,
    name: String,
    file_type: usize,
    types: FileTypeList,
    escape: DialogEscape,
    opening: bool,
}
impl InsertDialog {
    fn new() -> Self {
        Self {
            browser: FolderBrowser::new(Rc::new(NativeFolders)),
            name: String::new(),
            file_type: 0,
            types: FileTypeList::assemblies(),
            escape: DialogEscape::default(),
            opening: false,
        }
    }
    fn request(&mut self) {
        self.browser.refresh();
        self.name.clear();
        self.opening = true;
    }
    fn panel(&mut self, ui: &Ui) {
        if std::mem::take(&mut self.opening) {
            ui.open_popup(INSERT);
        }
        crate::application::document_saving::folder_browser::center_dialog(ui, [640., 400.]);
        let Some(_dialog) = ui.begin_modal_popup(INSERT) else {
            return;
        };
        let filter = self.types.filter(self.file_type);
        if let Some(name) = self.browser.controls(ui, "Look in", filter) {
            self.name = name;
        }
        ui.separator();
        file_name_field(ui, &mut self.name);
        file_type_field(ui, &self.types, &mut self.file_type);
        ui.text_disabled("Inserting a model is not available in this build.");
        if dialog_buttons(ui, "Open", false, false, &mut self.escape).is_some() {
            ui.close_current_popup();
        }
    }
}
#[cfg(test)]
#[path = "../../../tests/application/document-creation/workspaces.rs"]
mod tests;
