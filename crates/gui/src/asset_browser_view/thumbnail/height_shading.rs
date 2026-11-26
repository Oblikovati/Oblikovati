// SPDX-License-Identifier: GPL-2.0-only
//! Modulate local preview lighting using procedural or embedded height differences.
use oblikovati_model::assets::{
    AppearanceBump, AppearanceHeightMap, CeramicAppearance, GenericAppearanceEffects,
};

#[derive(Default)]
pub(super) struct HeightShading<'a> {
    wavy: f32,
    bump: Option<(&'a AppearanceHeightMap, f32)>,
    relief: Option<(&'a AppearanceHeightMap, f32)>,
}

impl<'a> HeightShading<'a> {
    pub(super) fn new(
        ceramic: Option<&'a CeramicAppearance>,
        generic: Option<&'a GenericAppearanceEffects>,
    ) -> Self {
        if let Some(ceramic) = ceramic {
            return Self::for_surface(
                ceramic.finish_bumps,
                ceramic.custom_bump.as_ref(),
                ceramic.relief,
                ceramic.relief_image.as_ref(),
            );
        }
        generic
            .and_then(|effects| {
                let surface = effects.rough_surface.as_ref()?;
                Some(Self::for_surface(
                    effects.bump,
                    surface.bump_image.as_ref(),
                    surface.relief,
                    surface.relief_image.as_ref(),
                ))
            })
            .unwrap_or_default()
    }

    fn for_surface(
        bump: AppearanceBump,
        bump_image: Option<&'a AppearanceHeightMap>,
        relief: AppearanceBump,
        relief_image: Option<&'a AppearanceHeightMap>,
    ) -> Self {
        let wavy = if bump.enabled && bump_image.is_none() {
            f32::from(bump.amount) / 100.
        } else {
            0.
        };
        Self {
            wavy,
            bump: image_effect(bump_image, bump),
            relief: image_effect(relief_image, relief),
        }
    }

    pub(super) fn enabled(&self) -> bool {
        self.wavy > 0. || self.bump.is_some() || self.relief.is_some()
    }

    pub(super) fn apply(&self, [x, y]: [f32; 2], diffuse: f32) -> f32 {
        let wave = if self.wavy > 0. {
            (y * 18. + 1.8 * (x * 7.).sin()).cos() * 0.25 * self.wavy
        } else {
            0.
        };
        let image = self
            .bump
            .into_iter()
            .chain(self.relief)
            .map(|(map, amount)| slope(map, [x, y]) * amount)
            .sum::<f32>();
        (diffuse + wave + image).max(0.)
    }
}

fn image_effect(
    map: Option<&AppearanceHeightMap>,
    effect: AppearanceBump,
) -> Option<(&AppearanceHeightMap, f32)> {
    map.filter(|_| effect.enabled && effect.amount > 0)
        .map(|map| (map, f32::from(effect.amount) / 100.))
}

fn slope(map: &AppearanceHeightMap, [x, y]: [f32; 2]) -> f32 {
    let side = AppearanceHeightMap::SIDE;
    let at = [x, y].map(|axis| (axis * 0.5 + 0.5).clamp(0., 1.) * (side - 1) as f32);
    let [u, v] = at.map(|axis| axis.round() as usize);
    let height = |x: usize, y: usize| {
        f32::from(map.samples().get(y * side + x).copied().unwrap_or(0)) / 255.
    };
    let dx = height((u + 1).min(side - 1), v) - height(u.saturating_sub(1), v);
    let dy = height(u, (v + 1).min(side - 1)) - height(u, v.saturating_sub(1));
    dx * 0.34 + dy * 0.42
}
