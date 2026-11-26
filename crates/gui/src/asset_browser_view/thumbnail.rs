// SPDX-License-Identifier: GPL-2.0-only
//! Draw visible appearance cards from their authored base color and finish.
use super::{AssetRows, GuiAssetChoice};
use imgui::{DrawListMut, StyleColor, Ui};

pub(super) fn grid(ui: &Ui, assets: &[&GuiAssetChoice], selection: &mut AssetRows<'_>, size: f32) {
    let width = ui.content_region_avail()[0].max(1.);
    let columns = ((width / (size + 16.).max(84.)) as usize).max(1);
    let Some(_table) = ui.begin_table("##appearance-grid", columns) else {
        return;
    };
    let rows = assets.len().div_ceil(columns);
    for row in imgui::ListClipper::new(rows as i32).begin(ui).iter() {
        ui.table_next_row();
        for asset in assets.iter().skip(row as usize * columns).take(columns) {
            ui.table_next_column();
            tile(ui, asset, selection, size);
        }
    }
}

fn tile(ui: &Ui, asset: &GuiAssetChoice, selection: &mut AssetRows<'_>, size: f32) {
    let _id = ui.push_id(&asset.id);
    let width = ui.content_region_avail()[0].max(1.);
    let size = size.min((width - 8.).max(1.));
    let height = size + ui.text_line_height_with_spacing() + 8.;
    if ui
        .selectable_config("##appearance-tile")
        .size([width, height])
        .selected(selection.selected.as_deref() == Some(&asset.id))
        .build()
    {
        *selection.selected = Some(asset.id.clone());
    }
    let hovered = ui.is_item_hovered();
    let [left, top] = ui.item_rect_min();
    let draw = ui.get_window_draw_list();
    paint_preview(
        &draw,
        [left + width / 2., top + size / 2. + 4.],
        size / 2.,
        asset,
    );
    let caption = crate::application::home_recent_documents::recent_rows::fit_recent_cell(
        &asset.name,
        (width - 8.).max(1.),
        |text| ui.calc_text_size(text)[0],
    );
    draw.add_text(
        [left + 4., top + size + 6.],
        ui.style_color(StyleColor::Text),
        &*caption,
    );
    selection.library_context_menu(ui, asset);
    if hovered && caption != asset.name {
        ui.tooltip_text(&asset.name);
    }
}

pub(crate) fn paint_preview(
    draw: &DrawListMut<'_>,
    center: [f32; 2],
    radius: f32,
    asset: &GuiAssetChoice,
) {
    let Some(finish) = asset.appearance_preview else {
        return;
    };
    let base = asset
        .swatch
        .unwrap_or([128; 3])
        .map(|v| f32::from(v) / 255.);
    let metalness = f32::from(finish.metalness) / 255.;
    let roughness = f32::from(finish.roughness) / 255.;
    let reflectivity = asset
        .appearance_metadata
        .as_ref()
        .and_then(|metadata| metadata.reflectivity)
        .filter(|effect| effect.enabled);
    let layers = if radius > 40. { 24 } else { 10 };
    for step in 0..=layers {
        let fraction = step as f32 / layers as f32;
        let offset = radius * fraction * 0.38;
        let shade = 0.26 + fraction * (0.81 + metalness * 0.22);
        let reflection = reflectivity.map_or(0., |effect| {
            let direct = f32::from(effect.direct) / 100.;
            let oblique = f32::from(effect.oblique) / 100.;
            0.50 * direct * fraction + 0.75 * oblique * (1. - fraction).powi(5)
        });
        let [r, g, b] = std::array::from_fn(|index| {
            let surface = (base[index] * shade + 0.025).min(1.);
            surface + ([0.75, 0.83, 0.93][index] - surface) * reflection
        });
        draw.add_circle(
            [center[0] - offset, center[1] - offset],
            radius * (1. - 0.82 * fraction),
            [r, g, b, 1.],
        )
        .filled(true)
        .build();
    }
    paint_reflection(
        draw,
        center,
        radius,
        roughness,
        f32::from(finish.anisotropy) / 255.,
        f32::from(finish.highlights) / 255.,
    );
    if finish.fuzz_weight > 0 {
        paint_fibers(draw, center, radius, finish);
    }
}

fn paint_reflection(
    draw: &DrawListMut<'_>,
    center: [f32; 2],
    radius: f32,
    roughness: f32,
    anisotropy: f32,
    highlights: f32,
) {
    let light = [center[0] - radius * 0.32, center[1] - radius * 0.36];
    let intensity = (0.18 + 0.55 * (1. - roughness)) * highlights;
    for layer in (1..=4).rev() {
        let diffuse = radius * (0.11 + roughness * 0.25) * layer as f32 / 4.;
        draw.add_circle(light, diffuse, [1., 1., 1., intensity / 4.])
            .filled(true)
            .build();
    }
    if anisotropy > 0. {
        let length = radius * (0.2 + 0.4 * anisotropy);
        draw.add_line(
            [light[0] - length, light[1]],
            [light[0] + length, light[1]],
            [1., 1., 1., anisotropy * intensity * 0.4],
        )
        .build();
    }
}

// A repeatable stipple represents the authored microfiber lobe at icon scale.
fn paint_fibers(
    draw: &DrawListMut<'_>,
    center: [f32; 2],
    radius: f32,
    finish: crate::GuiAppearancePreview,
) {
    let [r, g, b] = finish.fuzz_color.map(|v| f32::from(v) / 255.);
    let opacity = f32::from(finish.fuzz_weight) / 255. * 0.64;
    for index in 0..60 {
        let x = ((index * 47) % 67) as f32 / 33.5 - 1.;
        let y = ((index * 29) % 61) as f32 / 30.5 - 1.;
        if x * x + y * y < 0.85 {
            let at = [center[0] + x * radius, center[1] + y * radius];
            draw.add_circle(at, (radius * 0.035).clamp(0.55, 2.5), [r, g, b, opacity])
                .filled(true)
                .build();
        }
    }
}
