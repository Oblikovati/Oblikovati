// SPDX-License-Identifier: GPL-2.0-only
//! Drive native scenario inputs through the window's active Linux backend.
use super::super::{ApplicationWorld, x11_input};

impl ApplicationWorld {
    pub fn desktop_click(&mut self, position: [i32; 2]) {
        if self.app().window.native.view.uses_wayland() {
            self.click(position);
        } else {
            x11_input::click(position);
        }
    }

    pub fn desktop_right_click(&mut self, position: [i32; 2]) {
        if self.app().window.native.view.uses_wayland() {
            self.right_click(position);
        } else {
            x11_input::right_click(position);
        }
    }

    pub fn desktop_escape(&mut self) {
        if self.app().window.native.view.uses_wayland() {
            self.escape();
        } else {
            x11_input::press_escape();
        }
    }
}
