// SPDX-License-Identifier: GPL-2.0-only
//! Owned Linux viewports over the published Martensite device and presentation path.
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
