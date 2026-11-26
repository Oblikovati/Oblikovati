// SPDX-License-Identifier: GPL-2.0-only
//! Sample textured and transmissive surfaces for the browser and editor previews.
use crate::appearance_editor::CheckerSampler;
use imgui::DrawListMut;
use oblikovati_model::assets::{
    AppearanceChecker, AppearanceReflectivity, AppearanceTransparency, GenericAppearanceEffects,
};

pub(super) fn paint(
    draw: &DrawListMut<'_>,
    center: [f32; 2],
    radius: f32,
    material: &PreviewSurface<'_>,
) {
    let cells = if radius > 40. { 36 } else { 20 };
    let step = 2. / cells as f32;
    for y in 0..cells {
        for x in 0..cells {
            let left = x as f32 * step - 1.;
            let top = y as f32 * step - 1.;
            let color = material.sample([left + step * 0.5, top + step * 0.5]);
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
}

impl<'a> PreviewSurface<'a> {
    pub(super) fn new(
        base: [f32; 3],
        reflectivity: Option<AppearanceReflectivity>,
        transparency: Option<AppearanceTransparency>,
        checker: Option<AppearanceChecker>,
        effects: Option<&'a GenericAppearanceEffects>,
    ) -> Self {
        Self {
            base,
            effects,
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

    fn sample(&self, [x, y]: [f32; 2]) -> [f32; 4] {
        let squared_radius = x * x + y * y;
        if squared_radius >= 1. {
            return if self.transmission > 0. {
                checker(x, y).map_or([0.; 4], |shade| [shade, shade, shade, 1.])
            } else {
                [0.; 4]
            };
        }
        let normal_z = (1. - squared_radius).sqrt();
        let scale = 1. + self.bend * normal_z * 0.8;
        let through = checker(x * scale, y * scale).unwrap_or(0.40);
        let through = through + (0.40 - through) * self.scatter;
        let diffuse = (0.56 * normal_z - 0.23 * x - 0.28 * y).max(0.) + 0.27;
        let base = self.checker.as_ref().map_or(self.base, |(sampler, fade)| {
            let texture = sampler.color([(x + 1.) * 50., (y + 1.) * 50.]);
            std::array::from_fn(|channel| {
                self.base[channel] + (texture[channel] - self.base[channel]) * fade
            })
        });
        let base = self
            .effects
            .map_or(base, |effects| super::preview_effect_color(base, effects));
        let surface = super::surface_color(base, diffuse, normal_z, self.reflectivity);
        let [r, g, b] = surface.map(|channel| channel + (through - channel) * self.transmission);
        [r, g, b, 1.]
    }
}

fn checker(x: f32, y: f32) -> Option<f32> {
    if x.abs() > 1. || y.abs() > 1. {
        return None;
    }
    let parity = ((x * 3.).floor() as i32 + (y * 3.).floor() as i32).rem_euclid(2);
    Some(if parity == 0 { 0.23 } else { 0.57 })
}
