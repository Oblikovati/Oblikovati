// SPDX-License-Identifier: GPL-2.0-only
use super::{DrawingAnnotationDefinition, DrawingDocument, DrawingError, annotation::unique_name};
use std::f64::consts::PI;

const SCALLOP_MM: f64 = 6.;
const SEGMENTS_PER_SCALLOP: usize = 6;

/// Sheet-space definition for drawing the scalloped boundary and its optional tag.
pub struct DrawingRevisionCloudPresentation<'a> {
    origin_mm: [f64; 2],
    size_mm: [f64; 2],
    pub tag: &'a str,
}

impl DrawingDocument {
    /// Flag a sheet region, e.g. `drawing.add_revision_cloud("Change", [40., 50.], [60., 40.], "A")?`.
    pub fn add_revision_cloud(
        &mut self,
        requested_name: &str,
        [x_mm, y_mm]: [f64; 2],
        [width_mm, height_mm]: [f64; 2],
        tag: &str,
    ) -> Result<DrawingAnnotationDefinition, DrawingError> {
        let sheet = self.active_sheet();
        let name = unique_name(requested_name, &self.sheets()[sheet].annotations);
        let cloud = DrawingAnnotationDefinition::RevisionCloud {
            name,
            x_mm,
            y_mm,
            width_mm,
            height_mm,
            tag: tag.into(),
            characteristic: None,
            material_removal: None,
        };
        self.add_annotation_definition(sheet, cloud.clone())?;
        Ok(cloud)
    }
}

impl DrawingAnnotationDefinition {
    /// Get the annotation's presentation, e.g. `annotation.revision_cloud_presentation()`.
    pub fn revision_cloud_presentation(&self) -> Option<DrawingRevisionCloudPresentation<'_>> {
        let Self::RevisionCloud {
            x_mm,
            y_mm,
            width_mm,
            height_mm,
            tag,
            ..
        } = self
        else {
            return None;
        };
        Some(DrawingRevisionCloudPresentation {
            origin_mm: [*x_mm, *y_mm],
            size_mm: [*width_mm, *height_mm],
            tag,
        })
    }
}

impl DrawingRevisionCloudPresentation<'_> {
    /// Count the nominal curve segments, e.g. `cloud.curve_count()` for a wire reply.
    pub fn curve_count(&self) -> usize {
        nominal_segment_count(self.size_mm) as usize
    }

    /// Place the label above the middle of the upper edge.
    /// Example: `cloud.label_center_mm()` maps to a sheet text anchor.
    pub fn label_center_mm(&self) -> [f64; 2] {
        let [x, y] = self.origin_mm;
        [x + self.size_mm[0] / 2., y + self.size_mm[1] + 6.]
    }

    /// Sample the portions of the scalloped boundary along the paper.
    /// Example: `cloud.visible_segments([420., 297.], 0.3)` for a display at 0.3 mm per pixel.
    pub fn visible_segments(&self, sheet_mm: [f64; 2], mm_per_pixel: f64) -> Vec<[[f64; 2]; 2]> {
        let [x, y] = self.origin_mm;
        let [width, height] = self.size_mm;
        let corners = [
            [x, y],
            [x + width, y],
            [x + width, y + height],
            [x, y + height],
        ];
        let mut segments = Vec::new();
        for index in 0..4 {
            let axis = index % 2;
            let edge = ScallopEdge {
                start: corners[index],
                end: corners[(index + 1) % 4],
                axis,
                direction: if index < 2 { 1. } else { -1. },
                length: self.size_mm[axis],
            };
            edge.append_visible(&mut segments, sheet_mm[axis], mm_per_pixel);
        }
        segments
    }
}

struct ScallopEdge {
    start: [f64; 2],
    end: [f64; 2],
    axis: usize,
    direction: f64,
    length: f64,
}

impl ScallopEdge {
    fn append_visible(
        &self,
        segments: &mut Vec<[[f64; 2]; 2]>,
        sheet_length: f64,
        mm_per_pixel: f64,
    ) {
        let count = scallops(self.length);
        let step = self.length / count;
        // Below a screen pixel the bumps are indistinguishable. A straight edge also bounds
        // the rendering cost when viewing unusually large sheets.
        if step < mm_per_pixel {
            segments.push([self.start, self.end]);
            return;
        }
        let projection = |coordinate| self.direction * (coordinate - self.start[self.axis]);
        let a = projection(0.);
        let b = projection(sheet_length);
        let first = (a.min(b) / step).floor().clamp(0., count) as usize;
        let end = (a.max(b) / step).ceil().clamp(0., count) as usize;
        for scallop in first..end {
            let mut from = self.point(scallop, step, 0);
            for index in 1..=SEGMENTS_PER_SCALLOP {
                let to = self.point(scallop, step, index);
                segments.push([from, to]);
                from = to;
            }
        }
    }

    fn point(&self, scallop: usize, step: f64, index: usize) -> [f64; 2] {
        let theta = PI * index as f64 / SEGMENTS_PER_SCALLOP as f64;
        let radius = 0.5 * step;
        let distance = scallop as f64 * step + radius * (1. - theta.cos());
        let normal = if self.axis == 0 {
            -self.direction
        } else {
            self.direction
        };
        let mut point = self.start;
        point[self.axis] += self.direction * distance;
        point[1 - self.axis] += normal * radius * theta.sin();
        point
    }
}

fn scallops(length: f64) -> f64 {
    (length / SCALLOP_MM).round().max(1.)
}

fn nominal_segment_count([width, height]: [f64; 2]) -> f64 {
    2. * (scallops(width) + scallops(height)) * SEGMENTS_PER_SCALLOP as f64
}

pub(super) fn validate_cloud(
    origin: [f64; 2],
    size: [f64; 2],
    tag: &str,
) -> Result<(), DrawingError> {
    if !origin.into_iter().zip(size).all(|(start, length)| {
        length > 0. && (start + length).is_finite() && start + length > start
    }) || nominal_segment_count(size) >= (usize::MAX as f64).min(i64::MAX as f64)
    {
        return Err(DrawingError(format!(
            "revision cloud origin {origin:?}, size {size:?}: expected finite positive extents and a representable curve count"
        )));
    }
    if tag.contains('\0') {
        return Err(DrawingError(format!(
            "revision cloud tag {tag:?}: expected text without NUL"
        )));
    }
    Ok(())
}
