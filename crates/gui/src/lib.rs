// SPDX-License-Identifier: GPL-2.0-only
//! Owned Dear ImGui frames for the application viewport draw boundary.
mod engine;
mod snapshot;
pub use engine::{GuiFrameEngine, GuiFrameRequest, GuiShellAction};
pub use oblikovati_viewport::{GuiAtlas, GuiBatch, GuiFrame, GuiVertex};
