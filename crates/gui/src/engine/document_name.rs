// SPDX-License-Identifier: GPL-2.0-only
//! Resolve the active document name for desktop dialogs.
use super::GuiFrameEngine;

impl GuiFrameEngine {
    pub(super) fn document_name(&self, document: u64) -> String {
        self.known_documents
            .iter()
            .find(|row| row.id == document)
            .map_or_else(|| "Part".to_string(), |row| row.name.clone())
    }
}
