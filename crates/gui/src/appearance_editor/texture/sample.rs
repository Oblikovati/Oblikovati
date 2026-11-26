// SPDX-License-Identifier: GPL-2.0-only
//! Shared Checker sampling for the texture window and rendered appearance preview.
use imgui::DrawListMut;
use oblikovati_model::assets::{AppearanceChecker, TextureRepeat};

pub(crate) struct CheckerSampler {
    pattern: AppearanceChecker,
    sin: f32,
    cos: f32,
}

impl CheckerSampler {
    pub(crate) fn new(pattern: AppearanceChecker) -> Self {
        let (sin, cos) = f32::from(pattern.rotation_degrees).to_radians().sin_cos();
        Self { pattern, sin, cos }
    }

    pub(crate) fn color(&self, [x, y]: [f32; 2]) -> [f32; 3] {
        let x = x - f32::from(self.pattern.offset_mm[0]);
        let y = y - f32::from(self.pattern.offset_mm[1]);
        let u = (x * self.cos + y * self.sin) / f32::from(self.pattern.size_mm[0].max(1));
        let v = (y * self.cos - x * self.sin) / f32::from(self.pattern.size_mm[1].max(1));
        let soften = f32::from(self.pattern.soften) / 200.;
        let u = band(u, self.pattern.repeat[0], soften);
        let v = band(v, self.pattern.repeat[1], soften);
        let cross = u + v - 2. * u * v;
        std::array::from_fn(|channel| {
            let a = f32::from(self.pattern.colors[0][channel]);
            let b = f32::from(self.pattern.colors[1][channel]);
            (a + (b - a) * cross) / 255.
        })
    }
}

fn band(position: f32, mode: TextureRepeat, soften: f32) -> f32 {
    let position = match mode {
        TextureRepeat::Tile => position.rem_euclid(2.),
        TextureRepeat::Mirror => 2. - (position.rem_euclid(4.) - 2.).abs(),
        TextureRepeat::Clamp => position.clamp(0., 1.9999),
    };
    let fraction = position.fract();
    let stripe = position.floor().min(1.);
    if soften == 0. {
        return stripe;
    }
    let edge = fraction.min(1. - fraction);
    stripe + (0.5 - stripe) * (1. - edge / soften).max(0.)
}

pub(super) fn paint_square(
    draw: &DrawListMut<'_>,
    origin: [f32; 2],
    side: f32,
    pattern: AppearanceChecker,
) {
    let sampler = CheckerSampler::new(pattern);
    const CELLS: usize = 32;
    let cell = side / CELLS as f32;
    for row in 0..CELLS {
        for column in 0..CELLS {
            let point = [column as f32, row as f32];
            let start = std::array::from_fn(|axis| origin[axis] + point[axis] * cell);
            let end = start.map(|value| value + cell);
            let [r, g, b] = sampler.color(point.map(|index| (index + 0.5) * 100. / CELLS as f32));
            draw.add_rect(start, end, [r, g, b, 1.])
                .filled(true)
                .build();
        }
    }
}
