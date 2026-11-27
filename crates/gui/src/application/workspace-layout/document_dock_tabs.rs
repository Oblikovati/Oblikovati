// SPDX-License-Identifier: GPL-2.0-only
//! Remember the selected tab of the shared Model dock for each open document.
use std::{collections::HashMap, num::NonZeroU32};

#[derive(Debug, Default)]
pub(crate) struct DocumentDockTabs {
    document: Option<u64>,
    selected_tabs: HashMap<u64, NonZeroU32>,
    pending_tab: Option<NonZeroU32>,
}

impl DocumentDockTabs {
    /// Update the selection before ImGui lays out the dock. Return true when the dock
    /// needs a frame to reappear before it can receive the stored tab.
    pub(crate) fn synchronize(
        &mut self,
        context: &imgui::Context,
        state: &crate::GuiCommandState,
        home_active: bool,
    ) -> bool {
        let document = state.document.filter(|_| !home_active);
        let pending = self.pending_tab.take();
        if self.document == document {
            if pending.is_some() {
                crate::widget_abi::exchange_model_dock_tab(context, pending);
            }
            return false;
        }
        self.switch_document(context, state, document, pending.is_some())
    }

    fn switch_document(
        &mut self,
        context: &imgui::Context,
        state: &crate::GuiCommandState,
        document: Option<u64>,
        outgoing_restore_pending: bool,
    ) -> bool {
        let outgoing = std::mem::replace(&mut self.document, document);
        let incoming = document.and_then(|id| self.selected_tabs.get(&id).copied());
        let selected = crate::widget_abi::exchange_model_dock_tab(context, incoming);
        if !outgoing_restore_pending && let Some((id, tab)) = outgoing.zip(selected) {
            self.selected_tabs.insert(id, tab);
        }
        self.selected_tabs
            .retain(|id, _| state.documents.iter().any(|document| document.id == *id));
        // Immediately after leaving Home, Model may still be inactive in the previous frame.
        self.pending_tab = incoming.filter(|_| selected.is_none());
        self.pending_tab.is_some()
    }
}
