// SPDX-License-Identifier: GPL-2.0-only
//! Draft base-view settings until the user applies or cancels them.
use crate::dockable_window::DockableWindow;
use crate::{GuiBaseViewRequest, GuiCommandState, GuiShellAction};
use imgui::{Key, Ui};

const ORIENTATIONS: [(&str, &str); 7] = [
    ("Front", "front"),
    ("Back", "back"),
    ("Top", "top"),
    ("Bottom", "bottom"),
    ("Left", "left"),
    ("Right", "right"),
    ("Isometric", "iso"),
];
#[derive(Default)]
pub(super) struct BaseViewDialog {
    requested: bool,
    drawing: Option<u64>,
    dock_state: crate::dockable_lifecycle::DockableState,
    source: Option<u64>,
    orientation: usize,
    scale: f32,
    center: [f32; 2],
    previewed: bool,
}
impl DockableWindow for BaseViewDialog {}
impl BaseViewDialog {
    pub(super) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Base View", &self.dock_state)
    }
    fn set_document(&mut self, document: Option<u64>) {
        self.drawing = document;
        self.dock_state.set_open(document.is_some());
    }
    pub(super) fn request(&mut self) {
        self.requested = true;
    }
    pub(super) fn needs_frame(&self) -> bool {
        self.requested
    }
    pub(super) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if self.requested && self.previewed {
            actions.push(GuiShellAction::PreviewDrawingBaseView(None));
            self.previewed = false;
        }
        if std::mem::take(&mut self.requested) {
            self.open(state);
        }
        if self.drawing != state.document
            || state.document_type != Some(oblikovati_api::types::DOCUMENT_DRAWING)
        {
            self.set_document(None);
        }
        if self.drawing.is_none() && self.previewed {
            actions.push(GuiShellAction::PreviewDrawingBaseView(None));
            self.previewed = false;
        }
        let Some(drawing) = self.drawing else { return };
        crate::widget_abi::fit_dialog(ui, "Base View", [20., 180.], [360., 210.]);
        let mut open = true;
        Self::dockable_window(ui, "Base View", &mut open).build(|| {
            let mut edited = self.source_choice(ui, state);
            let labels = ORIENTATIONS.map(|(label, _)| label);
            edited |= ui.combo_simple_string("Orientation", &mut self.orientation, &labels);
            edited |= ui.input_float("Scale", &mut self.scale).build();
            edited |= ui.input_float2("Center (mm)", &mut self.center).build();
            if edited && self.previewed {
                actions.push(GuiShellAction::PreviewDrawingBaseView(None));
                self.previewed = false;
            }
            self.controls(ui, state, drawing, actions);
        });
        if !open {
            if self.previewed {
                actions.push(GuiShellAction::PreviewDrawingBaseView(None));
                self.previewed = false;
            }
            self.set_document(None);
        }
    }
    fn open(&mut self, state: &GuiCommandState) {
        self.set_document(state.document);
        self.source = preferred_source(state);
        self.orientation = 0;
        self.scale = 1.;
        self.center = state
            .document
            .and_then(|id| state.drawing_documents.get(&id))
            .and_then(|drawing| {
                drawing.sheets()[drawing.active_sheet()]
                    .dimensions_mm()
                    .ok()
            })
            .map_or([0., 0.], |size| {
                [(size[0] / 2.) as f32, (size[1] / 2.) as f32]
            });
    }
    fn source_choice(&mut self, ui: &Ui, state: &GuiCommandState) -> bool {
        let shown = state
            .documents
            .iter()
            .find(|row| Some(row.id) == self.source && row.type_ == "part")
            .map_or("Choose an open part", |row| row.name.as_str());
        let Some(_combo) = ui.begin_combo("Model", shown) else {
            return false;
        };
        let previous = self.source;
        for row in state.documents.iter().filter(|row| row.type_ == "part") {
            if ui
                .selectable_config(&row.name)
                .selected(Some(row.id) == self.source)
                .build()
            {
                self.source = Some(row.id);
            }
        }
        previous != self.source
    }
    fn controls(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        drawing: u64,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let action = self.source.map(|source| {
            GuiShellAction::CreateDrawingBaseView(GuiBaseViewRequest {
                drawing,
                source,
                orientation: ORIENTATIONS[self.orientation].1.into(),
                scale: f64::from(self.scale),
                center: self.center.map(f64::from),
            })
        });
        let enabled = action
            .as_ref()
            .is_some_and(|action| action.dispatchable(state));
        let disabled = ui.begin_disabled(!enabled);
        if crate::widget_abi::button(ui, "Preview", [0.; 2])
            && enabled
            && let Some(GuiShellAction::CreateDrawingBaseView(request)) = action.clone()
        {
            actions.push(GuiShellAction::PreviewDrawingBaseView(Some(request)));
            self.previewed = true;
        }
        ui.same_line();
        let apply = crate::widget_abi::button(ui, "Apply", [0.; 2]);
        if enabled && (apply || ui.is_window_focused() && ui.is_key_pressed_no_repeat(Key::Enter)) {
            actions.extend(action);
            self.set_document(None);
            self.previewed = false;
        }
        drop(disabled);
        ui.same_line();
        if crate::widget_abi::button(ui, "Cancel", [0.; 2])
            || ui.is_window_focused() && ui.is_key_pressed_no_repeat(Key::Escape)
        {
            if self.previewed {
                actions.push(GuiShellAction::PreviewDrawingBaseView(None));
                self.previewed = false;
            }
            self.set_document(None);
        }
    }
}
fn preferred_source(state: &GuiCommandState) -> Option<u64> {
    let reference = state
        .document
        .and_then(|id| state.drawing_documents.get(&id))
        .map(|drawing| drawing.model_reference())
        .unwrap_or("");
    let parts = || state.documents.iter().filter(|row| row.type_ == "part");
    parts()
        .find(|row| {
            state
                .document_storage
                .get(&row.id)
                .and_then(|storage| storage.path.as_deref())
                .unwrap_or(&row.name)
                == reference
        })
        .or_else(|| parts().next())
        .map(|row| row.id)
}

#[cfg(test)]
mod tests {
    use super::*;
    use oblikovati_api::wire::DocumentInfo;

    #[test]
    fn existing_reference_selects_its_open_part() {
        let mut state = GuiCommandState {
            document: Some(3),
            ..Default::default()
        };
        state.documents = vec![
            DocumentInfo {
                id: 1,
                name: "First".into(),
                type_: "part".into(),
                ..Default::default()
            },
            DocumentInfo {
                id: 2,
                name: "Second".into(),
                type_: "part".into(),
                ..Default::default()
            },
        ];
        let mut drawing = oblikovati_model::drawing::DrawingDocument::new();
        drawing.set_model_reference("/parts/second.opd").unwrap();
        state.drawing_documents.insert(3, drawing);
        state.document_storage.insert(
            2,
            crate::GuiDocumentStorage {
                path: Some("/parts/second.opd".into()),
                unsaved_edits: false,
            },
        );
        assert_eq!(preferred_source(&state), Some(2));
        state.documents.retain(|row| row.id != 2);
        assert_eq!(preferred_source(&state), Some(1));
        state.documents.clear();
        assert_eq!(preferred_source(&state), None);
    }
}
