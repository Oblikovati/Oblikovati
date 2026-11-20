// SPDX-License-Identifier: GPL-2.0-only
//! Positions for placement dimension text.
/// A witness's middle, moved off it by a fraction of its length so the text clears the line.
pub(super) fn annotation_beside([from, to]: [[f64; 2]; 2]) -> [f64; 2] {
    const SIDE: f64 = 0.2;
    let delta = [to[0] - from[0], to[1] - from[1]];
    [
        (from[0] + to[0]) * 0.5 - delta[1] * SIDE,
        (from[1] + to[1]) * 0.5 + delta[0] * SIDE,
    ]
}
/// Past a chord's middle, away from the center it turns about.
pub(super) fn annotation_outside(center: [f64; 2], [from, to]: [[f64; 2]; 2]) -> [f64; 2] {
    const REACH: f64 = 1.6;
    let middle = [(from[0] + to[0]) * 0.5, (from[1] + to[1]) * 0.5];
    [
        center[0] + (middle[0] - center[0]) * REACH,
        center[1] + (middle[1] - center[1]) * REACH,
    ]
}
