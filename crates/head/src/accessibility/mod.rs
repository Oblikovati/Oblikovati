// SPDX-License-Identifier: GPL-2.0-only
//! Platform-independent semantics for the native accessibility bridge.
mod actions;
pub(crate) mod frame;

#[cfg(target_os = "linux")]
pub(crate) mod unix;

#[cfg(target_os = "windows")]
pub(crate) mod windows;
