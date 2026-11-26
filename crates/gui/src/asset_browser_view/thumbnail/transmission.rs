// SPDX-License-Identifier: GPL-2.0-only
//! Composite a transmissive preview over a reference checkerboard.
use imgui::DrawListMut;
use oblikovati_model::assets::{AppearanceReflectivity, AppearanceTransparency};

pub(super) fn paint(
    draw: &DrawListMut<'_>,
    center: [f32; 2],
    radius: f32,
    base: [f32; 3],
    reflectivity: Option<AppearanceReflectivity>,
    effect: AppearanceTransparency,
) {
    let cells = if radius > 40. { 36 } else { 20 };
    let step = 2. / cells as f32;
    let optics = Optics::new(base, reflectivity, effect);
    for y in 0..cells {
        for x in 0..cells {
            let left = x as f32 * step - 1.;
            let top = y as f32 * step - 1.;
            let color = optics.sample([left + step * 0.5, top + step * 0.5]);
            let min = [center[0] + radius * left, center[1] + radius * top];
            let max = [min[0] + radius * step, min[1] + radius * step];
            draw.add_rect(min, max, color).filled(true).build();
        }
    }
}

struct Optics {
    base: [f32; 3],
    transmission: f32,
    scatter: f32,
    reflectivity: Option<AppearanceReflectivity>,
    bend: f32,
}

impl Optics {
    fn new(
        base: [f32; 3],
        reflectivity: Option<AppearanceReflectivity>,
        effect: AppearanceTransparency,
    ) -> Self {
        Self {
            base,
            reflectivity,
            transmission: f32::from(effect.amount) / 100.,
            scatter: f32::from(effect.translucency) / 100.,
            bend: (effect.index_of_refraction() - 1.).clamp(-0.9, 3.),
        }
    }

    fn sample(&self, [x, y]: [f32; 2]) -> [f32; 4] {
        let squared_radius = x * x + y * y;
        if squared_radius >= 1. {
            return checker(x, y).map_or([0.; 4], |shade| [shade, shade, shade, 1.]);
        }
        let normal_z = (1. - squared_radius).sqrt();
        let scale = 1. + self.bend * normal_z * 0.8;
        let through = checker(x * scale, y * scale).unwrap_or(0.40);
        let through = through + (0.40 - through) * self.scatter;
        let diffuse = (0.56 * normal_z - 0.23 * x - 0.28 * y).max(0.) + 0.27;
        let surface = super::surface_color(self.base, diffuse, normal_z, self.reflectivity);
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
