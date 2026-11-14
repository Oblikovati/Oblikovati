// SPDX-License-Identifier: GPL-2.0-only
//! Global preferences for the orientation control.

/// Read and persist the user's selection pivot preference.
/// Example: `store.set_orientation_lock(true)?` after a menu choice.
pub trait OrientationPreferenceAccess {
    /// Read the current saved lock; example: `store.orientation_lock()`.
    fn orientation_lock(&self) -> bool;
    /// Persist before changing the published value; example: `store.set_orientation_lock(false)?`.
    fn set_orientation_lock(&mut self, locked: bool) -> Result<(), String>;
}
