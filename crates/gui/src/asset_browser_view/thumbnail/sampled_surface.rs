// SPDX-License-Identifier: GPL-2.0-only
//! Sample textured and transmissive surfaces for the browser and editor previews.
use super::PreviewLight;
use super::height_shading::HeightShading;
use crate::GuiAssetChoice;
use crate::appearance_editor::CheckerSampler;
use imgui::DrawListMut;
use oblikovati_model::assets::{AppearanceReflectivity, GenericAppearanceEffects};

pub(super) fn paint(
    draw: &DrawListMut<'_>,
    center: [f32; 2],
    radius: f32,
    material: &PreviewSurface<'_>,
    light: PreviewLight,
) {
    let cells = if radius > 40. { 36 } else { 20 };
    let step = 2. / cells as f32;
    for y in 0..cells {
        for x in 0..cells {
            let left = x as f32 * step - 1.;
            let top = y as f32 * step - 1.;
            let color = material.sphere([left + step * 0.5, top + step * 0.5], light);
            let min = [center[0] + radius * left, center[1] + radius * top];
            let max = [min[0] + radius * step, min[1] + radius * step];
            draw.add_rect(min, max, color).filled(true).build();
        }
    }
}

pub(super) struct PreviewSurface<'a> {
    base: [f32; 3],
    transmission: f32,
    scatter: f32,
    reflectivity: Option<AppearanceReflectivity>,
    bend: f32,
    checker: Option<(CheckerSampler, f32)>,
    effects: Option<&'a GenericAppearanceEffects>,
    height_shading: HeightShading<'a>,
    rough_diffuse: f32,
}

impl<'a> PreviewSurface<'a> {
    pub(super) fn from_asset(asset: &'a GuiAssetChoice) -> Self {
        let base = asset
            .swatch
            .unwrap_or([128; 3])
            .map(|v| f32::from(v) / 255.);
        let metadata = asset.appearance_metadata.as_ref();
        let ceramic = metadata.and_then(|metadata| metadata.ceramic.as_ref());
        let base = ceramic
            .filter(|value| value.tint.enabled)
            .map_or(base, |value| {
                std::array::from_fn(|index| base[index] * f32::from(value.tint.color[index]) / 255.)
            });
        let reflectivity = metadata
            .and_then(|metadata| metadata.reflectivity)
            .filter(|effect| effect.enabled);
        let effects = metadata.and_then(|metadata| metadata.generic_effects.as_ref());
        let transparency = metadata
            .and_then(|metadata| metadata.transparency)
            .filter(|effect| effect.enabled);
        let checker = metadata
            .and_then(|metadata| metadata.checker)
            .filter(|texture| texture.fade > 0);
        let rough_surface =
            ceramic.is_some() || effects.is_some_and(|effect| effect.rough_surface.is_some());
        Self {
            base,
            effects,
            height_shading: HeightShading::new(ceramic, effects),
            rough_diffuse: asset
                .appearance_preview
                .filter(|_| rough_surface)
                .map_or(0., |finish| 0.48 * f32::from(finish.roughness) / 255.),
            reflectivity,
            transmission: transparency.map_or(0., |effect| f32::from(effect.amount) / 100.),
            scatter: transparency.map_or(0., |effect| f32::from(effect.translucency) / 100.),
            bend: transparency.map_or(0., |effect| {
                (effect.index_of_refraction() - 1.).clamp(-0.9, 3.)
            }),
            checker: checker
                .map(|texture| (CheckerSampler::new(texture), f32::from(texture.fade) / 100.)),
        }
    }

    pub(super) fn needs_samples(&self) -> bool {
        self.transmission > 0. || self.checker.is_some() || self.height_shading.enabled()
    }

    fn sphere(&self, [x, y]: [f32; 2], light: PreviewLight) -> [f32; 4] {
        let squared_radius = x * x + y * y;
        if squared_radius >= 1. {
            return if self.transmission > 0. {
                checker(x, y).map_or([0.; 4], |shade| [shade, shade, shade, 1.])
            } else {
                [0.; 4]
            };
        }
        let normal_z = (1. - squared_radius).sqrt();
        let lateral = [
            light.offset[0] - PreviewLight::GRID.offset[0] - 0.23,
            light.offset[1] - PreviewLight::GRID.offset[1] - 0.28,
        ];
        let diffuse = (0.56 * normal_z + lateral[0] * x + lateral[1] * y).max(0.) + 0.27;
        let [r, g, b] = light.illuminate(self.color([x, y], diffuse, normal_z));
        [r, g, b, 1.]
    }

    pub(super) fn color(&self, [x, y]: [f32; 2], diffuse: f32, facing: f32) -> [f32; 3] {
        let base = self.checker.as_ref().map_or(self.base, |(sampler, fade)| {
            let texture = sampler.color([(x + 1.) * 50., (y + 1.) * 50.]);
            std::array::from_fn(|channel| {
                self.base[channel] + (texture[channel] - self.base[channel]) * fade
            })
        });
        let base = self
            .effects
            .map_or(base, |effects| super::preview_effect_color(base, effects));
        let diffuse = self.height_shading.apply([x, y], diffuse);
        // A rough surface scatters the directional light into a broader, softer lobe.
        let diffuse = diffuse + (0.65 - diffuse) * self.rough_diffuse;
        let surface = super::surface_color(base, diffuse, facing, self.reflectivity);
        if self.transmission == 0. {
            return surface;
        }
        let scale = 1. + self.bend * facing * 0.8;
        let through = checker(x * scale, y * scale).unwrap_or(0.40);
        let through = through + (0.40 - through) * self.scatter;
        surface.map(|channel| channel + (through - channel) * self.transmission)
    }
}

fn checker(x: f32, y: f32) -> Option<f32> {
    if x.abs() > 1. || y.abs() > 1. {
        return None;
    }
    let parity = ((x * 3.).floor() as i32 + (y * 3.).floor() as i32).rem_euclid(2);
    Some(if parity == 0 { 0.23 } else { 0.57 })
}
