// SPDX-License-Identifier: GPL-2.0-only
use super::{DatumLength, WorkGeometryDefinition, WorkPointReference};
use crate::feature::work_plane::WorkPlaneReference;
use crate::feature::{FeatureReadFootprint, FeatureReadKey};

impl WorkGeometryDefinition {
    /// Attribute every authored datum input; example: `definition.reads()` before program insertion.
    pub fn reads(&self) -> FeatureReadFootprint {
        let keys = match self {
            Self::DeleteBody(definition) => return definition.reads(),
            Self::Chamfer(definition) => return definition.reads(),
            Self::Fillet(definition) => return definition.reads(),
            Self::Hole(definition) => return definition.reads(),
            Self::Slot(definition) => return definition.reads(),
            Self::Combine(definition) => return definition.reads(),
            Self::DirectEdit(definition) => return definition.reads(),
            Self::Draft(definition) => return definition.reads(),
            Self::Thread(definition) => return definition.reads(),
            Self::DeleteFace(definition) => return definition.reads(),
            Self::ThickenOffset(definition) => return definition.reads(),
            Self::Split(definition) => return definition.reads(),
            Self::Finish(definition) => return definition.reads(),
            Self::ExtendSurface(definition) => return definition.reads(),
            Self::Shell(definition) => return definition.reads(),
            Self::Mirror(definition) => return definition.reads(),
            Self::CircularPattern(definition) => return definition.reads(),
            Self::RectangularPattern(definition) => return definition.reads(),
            Self::MoveBody(definition) => return definition.reads(),
            Self::SketchLoft(definition) => return definition.source_reads(),
            Self::SketchRevolve(definition) => return definition.source_reads(),
            Self::SketchCoil(definition) => return definition.source_reads(),
            Self::SketchEmboss(definition) => return definition.source_reads(),
            Self::SketchRib(definition) => return definition.source_reads(),
            Self::SketchDecal(definition) => return definition.source_reads(),
            Self::DerivedPart(definition) => return definition.source_reads(),
            Self::ImportedModel(definition) => return definition.source_reads(),
            Self::Unwrap(definition) => return definition.source_reads(),
            Self::SketchSweep(definition) => return definition.source_reads(),
            Self::SketchPrism(definition) => {
                definition.source_reads().keys().iter().copied().collect()
            }
            Self::Offset { base, distance } => offset_keys(base.clone(), *distance),
            Self::ParallelThroughPoint(source) => parallel_keys(source.clone()),
            Self::LineThroughPoint(source) | Self::NormalToAxis(source) => line_point_keys(*source),
            Self::AtAngleAboutAxis(source) => line_angle_keys(source.clone()),
            Self::ThroughTwoAxes(axes) => axes.iter().copied().filter_map(axis_key).collect(),
            Self::ThroughThreePoints(points) => {
                points.iter().copied().filter_map(point_key).collect()
            }
            Self::WorkPlaneExtended(definition) => return definition.reads(),
            Self::PointAtReference(reference) => point_key(*reference).into_iter().collect(),
            Self::Point(coordinates) => coordinate_keys(coordinates),
            Self::WorkPoint(definition) => return definition.reads(),
            Self::AxisThroughPoints(points) => point_pair_keys(*points),
            Self::Ucs(definition) => return definition.reads(),
            _ => Vec::new(),
        };
        FeatureReadFootprint::new(keys, false)
    }
}
fn length_key(length: DatumLength) -> Option<FeatureReadKey> {
    length.parameter().map(FeatureReadKey::Parameter)
}
fn point_key(reference: WorkPointReference) -> Option<FeatureReadKey> {
    match reference {
        WorkPointReference::Origin => None,
        WorkPointReference::Feature(id) => Some(FeatureReadKey::Feature(id)),
    }
}
pub(crate) fn plane_key(base: &WorkPlaneReference) -> Option<FeatureReadKey> {
    match base {
        WorkPlaneReference::Origin(_) => None,
        WorkPlaneReference::Feature(id) => Some(FeatureReadKey::Feature(*id)),
        WorkPlaneReference::Face(reference) | WorkPlaneReference::GoPrismFace(reference) => {
            Some(FeatureReadKey::Feature(reference.source()))
        }
    }
}
fn offset_keys(base: WorkPlaneReference, distance: DatumLength) -> Vec<FeatureReadKey> {
    plane_key(&base)
        .into_iter()
        .chain(length_key(distance))
        .collect()
}
fn parallel_keys(source: super::WorkPlaneThroughPoint) -> Vec<FeatureReadKey> {
    plane_key(&source.base)
        .into_iter()
        .chain(point_key(source.point))
        .collect()
}

fn line_point_keys(source: super::WorkPlaneLinePoint) -> Vec<FeatureReadKey> {
    axis_key(source.line)
        .into_iter()
        .chain(point_key(source.point))
        .collect()
}
fn axis_key(reference: crate::feature::work_axis::WorkAxisReference) -> Option<FeatureReadKey> {
    match reference {
        crate::feature::work_axis::WorkAxisReference::Feature(id) => {
            Some(FeatureReadKey::Feature(id))
        }
        _ => None,
    }
}

fn line_angle_keys(source: super::WorkPlaneLineAngle) -> Vec<FeatureReadKey> {
    axis_key(source.line)
        .into_iter()
        .chain(plane_key(&source.base))
        .chain(source.angle.parameter_id().map(FeatureReadKey::Parameter))
        .collect()
}

fn coordinate_keys(coordinates: &[DatumLength; 3]) -> Vec<FeatureReadKey> {
    coordinates.iter().copied().filter_map(length_key).collect()
}
fn point_pair_keys(points: super::WorkAxisPoints) -> Vec<FeatureReadKey> {
    [points.first, points.second]
        .into_iter()
        .filter_map(point_key)
        .collect()
}
