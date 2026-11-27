// SPDX-License-Identifier: GPL-2.0-only
//! Generic Image menu and separate Checker texture editor.
use crate::dockable_lifecycle::{DockRegistry, DockableState};
use crate::dockable_window::DockableWindow;
use imgui::{Condition, Ui};
use oblikovati_model::assets::{AppearanceChecker, TextureRepeat};

mod sample;
pub(crate) use sample::CheckerSampler;

pub(super) fn slot(ui: &Ui, checker: &mut Option<AppearanceChecker>, open: &mut bool) {
    if ui.button("Image") {
        ui.open_popup("##image-menu");
    }
    ui.same_line_with_spacing(0., 14.);
    ui.text(if checker.is_some() {
        "Checker texture"
    } else {
        "No image selected"
    });
    if let Some(_menu) = ui.begin_popup("##image-menu") {
        ui.menu_item_config("Image").enabled(false).build();
        if ui.menu_item("Checker") {
            checker.get_or_insert_default();
            *open = true;
        }
        for label in [
            "Gradient", "Marble", "Noise", "Speckle", "Tiles", "Waves", "Wood",
        ] {
            ui.menu_item_config(label).enabled(false).build();
        }
        ui.separator();
        if ui
            .menu_item_config("Edit Image")
            .enabled(checker.is_some())
            .build()
        {
            *open = true;
        }
        if ui
            .menu_item_config("Remove Image")
            .enabled(checker.is_some())
            .build()
        {
            *checker = None;
            *open = false;
        }
    }
    let _disabled = ui.begin_disabled(checker.is_none());
    if let Some(checker) = checker {
        ui.slider_config("Image Fade", 0, 100)
            .display_format("%d%%")
            .build(&mut checker.fade);
    } else {
        ui.text("Image Fade");
    }
}

const TITLE: &str = "Texture Editor — Checker###texture-editor";

#[derive(Default)]
pub(super) struct TextureWindow(DockableState);
impl DockableWindow for TextureWindow {}

impl TextureWindow {
    pub(super) fn register_dockable(&self, registry: &mut DockRegistry) -> Result<(), String> {
        registry.register::<Self>(TITLE, &self.0)
    }
    pub(super) fn set_open(&self, open: bool) {
        self.0.set_open(open);
    }
    pub(super) fn draw(&self, ui: &Ui, checker: &mut Option<AppearanceChecker>, open: &mut bool) {
        self.set_open(checker.is_some() && *open);
        let Some(checker) = checker.as_mut().filter(|_| *open) else {
            return;
        };
        crate::widget_abi::window_geometry(
            ui,
            [ui.io().display_size[0] * 0.055, 170.],
            [370., 600.],
            Condition::FirstUseEver,
        );
        Self::dockable_window(ui, TITLE, open).build(|| {
            preview(ui, *checker);
            color(ui, "Color 1", &mut checker.colors[0]);
            color(ui, "Color 2", &mut checker.colors[1]);
            ui.slider_config("Soften", 0, 100)
                .display_format("%d%%")
                .build(&mut checker.soften);
            transforms(ui, checker);
        });
        self.set_open(*open);
    }
}

fn preview(ui: &Ui, checker: AppearanceChecker) {
    let [x, y] = ui.cursor_screen_pos();
    let width = ui.content_region_avail()[0].min(180.);
    let side = width.max(1.);
    ui.dummy([side, side]);
    sample::paint_square(&ui.get_window_draw_list(), [x, y], side, checker);
    let [width, height] = checker.size_mm;
    ui.text(format!("Sample {width} × {height} mm"));
}

fn color(ui: &Ui, label: &str, color: &mut [u8; 3]) {
    let mut rgb = color.map(|channel| f32::from(channel) / 255.);
    if ui.color_edit3(label, &mut rgb) {
        *color = rgb.map(|channel| (channel.clamp(0., 1.) * 255.).round() as u8);
    }
}

fn transforms(ui: &Ui, checker: &mut AppearanceChecker) {
    ui.separator();
    ui.text("Transforms");
    ui.text("Position");
    signed(ui, "X offset (mm)", &mut checker.offset_mm[0]);
    signed(ui, "Y offset (mm)", &mut checker.offset_mm[1]);
    signed(ui, "Rotation (°)", &mut checker.rotation_degrees);
    ui.text("Scale");
    ui.checkbox("Link texture transforms", &mut checker.link_scale);
    dimensions(ui, checker);
    ui.text("Repeat");
    repeat(ui, "Horizontal", &mut checker.repeat[0]);
    repeat(ui, "Vertical", &mut checker.repeat[1]);
}

fn dimensions(ui: &Ui, checker: &mut AppearanceChecker) {
    let labels = ["Sample width (mm)", "Sample height (mm)"];
    for (axis, label) in labels.into_iter().enumerate() {
        let before = checker.size_mm[axis];
        let mut value = i32::from(before);
        if ui.input_int(label, &mut value).build() {
            let value = value.clamp(1, i32::from(u16::MAX)) as u16;
            checker.size_mm[axis] = value;
            if checker.link_scale {
                let other = f32::from(checker.size_mm[1 - axis]);
                checker.size_mm[1 - axis] = (other * f32::from(value) / f32::from(before.max(1)))
                    .round()
                    .clamp(1., f32::from(u16::MAX))
                    as u16;
            }
        }
    }
}

fn signed(ui: &Ui, label: &str, value: &mut i16) {
    let mut edit = i32::from(*value);
    if ui.input_int(label, &mut edit).build() {
        *value = edit.clamp(i32::from(i16::MIN), i32::from(i16::MAX)) as i16;
    }
}

fn repeat(ui: &Ui, label: &str, mode: &mut TextureRepeat) {
    const MODES: [TextureRepeat; 3] = [
        TextureRepeat::Tile,
        TextureRepeat::Mirror,
        TextureRepeat::Clamp,
    ];
    let mut selected = match mode {
        TextureRepeat::Tile => 0,
        TextureRepeat::Mirror => 1,
        TextureRepeat::Clamp => 2,
    };
    if ui.combo_simple_string(label, &mut selected, &["Tile", "Mirror", "Clamp"]) {
        *mode = MODES[selected];
    }
}
