// SPDX-License-Identifier: GPL-2.0-only
//! Owned Dear ImGui frames for the application viewport draw boundary.
mod activity;
mod clipboard;
pub use activity::GuiFrameActivity;
pub use clipboard::GuiClipboard;
mod commands;
mod parameter_snapshot;
pub use parameter_snapshot::{
    GuiParameterGroup, GuiParameterLinkParameter, GuiParameterLinkSource, GuiParameterRow,
    GuiParameterSnapshot,
};
mod parameter_drafts;
mod parameter_panel;
pub use parameter_drafts::{
    GuiParameterAck, GuiParameterDrafts, GuiParameterTextEdit, GuiParameterTextField,
    GuiParameterTextTarget,
};
pub use parameter_panel::{
    GuiParameterAction, GuiParameterChoiceEdit, GuiParameterCreation, GuiParameterGroupAction,
    GuiParameterGroupEdit, GuiParameterInitialValue, GuiParameterLinkEdit,
    GuiParameterRowOperation, GuiParameterToggleField, GuiParameterToleranceEdit,
};
#[allow(unsafe_code)]
mod context_owner;
mod engine;
mod icon_assets;
mod input;
mod snapshot;
pub use commands::{
    GuiCommandState, GuiDocumentHistory, GuiDocumentTab, GuiDocumentUnits, GuiHistoryEntry,
    GuiModelBody, GuiModelFeature, GuiModelTree, GuiRibbonControl, GuiRibbonTab,
    GuiSelectionFilter, GuiSelectionFilterEntry, GuiShellAction, GuiSketchPreferences,
};
pub use engine::{GuiFrameEngine, GuiFrameOutput, GuiFrameRequest};
pub use input::{GuiInputCapture, GuiInputEvent, GuiKey, GuiModifiers, GuiPointerButton};
pub use oblikovati_viewport::{GuiAtlas, GuiBatch, GuiFrame, GuiTexture, GuiVertex};
