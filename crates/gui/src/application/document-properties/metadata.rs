// SPDX-License-Identifier: GPL-2.0-only
//! Document metadata, edited through the document owner's typed property API.
use crate::{GuiCommandState, GuiShellAction, dockable_window::DockableWindow};
use imgui::Ui;
use oblikovati_api::{
    types::{Variant, VariantValue},
    wire::SetPropertyArgs,
};

#[derive(Default)]
pub(crate) struct MetadataEditor {
    document: Option<u64>,
    set: String,
    name: String,
    value: String,
}
impl DockableWindow for MetadataEditor {}
impl MetadataEditor {
    pub(crate) fn open(&mut self, state: &GuiCommandState) {
        self.document = state.document;
        self.set = "Summary Information".into();
        self.name.clear();
        self.value.clear();
    }
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some(document) = self.document else {
            return;
        };
        if state.document != Some(document) {
            self.document = None;
            return;
        }
        let mut open = true;
        Self::dockable_window(ui, "iProperties", &mut open)
            .build(|| self.contents(ui, state, document, actions));
        if !open {
            self.document = None;
        }
    }
    fn contents(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        document: u64,
        actions: &mut Vec<GuiShellAction>,
    ) {
        ui.text("Document properties");
        for property in &state.document_properties {
            ui.text(format!(
                "{} / {}: {}",
                property.set,
                property.name,
                property_text(&property.value.value)
            ));
        }
        ui.separator();
        self.text_property_form(ui, state, document, actions);
    }
    fn text_property_form(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        document: u64,
        actions: &mut Vec<GuiShellAction>,
    ) {
        ui.text("Add or update a text property");
        ui.input_text("Property set", &mut self.set).build();
        ui.input_text("Name", &mut self.name).build();
        ui.input_text("Value", &mut self.value).build();
        let disabled = ui.begin_disabled(!self.can_set(state));
        if ui.button("Set property") {
            actions.push(self.set_action(document));
        }
        drop(disabled);
        if let Some(reason) = self.block_reason(state) {
            ui.text_wrapped(reason);
        }
    }
    fn can_set(&self, state: &GuiCommandState) -> bool {
        self.block_reason(state).is_none()
    }
    fn block_reason(&self, state: &GuiCommandState) -> Option<&'static str> {
        if self.set.is_empty() || self.name.is_empty() {
            return Some("Enter a property set and name.");
        }
        if state.transaction_open {
            return Some("Finish the active transaction before editing properties.");
        }
        let row = state
            .document_properties
            .iter()
            .find(|row| row.set == self.set && row.name == self.name)?;
        if !row.from_parameter.is_empty() {
            return Some("Edit this property through its parameter.");
        }
        if !matches!(row.value.value, VariantValue::String(_)) {
            return Some("This editor cannot replace a non-text property.");
        }
        None
    }
    fn set_action(&self, document: u64) -> GuiShellAction {
        GuiShellAction::SetDocumentProperty(SetPropertyArgs {
            document,
            set: self.set.clone(),
            name: self.name.clone(),
            value: Variant {
                value: VariantValue::String(self.value.clone()),
                unit: String::new(),
            },
        })
    }
}

fn property_text(value: &VariantValue) -> String {
    match value {
        VariantValue::String(value) => value.clone(),
        VariantValue::Integer(value) => value.to_string(),
        VariantValue::Double(value) => value.to_string(),
        VariantValue::Boolean(value) => value.to_string(),
        VariantValue::Bytes(value) => format!("{} bytes", value.0.as_ref().map_or(0, Vec::len)),
        VariantValue::Uninitialized => "Uninitialized".into(),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use oblikovati_api::wire::PropertyInfo;

    fn title_editor() -> (MetadataEditor, GuiCommandState) {
        let mut editor = MetadataEditor::default();
        let state = GuiCommandState {
            document: Some(4),
            ..Default::default()
        };
        editor.open(&state);
        editor.name = "Title".into();
        editor.value = "Motor".into();
        (editor, state)
    }
    fn existing_title(editor: &MetadataEditor, value: VariantValue) -> PropertyInfo {
        PropertyInfo {
            set: editor.set.clone(),
            name: editor.name.clone(),
            value: Variant {
                value,
                unit: String::new(),
            },
            ..Default::default()
        }
    }
    #[test]
    fn property_values_are_presented_without_debug_wrappers() {
        assert_eq!(
            property_text(&VariantValue::String("Motor".into())),
            "Motor"
        );
        assert_eq!(property_text(&VariantValue::Integer(3)), "3");
    }
    #[test]
    fn text_edit_creates_typed_owner_action() {
        let (editor, state) = title_editor();
        assert!(editor.can_set(&state));
        assert!(
            matches!(editor.set_action(4), GuiShellAction::SetDocumentProperty(request) if request.name == "Title" && request.value.value == VariantValue::String("Motor".into()))
        );
    }
    #[test]
    fn non_text_and_parameter_owned_properties_cannot_be_replaced() {
        let (editor, mut state) = title_editor();
        state
            .document_properties
            .push(existing_title(&editor, VariantValue::Integer(4)));
        assert!(!editor.can_set(&state));
        state.document_properties[0].value.value = VariantValue::String("old".into());
        state.document_properties[0].from_parameter = "title".into();
        assert!(!editor.can_set(&state));
    }
}
