// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{
    feature::{
        FeatureReadKey,
        work_geometry::plane_key,
        work_plane::{WorkPlaneReference, WorkPlaneResolver},
    },
    sketch::SketchPlane,
};

/// Associative bounds projected onto the profile normal, in document units.
#[derive(Clone, Debug, Eq, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "kebab-case", deny_unknown_fields)]
pub enum SketchPrismExtent {
    ToFace {
        to: WorkPlaneReference,
    },
    FromTo {
        from: WorkPlaneReference,
        to: WorkPlaneReference,
    },
}

impl SketchPrismDefinition {
    /// Author an extrusion bounded by planes. Example: `SketchPrismDefinition::between_planes(sketch, selection, extent)?`.
    pub fn between_planes(
        sketch: DocumentSketchId,
        selection: SketchPrismSelection,
        extent: SketchPrismExtent,
    ) -> Result<Self, SketchPrismError> {
        let definition = Self {
            sketch,
            selection,
            displacement: [DatumLength::Constant(0.); 3],
            direction: SketchPrismDirection::Positive,
            second_depth: None,
            extent: Some(extent),
        };
        validation::validate_sources(&definition).map_err(|cause| definition.decline(cause))?;
        Ok(definition)
    }
}

impl SketchPrismExtent {
    pub(super) fn reads(&self) -> impl Iterator<Item = FeatureReadKey> + '_ {
        self.references()
            .into_iter()
            .flatten()
            .filter_map(plane_key)
    }

    fn references(&self) -> [Option<&WorkPlaneReference>; 2] {
        match self {
            Self::ToFace { to } => [None, Some(to)],
            Self::FromTo { from, to } => [Some(from), Some(to)],
        }
    }

    pub(super) fn span(
        &self,
        sketch: SketchPlane,
        resolver: &impl WorkPlaneResolver,
    ) -> Result<(f64, f64), SketchPrismCause> {
        let [start, end] = self.references().map(|reference| {
            reference
                .map(|plane| plane_offset(sketch, plane, resolver))
                .transpose()
        });
        let start = start?.unwrap_or(0.);
        let end = end?.unwrap_or(0.);
        let depth = (end - start).abs();
        if !depth.is_finite() || depth == 0. {
            return Err(SketchPrismCause::Definition(format!(
                "extent planes at offsets {start:?} and {end:?}: expected distinct finite bounds"
            )));
        }
        Ok((start.min(end), depth))
    }
}

fn plane_offset(
    sketch: SketchPlane,
    reference: &WorkPlaneReference,
    resolver: &impl WorkPlaneResolver,
) -> Result<f64, SketchPrismCause> {
    let target = reference
        .clone()
        .frame(resolver)
        .map_err(SketchPrismCause::Extent)?;
    let normal = sketch.normal();
    let other = target.normal();
    let cross: [f64; 3] = std::array::from_fn(|axis| {
        let i = (axis + 1) % 3;
        let j = (axis + 2) % 3;
        normal[i] * other[j] - normal[j] * other[i]
    });
    if cross.into_iter().map(|v| v * v).sum::<f64>().sqrt() > sketch.options().angular_deviation {
        return Err(SketchPrismCause::Definition(format!(
            "extent plane {reference:?}: expected a plane parallel to the profile"
        )));
    }
    let origin = sketch.recipe().origin;
    let target = target.recipe().origin;
    Ok((0..3)
        .map(|axis| (target[axis] - origin[axis]) * normal[axis])
        .sum())
}
