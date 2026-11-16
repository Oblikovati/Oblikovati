// SPDX-License-Identifier: GPL-2.0-only
use super::{ViewTarget, Viewport};

impl Viewport {
    /// Update native presentation status without changing geometry; example: `view.set_title("Oblikovati - feature failed")?`.
    pub fn set_title(&mut self, title: &str) -> Result<(), String> {
        self.live()?;
        let Some(ViewTarget::Native(native)) = &self.target else {
            return Err(format!(
                "viewport {}: expected a native title target",
                self.id
            ));
        };
        native.set_title(title)
    }
    /// Read the actual X11 name; example: `view.native_title()?` checks displayed status metadata.
    pub fn native_title(&self) -> Result<String, String> {
        self.live()?;
        let Some(ViewTarget::Native(native)) = &self.target else {
            return Err(format!(
                "viewport {}: expected a native title target",
                self.id
            ));
        };
        native.title()
    }
}
