// SPDX-License-Identifier: GPL-2.0-only
//! Draw visible appearance cards from their authored base color and finish.
use super::{AssetRows, GuiAssetChoice};
use imgui::{DrawListMut, StyleColor, Ui};
use oblikovati_model::assets::AppearanceReflectivity;

mod concrete_shading;
mod cube;
mod height_shading;
mod sampled_surface;
pub(crate) use cube::paint_cube_preview;

#[derive(Clone, Copy)]
pub(crate) struct PreviewLight {
    offset: [f32; 2],
    color: [f32; 3],
}

impl PreviewLight {
    pub(crate) const GRID: Self = Self::new([-0.32, -0.36], [1.; 3]);

    pub(crate) const fn new(offset: [f32; 2], color: [f32; 3]) -> Self {
        Self { offset, color }
    }

    fn illuminate(self, surface: [f32; 3]) -> [f32; 3] {
        std::array::from_fn(|channel| (surface[channel] * self.color[channel]).min(1.))
    }
}

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
        PreviewLight::GRID,
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
    light: PreviewLight,
) {
    let Some(finish) = asset.appearance_preview else {
        return;
    };
    let material = sampled_surface::PreviewSurface::from_asset(asset);
    let metalness = f32::from(finish.metalness) / 255.;
    let roughness = f32::from(finish.roughness) / 255.;
    if material.needs_samples() {
        sampled_surface::paint(draw, center, radius, &material, light);
    } else {
        paint_surface(draw, center, radius, &material, metalness, light);
    }
    paint_reflection(
        draw,
        center,
        radius,
        roughness,
        f32::from(finish.anisotropy) / 255.,
        f32::from(finish.highlights) / 255.,
        light,
    );
    if finish.fuzz_weight > 0 {
        paint_fibers(draw, center, radius, finish);
    }
}

fn preview_effect_color(
    base: [f32; 3],
    effects: &oblikovati_model::assets::GenericAppearanceEffects,
) -> [f32; 3] {
    let tint = if effects.tint.enabled {
        effects.tint.color
    } else {
        [255; 3]
    };
    let emission = effects.illumination;
    let light = if emission.enabled {
        f32::from(emission.luminance) / 500.
    } else {
        0.
    };
    // Provide a warm/cool cue in the preview while keeping 6500 K neutral.
    let temperature = (f32::from(emission.temperature) - 6500.) / 5500.;
    let temperature_filter = [
        1. - temperature.max(0.) * 0.50,
        1.,
        1. + temperature.min(0.) * 0.60,
    ];
    std::array::from_fn(|channel| {
        let filtered = base[channel] * f32::from(tint[channel]) / 255.;
        let emitted =
            light * f32::from(emission.color[channel]) / 255. * temperature_filter[channel];
        (filtered + emitted).min(1.)
    })
}

fn paint_surface(
    draw: &DrawListMut<'_>,
    center: [f32; 2],
    radius: f32,
    material: &sampled_surface::PreviewSurface<'_>,
    metalness: f32,
    light: PreviewLight,
) {
    let layers = if radius > 40. { 24 } else { 10 };
    for step in 0..=layers {
        let fraction = step as f32 / layers as f32;
        let at = std::array::from_fn(|axis| {
            center[axis]
                + radius * fraction * (light.offset[axis] - PreviewLight::GRID.offset[axis] - 0.38)
        });
        let shade = 0.26 + fraction * (0.81 + metalness * 0.22);
        let [r, g, b] = light.illuminate(material.color([0.; 2], shade, fraction));
        draw.add_circle(at, radius * (1. - 0.82 * fraction), [r, g, b, 1.])
            .filled(true)
            .build();
    }
}

fn surface_color(
    base: [f32; 3],
    shade: f32,
    facing: f32,
    reflectivity: Option<AppearanceReflectivity>,
) -> [f32; 3] {
    let reflection = reflectivity.map_or(0., |effect| {
        let direct = f32::from(effect.direct) / 100.;
        let oblique = f32::from(effect.oblique) / 100.;
        0.50 * direct * facing + 0.75 * oblique * (1. - facing).powi(5)
    });
    std::array::from_fn(|index| {
        let surface = (base[index] * shade + 0.025).min(1.);
        surface + ([0.75, 0.83, 0.93][index] - surface) * reflection
    })
}

fn paint_reflection(
    draw: &DrawListMut<'_>,
    center: [f32; 2],
    radius: f32,
    roughness: f32,
    anisotropy: f32,
    highlights: f32,
    environment: PreviewLight,
) {
    let light =
        std::array::from_fn::<_, 2, _>(|axis| center[axis] + radius * environment.offset[axis]);
    let [r, g, b] = environment
        .illuminate([0.85; 3])
        .map(|channel| (channel / 0.85).min(1.));
    let intensity = (0.18 + 0.55 * (1. - roughness)) * highlights;
    for layer in (1..=4).rev() {
        let diffuse = radius * (0.11 + roughness * 0.25) * layer as f32 / 4.;
        draw.add_circle(light, diffuse, [r, g, b, intensity / 4.])
            .filled(true)
            .build();
    }
    if anisotropy > 0. {
        let length = radius * (0.2 + 0.4 * anisotropy);
        draw.add_line(
            [light[0] - length, light[1]],
            [light[0] + length, light[1]],
            [r, g, b, anisotropy * intensity * 0.4],
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
