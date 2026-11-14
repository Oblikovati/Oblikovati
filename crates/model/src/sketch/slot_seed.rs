// SPDX-License-Identifier: GPL-2.0-only
use super::{SketchError, SketchPathMember, require_position};
mod curved;
mod straight;
/// Seed placements for one slot; these author geometry without adding design constraints.
#[derive(Clone, Copy, Debug, PartialEq)]
pub enum SketchSlotPlacement {
    Centers([[f64; 2]; 2]),
    Overall([[f64; 2]; 2]),
    SlotCenter([[f64; 2]; 2]),
    /// The end supplies a direction; both cap centres use the start radius.
    Arc {
        center: [f64; 2],
        endpoints: [[f64; 2]; 2],
        counter_clockwise: bool,
    },
    ArcCenterPoint {
        center: [f64; 2],
        start: [f64; 2],
        sweep: f64,
    },
}
/// Ordered seed graph; each index refers to a single shared defining point.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchSlotLayout {
    pub points: Vec<[f64; 2]>,
    pub members: [SketchPathMember; 4],
}
impl SketchSlotPlacement {
    /// Prepare finite seeds before allocating IDs; example: `placement.layout(2.0)?`.
    pub fn layout(self, width: f64) -> Result<SketchSlotLayout, SketchError> {
        if !width.is_finite() || width <= 0.0 || width / 2.0 == 0.0 {
            return Err(SketchError::Definition(format!(
                "slot width {width}: expected a positive finite length with a representable half-width"
            )));
        }
        let layout = self.construct_layout(width)?;
        for point in &layout.points {
            require_position(*point)?;
        }
        layout.require_caps()?;
        Ok(layout)
    }
    fn construct_layout(self, width: f64) -> Result<SketchSlotLayout, SketchError> {
        match self {
            Self::Centers(pair) => straight::centers(pair, width),
            Self::Overall(pair) => straight::overall(pair, width),
            Self::SlotCenter(pair) => straight::slot_center(pair, width),
            Self::Arc {
                center,
                endpoints,
                counter_clockwise,
            } => curved::arc(center, endpoints, width, counter_clockwise),
            Self::ArcCenterPoint {
                center,
                start,
                sweep,
            } => curved::center_point(center, start, sweep, width),
        }
    }
}
fn direction(first: [f64; 2], last: [f64; 2]) -> Result<([f64; 2], f64), SketchError> {
    require_position(first)?;
    require_position(last)?;
    let delta = [last[0] - first[0], last[1] - first[1]];
    // Preserve the pinned Go seed arithmetic; this is not a topology predicate.
    let length = (delta[0] * delta[0] + delta[1] * delta[1]).sqrt();
    if !length.is_finite() || length <= 0.0 {
        return Err(SketchError::Definition(format!(
            "slot direction {first:?} -> {last:?}, length {length}: expected a positive finite representable length"
        )));
    }
    Ok((delta.map(|value| value * (1.0 / length)), length))
}
fn shifted(point: [f64; 2], offset: [f64; 2]) -> [f64; 2] {
    [point[0] + offset[0], point[1] + offset[1]]
}
fn arc_member(center: usize, endpoints: [usize; 2], counter_clockwise: bool) -> SketchPathMember {
    SketchPathMember::Arc {
        center,
        endpoints,
        counter_clockwise,
    }
}

impl SketchSlotLayout {
    fn require_caps(&self) -> Result<(), SketchError> {
        for index in [1, 3] {
            if let SketchPathMember::Arc {
                center, endpoints, ..
            } = self.members[index]
            {
                self.require_cap(index, center, endpoints)?;
            }
        }
        Ok(())
    }
    fn require_cap(
        &self,
        index: usize,
        center: usize,
        endpoints: [usize; 2],
    ) -> Result<(), SketchError> {
        direction(self.points[center], self.points[endpoints[0]])?;
        direction(self.points[center], self.points[endpoints[1]])?;
        if self.points[endpoints[0]] == self.points[endpoints[1]] {
            return Err(SketchError::Definition(format!(
                "slot cap {index}, points {:?}: expected distinct representable endpoints",
                self.points
            )));
        }
        Ok(())
    }
}
