// SPDX-License-Identifier: GPL-2.0-only
//! Ring geometry follows the Go marking menu and uses live slot dimensions (#4055).
const DIRECTIONS: [[f32; 2]; 8] = [
    [0., -1.],
    [0.7, -0.7],
    [1., 0.],
    [0.7, 0.7],
    [0., 1.],
    [-0.7, 0.7],
    [-1., 0.],
    [-0.7, -0.7],
];
const MIN_RADIUS: f32 = 72.;
const PADDING: f32 = 12.;
const GAP: f32 = 8.;

pub(crate) struct RingLayout {
    pub(crate) size: f32,
    radius: f32,
}
impl RingLayout {
    pub(crate) fn new(maximum: [f32; 2]) -> Self {
        let [width, height] = maximum.map(|value| value.max(0.));
        let radius = required_radius([width + GAP, height + GAP]);
        Self {
            size: 2. * radius + width.max(height) + 2. * PADDING,
            radius,
        }
    }
    pub(crate) fn position(&self, slot: u8, size: [f32; 2]) -> Option<[f32; 2]> {
        let direction = DIRECTIONS.get(usize::from(slot))?;
        Some(std::array::from_fn(|axis| {
            self.size / 2. + direction[axis] * self.radius - size[axis] / 2.
        }))
    }
}
fn required_radius(span: [f32; 2]) -> f32 {
    DIRECTIONS
        .iter()
        .zip(DIRECTIONS.iter().cycle().skip(1))
        .take(8)
        .fold(MIN_RADIUS, |radius, (first, second)| {
            radius.max(pair_radius(*first, *second, span))
        })
}
fn pair_radius(first: [f32; 2], second: [f32; 2], span: [f32; 2]) -> f32 {
    (0..2)
        .filter_map(|axis| {
            let distance = (first[axis] - second[axis]).abs();
            (distance > 0.).then(|| span[axis] / distance)
        })
        .fold(f32::INFINITY, f32::min)
}

pub(crate) fn popup_origin(
    pointer: [f32; 2],
    bounds: [[f32; 2]; 2],
    ring: f32,
    above: f32,
    below: f32,
    padding: [f32; 2],
) -> [f32; 2] {
    let size = [
        ring + 2. * padding[0],
        above + ring + below + 2. * padding[1],
    ];
    let desired = [
        pointer[0] - ring / 2. - padding[0],
        pointer[1] - above - ring / 2. - padding[1],
    ];
    clamp_origin(desired, bounds, size)
}
fn clamp_origin(desired: [f32; 2], bounds: [[f32; 2]; 2], size: [f32; 2]) -> [f32; 2] {
    std::array::from_fn(|axis| {
        desired[axis].clamp(
            bounds[0][axis],
            (bounds[1][axis] - size[axis]).max(bounds[0][axis]),
        )
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn popup_centers_the_ring_and_clamps_to_the_display() {
        let origin = popup_origin(
            [500., 400.],
            [[0., 0.], [1000., 800.]],
            200.,
            60.,
            40.,
            [8., 8.],
        );
        assert_eq!(origin, [392., 232.]);
        assert_eq!(
            popup_origin(
                [500., 200.],
                [[300., 180.], [1000., 800.]],
                200.,
                60.,
                40.,
                [8., 8.]
            ),
            [392., 180.]
        );
        assert_eq!(
            popup_origin(
                [0., 0.],
                [[0., 0.], [1000., 800.]],
                200.,
                60.,
                40.,
                [8., 8.]
            ),
            [0., 0.]
        );
        assert_eq!(
            popup_origin(
                [999., 799.],
                [[0., 0.], [1000., 800.]],
                200.,
                60.,
                40.,
                [8., 8.]
            ),
            [784., 484.]
        );
    }
    #[test]
    fn slots_stay_inside_and_do_not_overlap_at_different_scales() {
        for size in [[32., 20.], [160., 45.], [400., 100.]] {
            let layout = RingLayout::new(size);
            let positions: Vec<_> = (0..8)
                .map(|slot| layout.position(slot, size).unwrap())
                .collect();
            for (index, first) in positions.iter().enumerate() {
                assert!(first[0] >= 0. && first[1] >= 0.);
                assert!(first[0] + size[0] <= layout.size && first[1] + size[1] <= layout.size);
                for second in positions.iter().skip(index + 1) {
                    assert!(
                        (first[0] - second[0]).abs() >= size[0]
                            || (first[1] - second[1]).abs() >= size[1]
                    );
                }
            }
            assert!(layout.position(8, size).is_none());
        }
    }
}
