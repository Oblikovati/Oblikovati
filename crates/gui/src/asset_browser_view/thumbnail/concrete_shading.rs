// SPDX-License-Identifier: GPL-2.0-only
//! Sample concrete tooling and weathering on the editor and browser preview surfaces.
use super::height_shading::{height_at, slope};
use oblikovati_model::assets::{ConcreteAppearance, ConcreteFinish, ConcreteSealant};

pub(super) fn shade(concrete: &ConcreteAppearance, [x, y]: [f32; 2], light: f32) -> f32 {
    let grooves = match concrete.finish {
        ConcreteFinish::BroomStraight => (y * 44.).cos() * 0.075,
        ConcreteFinish::BroomCurved => (y * 42. + 5. * x * x).cos() * 0.075,
        ConcreteFinish::Smooth | ConcreteFinish::Polished => 0.,
        ConcreteFinish::Stamped => concrete
            .stamped_image
            .as_ref()
            .map_or_else(|| stamped([x, y]), |image| slope(image, [x, y]) * 0.5),
    };
    let coating = match concrete.sealant {
        ConcreteSealant::None => 1.,
        ConcreteSealant::Epoxy => 0.83,
        ConcreteSealant::Acrylic => 0.92,
    };
    ((light + grooves) * coating * weathering(concrete, [x, y])).max(0.)
}

fn stamped([x, y]: [f32; 2]) -> f32 {
    let band = ((y + 1.) * 3.).floor();
    let u = ((x + 1.) * 3. + band.rem_euclid(2.) * 0.5).fract();
    let v = ((y + 1.) * 3.).fract();
    if u < 0.09 || v < 0.09 { -0.16 } else { 0.025 }
}

fn weathering(concrete: &ConcreteAppearance, [x, y]: [f32; 2]) -> f32 {
    if !concrete.weathering {
        return 1.;
    }
    let stain = concrete.weathering_image.as_ref().map_or_else(
        || 0.5 + (x * 13. + y * 8.).sin() * (y * 17. - x * 3.).cos() * 0.5,
        |map| height_at(map, [x, y]),
    );
    1. - 0.24 * stain
}
