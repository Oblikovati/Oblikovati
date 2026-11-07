// SPDX-License-Identifier: GPL-2.0-only
//! Tabs are windows of documents: one per document until a person opens a second.
use crate::{GuiCommandState, GuiShellAction};
use oblikovati_api::wire::{ActivateDocumentArgs, DocumentInfo};

/// One extra view of a document; a document with a single view lists nothing here.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct GuiDocumentWindow {
    pub id: u64,
    pub document: u64,
    pub number: u32,
}
/// One tab: its document, its window, and its label.
#[derive(Clone, Debug, PartialEq)]
pub(crate) struct TabRow<'a> {
    pub row: &'a DocumentInfo,
    pub window: u64,
    /// A number only while a document has more than one window.
    pub name: String,
    pub numbered: bool,
}
/// Every open window in strip order.
pub(crate) fn tab_rows(state: &GuiCommandState) -> Vec<TabRow<'_>> {
    let mut rows = Vec::new();
    for row in state.documents.iter().filter(|row| row.visible) {
        let windows: Vec<&GuiDocumentWindow> = state
            .document_windows
            .iter()
            .filter(|window| window.document == row.id)
            .collect();
        if windows.len() < 2 {
            let window = windows.first().map_or(row.id, |window| window.id);
            rows.push(single_row(row, window, state));
            continue;
        }
        rows.extend(
            windows
                .into_iter()
                .map(|window| numbered_row(row, window, state)),
        );
    }
    rows
}
fn single_row<'a>(row: &'a DocumentInfo, window: u64, state: &GuiCommandState) -> TabRow<'a> {
    TabRow {
        row,
        window,
        name: document_caption(&row.name, state.document_model_states.get(&row.id)),
        numbered: false,
    }
}
fn numbered_row<'a>(
    row: &'a DocumentInfo,
    window: &GuiDocumentWindow,
    state: &GuiCommandState,
) -> TabRow<'a> {
    TabRow {
        row,
        window: window.id,
        name: document_caption(
            &format!("{}:{}", row.name, window.number),
            state.document_model_states.get(&row.id),
        ),
        numbered: true,
    }
}
pub(crate) fn document_caption(name: &str, member: Option<&String>) -> String {
    member.map_or_else(|| name.to_owned(), |member| format!("{name} <{member}>"))
}
/// True when this tab is what the workspace shows.
pub(crate) fn is_active_window(state: &GuiCommandState, tab: &TabRow<'_>) -> bool {
    state.document == Some(tab.row.id) && (!tab.numbered || state.active_window == Some(tab.window))
}
/// The name of the window in front; example: `Part1:2` for a part's second window.
pub fn active_window_name(state: &GuiCommandState) -> Option<String> {
    tab_rows(state)
        .into_iter()
        .find(|tab| is_active_window(state, tab))
        .map(|tab| tab.name)
}
/// What choosing this tab asks of the owner; nothing when it is already in front.
pub(crate) fn activation(state: &GuiCommandState, tab: &TabRow<'_>) -> Option<GuiShellAction> {
    if is_active_window(state, tab) {
        return None;
    }
    Some(if tab.numbered {
        GuiShellAction::ActivateWindow(tab.window)
    } else {
        GuiShellAction::ActivateDocument(ActivateDocumentArgs { id: tab.row.id })
    })
}

#[cfg(test)]
#[path = "../../../tests/application/document-tabs/windows.rs"]
mod tests;
