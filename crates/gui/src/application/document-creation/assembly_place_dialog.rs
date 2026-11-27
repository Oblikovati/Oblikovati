// SPDX-License-Identifier: GPL-2.0-only
//! Choose an open component for an assembly placement.
use crate::application::document_saving::folder_browser::{
    DialogEnd, DialogEscape, FileTypeList, FolderBrowser, center_dialog, dialog_buttons,
    file_name_field, full_path,
};
use crate::application::file_system::NativeFolders;
use crate::dockable_window::DockableWindow;
use crate::{GuiCommandState, GuiShellAction};
use imgui::{Key, Ui};
use oblikovati_api::{
    types::{DOCUMENT_ASSEMBLY, Matrix},
    wire::PlaceOccurrenceArgs,
};
use std::rc::Rc;

const FILE_CHOOSER: &str = "Choose Component File";

pub(super) struct AssemblyPlaceDialog {
    requested: bool,
    assembly: Option<u64>,
    dock_state: crate::dockable_lifecycle::DockableState,
    source: Option<u64>,
    name: String,
    selected_path: Option<String>,
    browser: FolderBrowser,
    file_name: String,
    file_escape: DialogEscape,
    preview_source: Option<u64>,
    preview_path: Option<String>,
}
impl Default for AssemblyPlaceDialog {
    fn default() -> Self {
        Self {
            requested: false,
            assembly: None,
            dock_state: Default::default(),
            source: None,
            name: String::new(),
            selected_path: None,
            browser: FolderBrowser::new(Rc::new(NativeFolders)),
            file_name: String::new(),
            file_escape: DialogEscape::default(),
            preview_source: None,
            preview_path: None,
        }
    }
}
impl DockableWindow for AssemblyPlaceDialog {}
impl AssemblyPlaceDialog {
    pub(super) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Place Component", &self.dock_state)
    }
    fn set_document(&mut self, document: Option<u64>) {
        self.assembly = document;
        self.dock_state.set_open(document.is_some());
    }
    pub(super) fn request(&mut self) {
        self.requested = true;
    }
    pub(super) fn needs_frame(&self) -> bool {
        self.requested || self.assembly.is_some()
    }
    pub(super) fn sync_before_docking(
        &mut self,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if self.requested {
            return;
        }
        if self.assembly != state.document || state.document_type != Some(DOCUMENT_ASSEMBLY) {
            self.set_document(None);
        }
        if self.assembly.is_none() {
            self.clear_preview(actions);
        }
    }
    pub(super) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if std::mem::take(&mut self.requested) {
            self.set_document(state.document);
            self.source = None;
            self.selected_path = None;
            self.name.clear();
        }
        self.sync_before_docking(state, actions);
        let Some(assembly) = self.assembly else {
            self.clear_preview(actions);
            return;
        };
        Self::fit_dockable(ui, "Place Component", [320., 200.], [440., 210.]);
        let mut open = true;
        Self::dockable_window(ui, "Place Component", &mut open).build(|| {
            self.source_choice(ui, state, assembly);
            ui.same_line();
            if crate::widget_abi::button(ui, "Browse...", [0., 0.]) {
                self.browser.refresh();
                self.file_name.clear();
                ui.open_popup(FILE_CHOOSER);
            }
            self.file_chooser(ui);
            if self.preview_source != self.source || self.preview_path != self.selected_path {
                self.preview_source = self.source;
                self.preview_path = self.selected_path.clone();
                if let Some(path) = &self.preview_path {
                    actions.push(GuiShellAction::PreviewComponentFile {
                        assembly,
                        path: path.clone(),
                    });
                } else {
                    actions.push(GuiShellAction::PreviewComponent(
                        self.source.map(|source| (assembly, source)),
                    ));
                }
            }
            ui.input_text("Occurrence name", &mut self.name).build();
            let action = self.action(assembly);
            let disabled =
                ui.begin_disabled(action.as_ref().is_none_or(|item| !item.enabled(state)));
            if crate::widget_abi::button(ui, "Place", [90., 0.]) {
                self.clear_preview(actions);
                if let Some(action) = action {
                    actions.push(action);
                }
                self.set_document(None);
            }
            drop(disabled);
            ui.same_line();
            if crate::widget_abi::button(ui, "Cancel", [90., 0.]) || ui.is_key_pressed(Key::Escape)
            {
                self.clear_preview(actions);
                self.set_document(None);
            }
        });
        if !open {
            self.clear_preview(actions);
            self.set_document(None);
        }
    }
    fn clear_preview(&mut self, actions: &mut Vec<GuiShellAction>) {
        let source = self.preview_source.take();
        let path = self.preview_path.take();
        if source.is_some() || path.is_some() {
            actions.push(GuiShellAction::PreviewComponent(None));
        }
    }
    fn source_choice(&mut self, ui: &Ui, state: &GuiCommandState, assembly: u64) {
        let shown = self.selected_path.as_deref().unwrap_or_else(|| {
            state
                .documents
                .iter()
                .find(|row| Some(row.id) == self.source)
                .map_or("Choose an open saved component", |row| row.name.as_str())
        });
        let Some(_combo) = ui.begin_combo("Component", shown) else {
            return;
        };
        for row in state.documents.iter().filter(|row| {
            row.id != assembly
                && matches!(row.type_.as_str(), "part" | "assembly")
                && state
                    .document_storage
                    .get(&row.id)
                    .is_some_and(|file| file.path.is_some())
        }) {
            if ui
                .selectable_config(&row.name)
                .selected(Some(row.id) == self.source)
                .build()
            {
                self.source = Some(row.id);
                self.selected_path = None;
                self.name = format!("{}:1", row.name);
            }
        }
    }
    fn file_chooser(&mut self, ui: &Ui) {
        center_dialog(ui, [800., 480.]);
        let Some(_chooser) = ui.begin_modal_popup(FILE_CHOOSER) else {
            return;
        };
        let types = FileTypeList::any();
        if let Some(name) = self.browser.controls(ui, "Look in", types.filter(0)) {
            self.file_name = name;
        }
        let entered = file_name_field(ui, &mut self.file_name);
        let end = dialog_buttons(
            ui,
            "Open",
            !self.file_name.is_empty(),
            entered,
            &mut self.file_escape,
        );
        if end == Some(DialogEnd::Accepted) {
            self.selected_path = Some(full_path(&self.browser.folder, &self.file_name));
            self.source = None;
            self.name = format!("{}:1", self.file_name);
        }
        if end.is_some() {
            ui.close_current_popup();
        }
    }
    fn action(&self, assembly: u64) -> Option<GuiShellAction> {
        if let Some(path) = &self.selected_path {
            return Some(GuiShellAction::PlaceComponentFile {
                assembly,
                path: path.clone(),
                name: self.name.clone(),
            });
        }
        Some(GuiShellAction::PlaceComponent {
            assembly,
            request: PlaceOccurrenceArgs {
                document: self.source?,
                name: self.name.clone(),
                transform: Matrix {
                    cells: [
                        1., 0., 0., 0., 0., 1., 0., 0., 0., 0., 1., 0., 0., 0., 0., 1.,
                    ],
                },
            },
        })
    }
}

#[cfg(test)]
mod dock_tests {
    use super::*;
    #[test]
    fn pre_dock_sync_does_not_consume_a_new_request() {
        let mut dialog = AssemblyPlaceDialog::default();
        dialog.request();
        dialog.sync_before_docking(&GuiCommandState::default(), &mut Vec::new());
        assert!(dialog.requested);
    }
    #[test]
    fn document_change_closes_placement_and_clears_preview_once() {
        let mut dialog = AssemblyPlaceDialog::default();
        dialog.set_document(Some(3));
        dialog.preview_source = Some(4);
        let mut actions = Vec::new();
        dialog.sync_before_docking(&GuiCommandState::default(), &mut actions);
        dialog.sync_before_docking(&GuiCommandState::default(), &mut actions);
        assert!(dialog.assembly.is_none());
        assert_eq!(actions, [GuiShellAction::PreviewComponent(None)]);
    }
}
