// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::*;
use std::f64::consts::TAU;

pub(super) fn characteristic_glyph(
    characteristic: GeometricCharacteristic,
    [x, y]: [f64; 2],
    r: f64,
) -> Vec<[[f64; 2]; 2]> {
    let horizontal = [[x - r * 1.4, y], [x + r * 1.4, y]];
    match characteristic {
        CHARACTERISTIC_POSITION => circle_segments([x, y], r)
            .chain([horizontal, [[x, y - r * 1.4], [x, y + r * 1.4]]])
            .collect(),
        CHARACTERISTIC_CIRCULARITY => circle_segments([x, y], r).collect(),
        CHARACTERISTIC_CONCENTRICITY => circle_segments([x, y], r)
            .chain(circle_segments([x, y], r * 0.55))
            .collect(),
        CHARACTERISTIC_STRAIGHTNESS => vec![horizontal],
        CHARACTERISTIC_FLATNESS => flatness([x, y], r),
        CHARACTERISTIC_PERPENDICULARITY => {
            vec![[[x, y + r], [x, y - r]], [[x - r, y - r], [x + r, y - r]]]
        }
        CHARACTERISTIC_PARALLELISM => vec![
            [[x - r * 0.2, y + r], [x - r, y - r]],
            [[x + r, y + r], [x + r * 0.2, y - r]],
        ],
        CHARACTERISTIC_ANGULARITY => vec![
            [[x - r, y - r], [x + r, y + r]],
            [[x - r, y - r], [x + r, y - r]],
        ],
        _ => Vec::new(),
    }
}

fn flatness([x, y]: [f64; 2], r: f64) -> Vec<[[f64; 2]; 2]> {
    let top_left = [x - r * 0.4, y + r];
    let top_right = [x + r, y + r];
    let bottom_right = [x + r * 0.4, y - r];
    let bottom_left = [x - r, y - r];
    vec![
        [top_left, top_right],
        [top_right, bottom_right],
        [bottom_right, bottom_left],
        [bottom_left, top_left],
    ]
}

pub(super) fn characteristic_text(characteristic: GeometricCharacteristic) -> &'static str {
    match characteristic {
        CHARACTERISTIC_CYLINDRICITY => "CYL",
        CHARACTERISTIC_PROFILE_OF_ANY_LINE => "PROL",
        CHARACTERISTIC_PROFILE_OF_ANY_SURFACE => "PROS",
        CHARACTERISTIC_SYMMETRY => "SYM",
        CHARACTERISTIC_CIRCULAR_RUNOUT => "RUN",
        CHARACTERISTIC_TOTAL_RUNOUT => "TRUN",
        _ => "GD&T",
    }
}

pub(super) fn circle_segments(
    [x, y]: [f64; 2],
    radius: f64,
) -> impl Iterator<Item = [[f64; 2]; 2]> {
    const SEGMENTS: u32 = 24;
    let point = move |index: u32| {
        let angle = TAU * f64::from(index % SEGMENTS) / f64::from(SEGMENTS);
        [x + radius * angle.cos(), y + radius * angle.sin()]
    };
    (0..SEGMENTS).map(move |index| [point(index), point(index + 1)])
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn characteristic_line_symbols_fit_inside_an_eight_millimetre_cell() {
        for (characteristic, count) in [
            (CHARACTERISTIC_POSITION, 26),
            (CHARACTERISTIC_CIRCULARITY, 24),
            (CHARACTERISTIC_CONCENTRICITY, 48),
            (CHARACTERISTIC_STRAIGHTNESS, 1),
            (CHARACTERISTIC_FLATNESS, 4),
            (CHARACTERISTIC_PERPENDICULARITY, 2),
            (CHARACTERISTIC_PARALLELISM, 2),
            (CHARACTERISTIC_ANGULARITY, 2),
        ] {
            let lines = characteristic_glyph(characteristic, [0., 0.], 2.2);
            assert_eq!(lines.len(), count);
            assert!(lines.into_iter().flatten().flatten().all(|p| p.abs() < 4.));
        }
    }
}
