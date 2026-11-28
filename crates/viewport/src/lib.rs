// SPDX-License-Identifier: GPL-2.0-only
//! Owned desktop viewports over the published Martensite device and presentation path.
mod background;
pub use background::ViewportBackground;
mod clipboard;
mod window_frame;
pub use window_frame::ViewportWindowAction;
mod input;
pub use clipboard::ViewportClipboard;
#[cfg(target_os = "linux")]
pub use clipboard::{NativeClipboardEndpoint, NativeClipboardHandoff, NativeClipboardOptions};
#[cfg(any(
    target_os = "linux",
    target_os = "windows",
    target_arch = "wasm32",
    test
))]
mod fit_region;
mod keyboard;
pub use input::{
    ViewportButton, ViewportInput, ViewportKey, ViewportModifiers, ViewportNavigation,
};
pub use keyboard::ViewportPhysicalKey;
pub use oblikovati_model::viewport_keyboard_keys;
mod presentation;
mod scene_placement;
pub use presentation::{PartPresentation, PartPresentationDecline};
mod datum_axis;
mod datum_plane;
#[path = "scene/normal.rs"]
mod render_normal;
pub use datum_axis::ViewportDatumAxis;
pub use datum_plane::ViewportDatumPlane;
mod types;
pub use types::{
    ViewportCamera, ViewportCapture, ViewportEvent, ViewportGuiSubmission, ViewportOptions,
    ViewportPresentation, ViewportPreviewPixels, ViewportScene, ViewportTriangle,
};
mod web_input;
pub use web_input::{
    WebKeyEvent, WebModifiers, WebPointerEvent, WebPointerKind, WebResizeEvent, WebWheelEvent,
    translate_dom_code, translate_web_key, translate_web_pointer, translate_web_resize,
    translate_web_wheel,
};
mod web_clipboard;
pub use web_clipboard::WebClipboard;
#[cfg(target_os = "linux")]
mod linux;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod renderer;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod runtime;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod scene;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod target;
#[cfg(target_os = "linux")]
mod wayland;
#[cfg(target_os = "linux")]
pub use wayland::clipboard::WaylandClipboard;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
pub mod web;
#[cfg(target_os = "windows")]
#[allow(unsafe_code)]
mod win32;
#[cfg(target_os = "linux")]
#[allow(unsafe_code)]
mod x11;
#[cfg(target_os = "linux")]
use linux as native;
#[cfg(target_arch = "wasm32")]
use web as native;
#[cfg(target_os = "windows")]
use win32 as native;
#[cfg(target_os = "windows")]
pub use win32::WindowsClipboard;
#[cfg(target_os = "windows")]
#[path = "win32/wake.rs"]
#[allow(unsafe_code)]
mod wake;
#[cfg(target_arch = "wasm32")]
#[path = "web/wake.rs"]
mod wake;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
pub use runtime::{Viewport, ViewportRuntime};

#[cfg(target_os = "linux")]
mod wake;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
pub use wake::{ViewportReady, ViewportWake};

#[path = "gui/packet.rs"]
mod gui_packet;
#[path = "gui/resources.rs"]
mod gui_resources;
pub use gui_packet::{GuiAtlas, GuiBatch, GuiFrame, GuiVertex};
pub use gui_resources::{GuiTextureKey, GuiTextureResources, GuiTextureSource};
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
#[path = "gui/render.rs"]
mod gui_render;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
pub use gui_render::{GuiFrameCompiler, compile_gui_frame};

mod sketch_preview;
pub use sketch_preview::ViewportSketchPreview;

#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod camera_transition;
mod sketch_camera;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
pub use camera_transition::ViewportCameraTransition;

mod sketch_display;
pub use sketch_display::{
    ViewportSketchBlock, ViewportSketchCurvature, ViewportSketchDecline, ViewportSketchDisplay,
    ViewportSketchFeedback, ViewportSketchHover, ViewportSketchSelection,
};

mod sketch_palette;
pub use sketch_palette::ViewportSketchPalette;

pub mod assembly_presentation;
