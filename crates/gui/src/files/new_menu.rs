// SPDX-License-Identifier: GPL-2.0-only
//! Open a document-kind template from the File menu without creating it prematurely.
use super::GuiDocumentFiles;
use crate::{GuiCommandState, application::document_creation::templates::GuiTemplateKind};
use imgui::Ui;

const NEW_MENU: &str = "##file-new-types";
impl GuiDocumentFiles {
    pub(super) fn new_menu_item(&mut self, ui: &Ui, state: &GuiCommandState) {
        let selected = ui
            .menu_item_config("New")
            .enabled(!state.transaction_open)
            .build();
        let hover = ui.is_item_hovered() && !state.transaction_open;
        let position = [ui.item_rect_max()[0], ui.item_rect_min()[1]];
        let new_bottom = ui.item_rect_max()[1];
        let parent = [ui.window_pos(), ui.window_size()];
        if selected {
            self.templates.request();
            return;
        }
        if hover {
            ui.open_popup(NEW_MENU);
        }
        crate::widget_abi::next_window_position(ui, position);
        let Some(_menu) = ui.begin_popup(NEW_MENU) else {
            return;
        };
        if pointer_on_other_file_command(ui, parent, position, new_bottom) {
            ui.close_current_popup();
            return;
        }
        self.new_menu_choices(ui);
    }
    fn new_menu_choices(&mut self, ui: &Ui) {
        ui.text("Create a file from the available templates");
        ui.separator();
        self.new_kind(ui, "Assembly", GuiTemplateKind::Assembly);
        super::unavailable_file_item(
            ui,
            "Weldment",
            "Weldment templates are not available in this build.",
        );
        ui.menu("Drawing", || {
            self.new_kind(ui, "Choose drawing template", GuiTemplateKind::Drawing);
        });
        self.new_kind(ui, "Part", GuiTemplateKind::Part);
        self.new_kind(ui, "Sheet Metal", GuiTemplateKind::SheetMetal);
        self.new_kind(ui, "Presentation", GuiTemplateKind::Presentation);
    }
    fn new_kind(&mut self, ui: &Ui, label: &str, kind: GuiTemplateKind) {
        if ui.menu_item(label) {
            self.templates.request_kind(kind);
        }
    }
}

fn pointer_on_other_file_command(
    ui: &Ui,
    parent: [[f32; 2]; 2],
    new_position: [f32; 2],
    new_bottom: f32,
) -> bool {
    let [x, y] = ui.io().mouse_pos;
    let [origin, size] = parent;
    let on_commands =
        x >= origin[0] && x < new_position[0] && y >= origin[1] && y < origin[1] + size[1];
    on_commands && !(new_position[1]..=new_bottom).contains(&y)
}
