// SPDX-License-Identifier: GPL-2.0-only
/// Owner-thread text clipboard I/O behind the internal platform boundary.
/// Implementations keep native handles private and return named transfer failures.
pub trait ViewportClipboard {
    /// Read current text without returning partial transfers; example: `clipboard.read_text()?`.
    fn read_text(&self) -> Result<Option<String>, String>;
    /// Publish an owned text snapshot; example: `clipboard.write_text("width = 6 cm")?`.
    fn write_text(&self, text: &str) -> Result<(), String>;
}

#[cfg(target_os = "linux")]
mod native;
#[cfg(target_os = "linux")]
pub use native::{NativeClipboardEndpoint, NativeClipboardHandoff, NativeClipboardOptions};
