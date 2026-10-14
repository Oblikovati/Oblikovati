// SPDX-License-Identifier: GPL-2.0-only
use crate::sketch::{Sketch2D, SketchEntityId};
use std::collections::BTreeSet;
impl Sketch2D {
    pub(super) fn retain_surviving_geometry(&mut self, retired: &BTreeSet<SketchEntityId>) {
        self.points.retain(|point| !retired.contains(&point.id));
        self.lines.retain(|line| !retired.contains(&line.id));
        self.circles.retain(|circle| !retired.contains(&circle.id));
        self.arcs.retain(|arc| !retired.contains(&arc.id));
        self.ellipses
            .retain(|ellipse| !retired.contains(&ellipse.id()));
        self.splines
            .retain(|parent| !retired.contains(&parent.id()));
        self.spline_handles.retain(|handle| {
            !retired.contains(&handle.parent()) && !retired.contains(&handle.end())
        });
        self.offset_splines
            .retain(|offset| !retired.contains(&offset.id()));
        self.entity_order.retain(|id| !retired.contains(id));
        self.defining_points.retain(|id| !retired.contains(id));
        self.construction_curves.retain(|id| !retired.contains(id));
        self.centerlines.retain(|id| !retired.contains(id));
    }
}
