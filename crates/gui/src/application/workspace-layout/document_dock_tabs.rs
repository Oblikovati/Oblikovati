// SPDX-License-Identifier: GPL-2.0-only
//! Remember the selected tab of the shared Model dock for each open document.
use std::{collections::HashMap, num::NonZeroU32};

#[derive(Debug, Default)]
pub(crate) struct DocumentDockTabs {
    document: Option<u64>,
    selected_tabs: HashMap<u64, NonZeroU32>,
}

impl DocumentDockTabs {
    /// Swap the dock selection before the next ImGui frame lays out the tab bar.
    pub(crate) fn synchronize(&mut self, context: &imgui::Context, state: &crate::GuiCommandState) {
        if self.document == state.document {
            return;
        }
        let outgoing = std::mem::replace(&mut self.document, state.document);
        let incoming = state
            .document
            .and_then(|id| self.selected_tabs.get(&id).copied());
        let selected = crate::widget_abi::exchange_model_dock_tab(context, incoming);
        if let Some((id, tab)) = outgoing.zip(selected) {
            self.selected_tabs.insert(id, tab);
        }
        self.selected_tabs
            .retain(|id, _| state.documents.iter().any(|document| document.id == *id));
    }
}
