// SPDX-License-Identifier: GPL-2.0-only
//! Trusted native add-in admission; C prototypes come from the standalone API.
use oblikovati_api::Error;
#[cfg(target_os = "linux")]
#[allow(unsafe_code)]
mod linux;
#[cfg(target_os = "linux")]
pub use linux::{
    AddInState, HostBufferCounts, HostCallbackBinding, LoadedAddIn, NativeAddInRegistry,
    open_addin, probe_library,
};
mod compatibility;
pub use compatibility::check_api_version;
/// Copied identity and capability evidence, not an activated library or compatibility promise.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct AddInProbe {
    pub id: String,
    pub manifest: String,
    pub major: i32,
    pub minor: i32,
    pub has_automation: bool,
}
