// SPDX-License-Identifier: GPL-2.0-only
use super::{ViewTarget, Viewport};

impl Viewport {
    /// Dispatch an application-bar window action; example: `view.window_action(ViewportWindowAction::Minimize)?`.
    pub fn window_action(&mut self, action: crate::ViewportWindowAction) -> Result<(), String> {
        self.live()?;
        let Some(ViewTarget::Native(native)) = &self.target else {
            return Err(format!(
                "viewport {}: expected a native window for {action:?}",
                self.id
            ));
        };
        native.window_action(action)
    }

    /// Read the native maximize state; example: `view.window_maximized()?` before drawing Restore.
    pub fn window_maximized(&self) -> Result<bool, String> {
        self.live()?;
        let Some(ViewTarget::Native(native)) = &self.target else {
            return Err(format!(
                "viewport {}: expected a native window for maximize state",
                self.id
            ));
        };
        native.window_maximized()
    }

    /// Read physical screen bounds after a native geometry event; example: `view.window_screen_bounds()?`.
    #[cfg(target_os = "linux")]
    pub fn window_screen_bounds(&self) -> Result<[i32; 4], String> {
        self.live()?;
        let Some(ViewTarget::Native(native)) = &self.target else {
            return Err(format!(
                "viewport {}: expected a native window for screen bounds",
                self.id
            ));
        };
        native.window_screen_bounds()
    }

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
    /// Read the actual native window title; example: `view.native_title()?` checks displayed status metadata.
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
