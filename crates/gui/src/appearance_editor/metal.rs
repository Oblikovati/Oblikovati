// SPDX-License-Identifier: GPL-2.0-only
//! Edit the alloy, finish, and optional maps for a Metal appearance.
use super::AppearanceDraft;
use super::image_chooser::{HeightMapTarget, ImageChooser};
use super::optional_effects::{choice, edit_effect_color, section};
use imgui::Ui;
use oblikovati_model::assets::{AppearanceHeightMap, MetalAppearance, MetalFinish, MetalKind};

const ALLOYS: [(MetalKind, &str); 8] = [
    (MetalKind::Aluminum, "Aluminum"),
    (MetalKind::AnodizedAluminum, "Anodized Aluminum"),
    (MetalKind::Chrome, "Chrome"),
    (MetalKind::Copper, "Copper"),
    (MetalKind::Brass, "Brass"),
    (MetalKind::Bronze, "Bronze"),
    (MetalKind::StainlessSteel, "Stainless Steel"),
    (MetalKind::Zinc, "Zinc"),
];

const FINISHES: [(MetalFinish, &str); 4] = [
    (MetalFinish::Polished, "Polished"),
    (MetalFinish::SemiPolished, "Semi-polished"),
    (MetalFinish::Satin, "Satin"),
    (MetalFinish::Brushed, "Brushed"),
];

pub(super) fn draw(ui: &Ui, draft: &mut AppearanceDraft, chooser: &mut ImageChooser) {
    let Some(metal) = &mut draft.metadata.metal else {
        return;
    };
    if let Some(_properties) = ui.tree_node_config("Metal").default_open(true).push() {
        let previous = (metal.kind, metal.finish);
        choice(ui, "Type", &mut metal.kind, &ALLOYS);
        choice(ui, "Finish", &mut metal.finish, &FINISHES);
        if metal.kind != previous.0 {
            draft.color = metal.kind.color();
        }
        if metal.finish != previous.1 {
            [draft.roughness, draft.highlights] = metal.finish.surface();
        }
    }
    effects(ui, metal, chooser);
}

fn effects(ui: &Ui, metal: &mut MetalAppearance, chooser: &mut ImageChooser) {
    section(ui, "Relief Pattern", &mut metal.relief.enabled, || {
        image_slot(
            ui,
            &mut metal.relief_image,
            chooser,
            HeightMapTarget::MetalRelief,
        );
        ui.slider_config("Amount", 0, 100)
            .display_format("%d%%")
            .build(&mut metal.relief.amount);
    });
    section(ui, "Cutouts", &mut metal.cutouts, || {
        image_slot(
            ui,
            &mut metal.cutout_image,
            chooser,
            HeightMapTarget::MetalCutouts,
        );
    });
    section(ui, "Tint", &mut metal.tint.enabled, || {
        edit_effect_color(ui, "Tint Color", &mut metal.tint.color);
    });
}

fn image_slot(
    ui: &Ui,
    image: &mut Option<AppearanceHeightMap>,
    chooser: &mut ImageChooser,
    target: HeightMapTarget,
) {
    if ui.button("Image") {
        chooser.request(target);
    }
    ui.same_line_with_spacing(0., 14.);
    ui.text(image.as_ref().map_or("No image selected", |map| map.name()));
    if image.is_some() && ui.button("Remove image") {
        *image = None;
    }
}
