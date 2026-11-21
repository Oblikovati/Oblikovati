// SPDX-License-Identifier: GPL-2.0-only
//! Document-owned spatial sketch points and lines.
use std::collections::BTreeSet;

use crate::{document_sketch::DocumentSketchId, sketch::SketchEntityId};

/// A spatial point in the document's working length unit.
#[derive(Clone, Copy, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SpatialSketchPoint {
    pub id: SketchEntityId,
    pub position: [f64; 3],
}

/// A spatial line references two points in the same sketch.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SpatialSketchLine {
    pub id: SketchEntityId,
    pub endpoints: [SketchEntityId; 2],
}

/// Validated spatial point and line geometry for one document sketch.
#[derive(Clone, Debug, PartialEq)]
pub struct SpatialSketch {
    id: DocumentSketchId,
    name: Option<String>,
    points: Vec<SpatialSketchPoint>,
    lines: Vec<SpatialSketchLine>,
}

/// Serialized spatial geometry, validated when restored into a document.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SpatialSketchRecipe {
    pub id: DocumentSketchId,
    pub name: Option<String>,
    pub points: Vec<SpatialSketchPoint>,
    pub lines: Vec<SpatialSketchLine>,
}

impl SpatialSketch {
    /// Validate a sketch's identity, coordinates, and references.
    /// Example: `SpatialSketch::new(id, Some("Sketch1".into()), points, lines)?`.
    pub fn new(
        id: DocumentSketchId,
        name: Option<String>,
        points: Vec<SpatialSketchPoint>,
        lines: Vec<SpatialSketchLine>,
    ) -> Result<Self, String> {
        if id.0 == 0 {
            return Err(format!(
                "spatial sketch {id:?}: expected a nonzero identity"
            ));
        }
        validate_geometry(&points, &lines)?;
        Ok(Self {
            id,
            name,
            points,
            lines,
        })
    }

    /// Compare authored coordinates by their exact bits; example: `first.same_recipe(&second)`.
    pub fn same_recipe(&self, other: &Self) -> bool {
        self.id == other.id
            && self.name == other.name
            && self.lines == other.lines
            && self.points.len() == other.points.len()
            && self.points.iter().zip(&other.points).all(|(left, right)| {
                left.id == right.id
                    && left
                        .position
                        .iter()
                        .zip(right.position)
                        .all(|(a, b)| a.to_bits() == b.to_bits())
            })
    }

    /// Capture validated geometry; example: `sketch.recipe()`.
    pub fn recipe(&self) -> SpatialSketchRecipe {
        SpatialSketchRecipe {
            id: self.id,
            name: self.name.clone(),
            points: self.points.clone(),
            lines: self.lines.clone(),
        }
    }

    /// Validate saved geometry; example: `SpatialSketch::from_recipe(&recipe)?`.
    pub fn from_recipe(recipe: &SpatialSketchRecipe) -> Result<Self, String> {
        Self::new(
            recipe.id,
            recipe.name.clone(),
            recipe.points.clone(),
            recipe.lines.clone(),
        )
    }

    /// Return the document sketch identity; example: `sketch.id()`.
    pub fn id(&self) -> DocumentSketchId {
        self.id
    }

    /// Return the source name when known; example: `sketch.name()`.
    pub fn name(&self) -> Option<&str> {
        self.name.as_deref()
    }

    /// Return points in source order; example: `sketch.points()`.
    pub fn points(&self) -> &[SpatialSketchPoint] {
        &self.points
    }

    /// Return lines in source order; example: `sketch.lines()`.
    pub fn lines(&self) -> &[SpatialSketchLine] {
        &self.lines
    }
}

fn validate_geometry(
    points: &[SpatialSketchPoint],
    lines: &[SpatialSketchLine],
) -> Result<(), String> {
    let mut identities = BTreeSet::new();
    for point in points {
        validate_identity(&mut identities, point.id)?;
        if !point.position.iter().all(|value| value.is_finite()) {
            return Err(format!(
                "spatial point {:?} position {:?}: expected three finite coordinates",
                point.id, point.position
            ));
        }
    }
    let point_ids = identities.clone();
    for line in lines {
        validate_identity(&mut identities, line.id)?;
        validate_line_endpoints(line, &point_ids)?;
    }
    Ok(())
}

fn validate_identity(
    seen: &mut BTreeSet<SketchEntityId>,
    id: SketchEntityId,
) -> Result<(), String> {
    if id.0 == 0 || !seen.insert(id) {
        return Err(format!(
            "spatial sketch entity {id:?}: expected a unique nonzero identity"
        ));
    }
    Ok(())
}

fn validate_line_endpoints(
    line: &SpatialSketchLine,
    points: &BTreeSet<SketchEntityId>,
) -> Result<(), String> {
    for endpoint in line.endpoints {
        if !points.contains(&endpoint) {
            return Err(format!(
                "spatial line {:?} endpoint {endpoint:?}: expected a point in the same sketch",
                line.id
            ));
        }
    }
    Ok(())
}
