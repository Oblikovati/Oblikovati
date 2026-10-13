// SPDX-License-Identifier: GPL-2.0-only
use super::*;
pub(super) fn field(
    label: &str,
    value: f64,
    witness: [[f64; 2]; 2],
    binding: SketchPlacementBinding,
) -> SketchPlacementField {
    SketchPlacementField {
        label: label.into(),
        value,
        witness,
        binding,
    }
}
pub(super) fn distance([a, b]: [[f64; 2]; 2]) -> f64 {
    let delta = [b[0] - a[0], b[1] - a[1]];
    (delta[0] * delta[0] + delta[1] * delta[1]).sqrt()
}
