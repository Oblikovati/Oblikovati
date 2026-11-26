// SPDX-License-Identifier: GPL-2.0-only
//! The ceramic property page uses the shared appearance draft and color picker.
use super::image_chooser::{HeightMapTarget, ImageChooser};
use super::optional_effects::{edit_effect_color, section};
use super::{AppearanceDraft, ColorDialog, base_color};
use imgui::Ui;
use oblikovati_model::assets::{CeramicAppearance, CeramicFinish, CeramicKind};

pub(super) fn draw(
    ui: &Ui,
    draft: &mut AppearanceDraft,
    color_dialog: &mut ColorDialog,
    image_chooser: &mut ImageChooser,
) {
    let Some(ceramic) = draft.metadata.ceramic.as_mut() else {
        return;
    };
    if let Some(_main) = ui.tree_node_config("Ceramic").default_open(true).push() {
        choose_kind(ui, &mut ceramic.kind);
        base_color(ui, &mut draft.color, color_dialog);
        if choose_finish(ui, &mut ceramic.finish) {
            [draft.roughness, draft.highlights] = ceramic.finish.surface();
        }
    }
    effects(ui, ceramic, image_chooser);
}

fn choose_kind(ui: &Ui, kind: &mut CeramicKind) {
    let variants = [CeramicKind::Ceramic, CeramicKind::Porcelain];
    let mut selected = usize::from(*kind == CeramicKind::Porcelain);
    if ui.combo_simple_string("Type", &mut selected, &["Ceramic", "Porcelain"]) {
        *kind = variants[selected];
    }
}

fn choose_finish(ui: &Ui, finish: &mut CeramicFinish) -> bool {
    let variants = [
        CeramicFinish::Glazed,
        CeramicFinish::Satin,
        CeramicFinish::Matte,
    ];
    let mut selected = match finish {
        CeramicFinish::Glazed => 0,
        CeramicFinish::Satin => 1,
        CeramicFinish::Matte => 2,
    };
    if !ui.combo_simple_string(
        "Finish",
        &mut selected,
        &["High Gloss / Glazed", "Satin", "Matte"],
    ) {
        return false;
    }
    *finish = variants[selected];
    true
}

fn effects(ui: &Ui, ceramic: &mut CeramicAppearance, image_chooser: &mut ImageChooser) {
    let bumps = &mut ceramic.finish_bumps;
    section(ui, "Finish Bumps", &mut bumps.enabled, || {
        bump_type(ui, &mut ceramic.custom_bump, image_chooser);
        amount(ui, &mut bumps.amount);
    });
    let relief = &mut ceramic.relief;
    section(ui, "Relief Pattern", &mut relief.enabled, || {
        if ui.button("Image") {
            image_chooser.request(HeightMapTarget::Relief);
        }
        ui.same_line_with_spacing(0., 14.);
        ui.text(
            ceramic
                .relief_image
                .as_ref()
                .map_or("No image selected", |map| map.name()),
        );
        if ceramic.relief_image.is_some() && ui.button("Remove image") {
            ceramic.relief_image = None;
        }
        amount(ui, &mut relief.amount);
    });
    let tint = &mut ceramic.tint;
    section(ui, "Tint", &mut tint.enabled, || {
        edit_effect_color(ui, "Tint Color", &mut tint.color);
    });
}

fn bump_type(
    ui: &Ui,
    custom: &mut Option<oblikovati_model::assets::AppearanceHeightMap>,
    image_chooser: &mut ImageChooser,
) {
    let preview = if custom.is_some() { "Custom" } else { "Wavy" };
    if let Some(_choices) = ui.begin_combo("Type", preview) {
        if ui
            .selectable_config("Wavy")
            .selected(custom.is_none())
            .build()
        {
            *custom = None;
        }
        if ui
            .selectable_config("Custom")
            .selected(custom.is_some())
            .build()
        {
            image_chooser.request(HeightMapTarget::FinishBumps);
        }
    }
    if let Some(map) = custom {
        ui.text(map.name());
    }
}

fn amount(ui: &Ui, value: &mut u8) {
    ui.slider_config("Amount", 0, 100)
        .display_format("%d%%")
        .build(value);
}
