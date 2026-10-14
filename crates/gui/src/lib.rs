// SPDX-License-Identifier: GPL-2.0-only
//! Owned Dear ImGui frames for the application viewport draw boundary.
mod activity;
mod clipboard;
pub use activity::GuiFrameActivity;
pub use clipboard::GuiClipboard;
mod commands;
#[allow(unsafe_code)]
mod context_owner;
mod engine;
mod input;
mod snapshot;
pub use commands::{GuiCommandState, GuiShellAction};
pub use engine::{GuiFrameEngine, GuiFrameOutput, GuiFrameRequest};
pub use input::{GuiInputCapture, GuiInputEvent, GuiKey, GuiModifiers, GuiPointerButton};
pub use oblikovati_viewport::{GuiAtlas, GuiBatch, GuiFrame, GuiVertex};
