// SPDX-License-Identifier: GPL-2.0-only
//! Owned Linux viewports over the published Martensite device and presentation path.
mod input;
pub use input::{
    ViewportButton, ViewportInput, ViewportKey, ViewportModifiers, ViewportNavigation,
};
mod presentation;
pub use presentation::{PartPresentation, PartPresentationDecline};
mod types;
pub use types::{
    ViewportCamera, ViewportCapture, ViewportEvent, ViewportOptions, ViewportPresentation,
    ViewportScene, ViewportTriangle,
};
#[cfg(target_os = "linux")]
mod renderer;
#[cfg(target_os = "linux")]
mod runtime;
#[cfg(target_os = "linux")]
mod scene;
#[cfg(target_os = "linux")]
mod target;
#[cfg(target_os = "linux")]
#[allow(unsafe_code)]
mod x11;
#[cfg(target_os = "linux")]
pub use runtime::{Viewport, ViewportRuntime};

#[cfg(target_os = "linux")]
mod wake;
#[cfg(target_os = "linux")]
pub use wake::{ViewportReady, ViewportWake};

#[path = "gui/packet.rs"]
mod gui_packet;
pub use gui_packet::{GuiAtlas, GuiBatch, GuiFrame, GuiVertex};
#[cfg(target_os = "linux")]
#[path = "gui/render.rs"]
mod gui_render;
#[cfg(target_os = "linux")]
pub use gui_render::{GuiFrameCompiler, compile_gui_frame};

mod sketch_preview;
pub use sketch_preview::ViewportSketchPreview;

mod sketch_camera;

mod sketch_display;
pub use sketch_display::{ViewportSketchDecline, ViewportSketchDisplay};
