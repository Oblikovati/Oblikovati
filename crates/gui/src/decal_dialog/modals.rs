// SPDX-License-Identifier: GPL-2.0-only
//! Transient image-sketch-required tip modal for dockable Decal.
use crate::widget_abi;
use imgui::Ui;

pub(super) const IMAGE_TIP_TITLE: &str = "Decal - Image Sketch Required";

#[derive(Default)]
pub(super) struct DecalModals {
    image_tip: bool,
    image_tip_open: bool,
}

impl DecalModals {
    pub(super) fn reset(&mut self) {
        self.image_tip = false;
        self.image_tip_open = false;
    }

    pub(super) fn show_image_tip(&mut self) {
        self.image_tip = true;
    }

    pub(super) fn draw(&mut self, ui: &Ui) {
        if self.image_tip && !self.image_tip_open {
            ui.open_popup(IMAGE_TIP_TITLE);
        }
        self.image_tip_open = self.image_tip;
        let Some(_popup) = ui
            .modal_popup_config(IMAGE_TIP_TITLE)
            .always_auto_resize(true)
            .begin_popup()
        else {
            return;
        };
        if !self.image_tip {
            ui.close_current_popup();
            return;
        }
        ui.text_wrapped(
            "No visible 2D sketch with an inserted image is available. Make a 2D sketch with an inserted image visible before creating a Decal.",
        );
        ui.spacing();
        if widget_abi::button(ui, "OK", [64., 28.])
            || ui.is_key_pressed_no_repeat(imgui::Key::Escape)
        {
            self.image_tip = false;
            self.image_tip_open = false;
            ui.close_current_popup();
        }
    }
}
