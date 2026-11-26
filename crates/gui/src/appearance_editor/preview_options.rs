// SPDX-License-Identifier: GPL-2.0-only
//! Scene and lighting choices belong to the open preview, outside the asset draft.
use crate::GuiAssetChoice;
use crate::asset_browser_view::{PreviewLight, paint_appearance_preview, paint_cube_preview};
use imgui::Ui;

#[derive(Default)]
pub(super) struct PreviewOptions {
    cube: bool,
    environment: usize,
}

const ENVIRONMENTS: [(&str, PreviewLight); 4] = [
    ("Grid Light", PreviewLight::GRID),
    (
        "Plaza",
        PreviewLight::new([0.26, -0.46], [1.12, 1.04, 0.88]),
    ),
    (
        "Snowfield",
        PreviewLight::new([-0.13, -0.28], [0.90, 1.10, 1.23]),
    ),
    (
        "Warm Light",
        PreviewLight::new([-0.45, -0.23], [1.22, 1.02, 0.70]),
    ),
];

impl PreviewOptions {
    pub(super) fn draw(&mut self, ui: &Ui, asset: &GuiAssetChoice) {
        let [left, top] = ui.cursor_screen_pos();
        let width = ui.content_region_avail()[0];
        ui.dummy([width, 116.]);
        let draw = ui.get_window_draw_list();
        let center = [left + width / 2., top + 55.];
        let light = ENVIRONMENTS[self.environment].1;
        if self.cube {
            paint_cube_preview(&draw, center, 49., asset, light);
        } else {
            paint_appearance_preview(&draw, center, 49., asset, light);
        }
        let next_row = ui.cursor_screen_pos();
        let button_width = ui.calc_text_size("Preview Options")[0] + 15.;
        ui.set_cursor_screen_pos([left + (width - button_width).max(0.), top + 88.]);
        self.menu(ui);
        ui.set_cursor_screen_pos(next_row);
    }

    fn menu(&mut self, ui: &Ui) {
        if ui.button("Preview Options") {
            ui.open_popup("##appearance-preview-options");
        }
        let Some(_menu) = ui.begin_popup("##appearance-preview-options") else {
            return;
        };
        if let Some(_scene) = ui.begin_menu("Scene") {
            self.scene_menu(ui);
        }
        if let Some(_environment) = ui.begin_menu("Environment") {
            for (index, (label, _)) in ENVIRONMENTS.iter().enumerate() {
                if ui
                    .menu_item_config(label)
                    .selected(index == self.environment)
                    .build()
                {
                    self.environment = index;
                }
            }
        }
        if let Some(_render) = ui.begin_menu("Render Settings") {
            ui.menu_item_config("Quick").selected(true).build();
        }
    }

    fn scene_menu(&mut self, ui: &Ui) {
        if ui.menu_item_config("Sphere").selected(!self.cube).build() {
            self.cube = false;
        }
        if ui.menu_item_config("Cube").selected(self.cube).build() {
            self.cube = true;
        }
        for label in [
            "Cylinder",
            "Canvas",
            "Plane",
            "Object",
            "Vase",
            "Draped Fabric",
            "Glass Curtain Wall",
            "Walls",
            "Pool of Liquid",
            "Utility",
            "Torus",
        ] {
            ui.menu_item_config(label).enabled(false).build();
        }
    }
}
