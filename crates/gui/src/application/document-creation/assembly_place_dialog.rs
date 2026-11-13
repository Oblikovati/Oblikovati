// SPDX-License-Identifier: GPL-2.0-only
//! Choose an open component for an assembly placement.
use crate::dockable_window::DockableWindow;
use crate::{GuiCommandState, GuiShellAction};
use imgui::{Key, Ui};
use oblikovati_api::{
    types::{DOCUMENT_ASSEMBLY, Matrix},
    wire::PlaceOccurrenceArgs,
};

#[derive(Default)]
pub(super) struct AssemblyPlaceDialog {
    requested: bool,
    assembly: Option<u64>,
    source: Option<u64>,
    name: String,
}
impl DockableWindow for AssemblyPlaceDialog {}
impl AssemblyPlaceDialog {
    pub(super) fn request(&mut self) {
        self.requested = true;
    }
    pub(super) fn needs_frame(&self) -> bool {
        self.requested || self.assembly.is_some()
    }
    pub(super) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if std::mem::take(&mut self.requested) {
            self.assembly = state.document;
            self.source = None;
            self.name.clear();
        }
        if self.assembly != state.document || state.document_type != Some(DOCUMENT_ASSEMBLY) {
            self.assembly = None;
        }
        let Some(assembly) = self.assembly else {
            return;
        };
        Self::fit_dockable(ui, "Place Component", [420., 200.], [380., 190.]);
        let mut open = true;
        Self::dockable_window(ui, "Place Component", &mut open).build(|| {
            self.source_choice(ui, state, assembly);
            ui.input_text("Occurrence name", &mut self.name).build();
            let action = self.action(assembly);
            let disabled =
                ui.begin_disabled(action.as_ref().is_none_or(|item| !item.enabled(state)));
            if crate::widget_abi::button(ui, "Place", [90., 0.]) {
                if let Some(action) = action {
                    actions.push(action);
                }
                self.assembly = None;
            }
            drop(disabled);
            ui.same_line();
            if crate::widget_abi::button(ui, "Cancel", [90., 0.]) || ui.is_key_pressed(Key::Escape)
            {
                self.assembly = None;
            }
        });
        if !open {
            self.assembly = None;
        }
    }
    fn source_choice(&mut self, ui: &Ui, state: &GuiCommandState, assembly: u64) {
        let shown = state
            .documents
            .iter()
            .find(|row| Some(row.id) == self.source)
            .map_or("Choose an open saved component", |row| row.name.as_str());
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
                self.name = format!("{}:1", row.name);
            }
        }
    }
    fn action(&self, assembly: u64) -> Option<GuiShellAction> {
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
