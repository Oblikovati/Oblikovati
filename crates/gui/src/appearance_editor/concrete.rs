// SPDX-License-Identifier: GPL-2.0-only
//! Concrete uses the common base color and its own finish and weathering choices.
use super::image_chooser::{HeightMapTarget, ImageChooser};
use super::optional_effects::{choice, edit_effect_color, section};
use super::{AppearanceDraft, ColorDialog, base_color};
use imgui::Ui;
use oblikovati_model::assets::{ConcreteAppearance, ConcreteFinish, ConcreteSealant};

pub(super) fn draw(
    ui: &Ui,
    draft: &mut AppearanceDraft,
    color_dialog: &mut ColorDialog,
    chooser: &mut ImageChooser,
) {
    let Some(concrete) = draft.metadata.concrete.as_mut() else {
        return;
    };
    let previous = concrete.surface();
    if let Some(_main) = ui.tree_node_config("Concrete").default_open(true).push() {
        base_color(ui, &mut draft.color, color_dialog);
        sealant(ui, &mut concrete.sealant);
    }
    finish(ui, concrete, chooser);
    weathering(ui, concrete, chooser);
    let tint = &mut concrete.tint;
    section(ui, "Tint", &mut tint.enabled, || {
        edit_effect_color(ui, "Tint Color", &mut tint.color);
    });
    let after = concrete.surface();
    if after != previous {
        [draft.roughness, draft.highlights] = after;
    }
}

fn sealant(ui: &Ui, sealant: &mut ConcreteSealant) {
    let values = [
        ConcreteSealant::None,
        ConcreteSealant::Epoxy,
        ConcreteSealant::Acrylic,
    ];
    let mut index = match sealant {
        ConcreteSealant::None => 0,
        ConcreteSealant::Epoxy => 1,
        ConcreteSealant::Acrylic => 2,
    };
    if ui.combo_simple_string("Sealant", &mut index, &["None", "Epoxy", "Acrylic"]) {
        *sealant = values[index];
    }
}

fn finish(ui: &Ui, concrete: &mut ConcreteAppearance, chooser: &mut ImageChooser) {
    let Some(_finish) = ui.tree_node_config("Finish Bumps").push() else {
        return;
    };
    choice(
        ui,
        "Type",
        &mut concrete.finish,
        &[
            (ConcreteFinish::BroomStraight, "Broom Straight"),
            (ConcreteFinish::BroomCurved, "Broom Curved"),
            (ConcreteFinish::Smooth, "Smooth"),
            (ConcreteFinish::Polished, "Polished"),
            (ConcreteFinish::Stamped, "Stamped/Custom"),
        ],
    );
    if concrete.finish == ConcreteFinish::Stamped {
        if ui.button("Image") {
            chooser.request(HeightMapTarget::ConcreteStamp);
        }
        image_name(ui, concrete.stamped_image.as_ref());
    }
}

fn weathering(ui: &Ui, concrete: &mut ConcreteAppearance, chooser: &mut ImageChooser) {
    let image = &mut concrete.weathering_image;
    section(ui, "Weathering", &mut concrete.weathering, || {
        let label = if image.is_some() {
            "Custom Image"
        } else {
            "Automatic"
        };
        if let Some(_types) = ui.begin_combo("Type", label) {
            if ui
                .selectable_config("Automatic")
                .selected(image.is_none())
                .build()
            {
                *image = None;
            }
            if ui
                .selectable_config("Custom Image")
                .selected(image.is_some())
                .build()
            {
                chooser.request(HeightMapTarget::ConcreteWeathering);
            }
        }
        if image.is_some() {
            if ui.button("Image") {
                chooser.request(HeightMapTarget::ConcreteWeathering);
            }
            image_name(ui, image.as_ref());
        }
    });
}

fn image_name(ui: &Ui, map: Option<&oblikovati_model::assets::AppearanceHeightMap>) {
    ui.same_line_with_spacing(0., 14.);
    ui.text(map.map_or("No image selected", |value| value.name()));
}
