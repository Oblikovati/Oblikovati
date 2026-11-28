// SPDX-License-Identifier: GPL-2.0-only
//! WebAssembly and WebGPU browser platform target for Oblikovati.

mod commands;
mod navigation;
mod sample;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod session;
#[cfg(target_arch = "wasm32")]
mod wasm_api;

pub use commands::{
    WebCameraSnapshot, WebCommand, WebDocumentSummary, WebFeatureSummary, WebParameterSummary,
    WebStateSnapshot, WebTheme, WebViewOrientation,
};
pub use navigation::WebNavigationController;
pub use sample::{empty_part_package, sample_prism_package};
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
pub use session::WebSession;
#[cfg(target_arch = "wasm32")]
pub use wasm_api::WebCadRuntime;
