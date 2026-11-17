// SPDX-License-Identifier: GPL-2.0-only
//! Window-manager operations for the application frame.

/// A window action requested by its application bar.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ViewportWindowAction {
    /// Hide the window in the taskbar.
    Minimize,
    /// Toggle between the restored and maximized states.
    ToggleMaximize,
    /// Ask the window manager to move the restored window from the current pointer.
    Move,
}
