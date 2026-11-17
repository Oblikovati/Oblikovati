// SPDX-License-Identifier: GPL-2.0-only
use crate::{GuiCommandState, GuiParameterAction, GuiParameterSnapshot};
use imgui::Ui;

pub(crate) struct GuiParameterEditor {
    document: Option<u64>,
    panel: crate::parameter_panel::ParameterPanel,
}
impl GuiParameterEditor {
    pub(crate) fn register_dockables(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        self.panel.register_dockables(registry)
    }
    pub(crate) fn new() -> Result<Self, String> {
        Ok(Self {
            document: None,
            panel: crate::parameter_panel::ParameterPanel::new()?,
        })
    }
    pub(crate) fn open(&mut self, document: Option<u64>) {
        self.panel
            .set_open(document.is_some() && document == self.document);
    }
    pub(crate) fn toolbar(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        icons: &crate::icons::GuiRibbonIcons,
    ) -> bool {
        let disabled = ui.begin_disabled(state.document.is_none());
        let clicked = icons.button(
            ui,
            "Parameters",
            crate::icons::IconGlyph::Parameters,
            icons.appearance.single_cell(),
        );
        if clicked {
            self.panel.set_open(!self.panel.is_open());
        }
        drop(disabled);
        clicked
    }
    pub(crate) fn panel(&mut self, ui: &Ui) -> Vec<GuiParameterAction> {
        self.panel.draw(ui)
    }
    pub(crate) fn synchronize_document(&mut self, document: Option<u64>) -> Result<(), String> {
        if self.document == document {
            return Ok(());
        }
        self.panel.replace_snapshot(None)?;
        self.document = document;
        if document.is_none() {
            self.panel.set_open(false);
        }
        Ok(())
    }
    pub(crate) fn set_open(&mut self, open: bool) {
        self.panel.set_open(open);
    }
    pub(crate) fn is_open(&self) -> bool {
        self.panel.is_open()
    }
    pub(crate) fn set_snapshot(
        &mut self,
        snapshot: Option<GuiParameterSnapshot>,
    ) -> Result<bool, String> {
        if let Some(view) = &snapshot
            && Some(view.document) != self.document
        {
            return Err(format!(
                "parameter view document {}: expected active document {:?}",
                view.document, self.document
            ));
        }
        self.panel.replace_snapshot(snapshot)
    }
    pub(crate) fn report_issue(&mut self, issue: String) -> Result<bool, String> {
        self.panel.report_metadata_issue(issue)
    }
    pub(crate) fn acknowledge(&mut self, action: &GuiParameterAction, result: Result<(), String>) {
        self.panel.acknowledge_action(action, result);
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn snapshot(document: u64) -> GuiParameterSnapshot {
        GuiParameterSnapshot {
            document,
            rows: Vec::new(),
            linked_tables: Vec::new(),
            link_sources: Vec::new(),
            groups: Vec::new(),
        }
    }

    #[test]
    fn document_change_preserves_open_editor_and_retires_the_previous_parameter_view() {
        let mut editor = GuiParameterEditor::new().unwrap();
        editor.synchronize_document(Some(1)).unwrap();
        assert!(editor.set_snapshot(Some(snapshot(1))).unwrap());
        editor.open(Some(1));
        assert!(editor.is_open());
        editor.synchronize_document(Some(2)).unwrap();
        assert!(editor.is_open());
        assert!(editor.set_snapshot(Some(snapshot(1))).is_err());
        assert!(editor.set_snapshot(Some(snapshot(2))).unwrap());
        editor.open(Some(1));
        assert!(!editor.is_open());
    }

    #[test]
    fn closing_the_last_document_closes_the_parameter_editor() {
        let mut editor = GuiParameterEditor::new().unwrap();
        editor.synchronize_document(Some(1)).unwrap();
        editor.open(Some(1));
        editor.synchronize_document(None).unwrap();
        assert!(!editor.is_open());
    }
}
