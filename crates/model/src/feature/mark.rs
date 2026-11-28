// SPDX-License-Identifier: GPL-2.0-only
//! Authored Part Modify Mark feature recipe, validation, and 3D curve resolution (#3918).
use super::{
    FeatureId, FeatureReadFootprint, FeatureReadKey, fillet::reference_identity,
    work_geometry::WorkGeometryEvaluation,
};
use crate::{
    document_sketch::DocumentSketchId,
    history::DocumentSnapshot,
    sketch::{Sketch2D, SketchEntityId, SketchPlane},
};
use std::collections::BTreeSet;

/// Marking depth mode applied to one geometry set.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum MarkType {
    /// Etch or imprint onto the target surface only.
    #[default]
    MarkSurface,
    /// Project and mark through the target solid onto opposing faces.
    MarkThrough,
}

impl MarkType {
    /// User-facing label shown in the Mark Properties panel dropdown.
    pub fn label(self) -> &'static str {
        match self {
            Self::MarkSurface => "Mark Surface",
            Self::MarkThrough => "Mark Through",
        }
    }
}

/// Target surface projection method for Mark.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum MarkMethod {
    /// Method 1: project sketch geometry onto the closest solid face.
    #[default]
    ProjectToClosestFace,
    /// Method 2: wrap sketch geometry onto explicitly selected faces (with optional face chain).
    WrapToFace,
}

impl MarkMethod {
    /// User-facing label shown in the Mark Properties panel.
    pub fn label(self) -> &'static str {
        match self {
            Self::ProjectToClosestFace => "Project to Closest Face",
            Self::WrapToFace => "Wrap to Face",
        }
    }
}

/// Projection direction relative to the source sketch plane normal.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum MarkDirection {
    #[default]
    Positive,
    Negative,
    Symmetric,
}

/// One authored Mark Geometry Set referencing sketch entities and its marking type.
#[derive(Clone, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct MarkGeometrySet {
    pub sketch: DocumentSketchId,
    pub curves: Vec<SketchEntityId>,
    #[serde(default)]
    pub mark_type: MarkType,
}

/// Persisted Part Modify Mark recipe (source version 54 / Part program version 64).
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct MarkRecipe {
    pub version: u32,
    pub body: Vec<u8>,
    pub geometry_sets: Vec<MarkGeometrySet>,
    #[serde(default)]
    pub method: MarkMethod,
    #[serde(default)]
    pub faces: Vec<Vec<u8>>,
    #[serde(default)]
    pub automatic_face_chain: bool,
    #[serde(default)]
    pub direction: MarkDirection,
}

/// Validated authored Mark definition.
#[derive(Clone, Debug, PartialEq)]
pub struct MarkDefinition {
    recipe: MarkRecipe,
    body_source: FeatureId,
    face_sources: Vec<FeatureId>,
}

/// Resolved 3D geometry for one Mark Geometry Set ready for Martensite evaluation.
#[derive(Clone, Debug, PartialEq)]
pub struct ResolvedMarkGeometrySet {
    pub mark_type: MarkType,
    pub plane: SketchPlane,
    pub polylines_3d: Vec<Vec<[f64; 3]>>,
}

/// Resolved Mark inputs ready for certified Martensite evaluation (model units: cm).
#[derive(Clone, Debug, PartialEq)]
pub struct ResolvedMark {
    pub body: Vec<u8>,
    pub geometry_sets: Vec<ResolvedMarkGeometrySet>,
    pub method: MarkMethod,
    pub faces: Vec<Vec<u8>>,
    pub automatic_face_chain: bool,
    pub direction: MarkDirection,
}

impl MarkDefinition {
    /// Validate and construct a Mark definition from its serialized recipe.
    /// Example: `MarkDefinition::from_recipe(recipe)?`.
    pub fn from_recipe(recipe: MarkRecipe) -> Result<Self, String> {
        if recipe.version != 1 {
            return Err(format!(
                "mark recipe version {}: expected 1",
                recipe.version
            ));
        }
        let (document, body_source) = reference_identity(&recipe.body, 4, "mark body")?;
        if recipe.geometry_sets.is_empty() {
            return Err("mark geometry sets: expected at least one geometry set".into());
        }
        for (index, set) in recipe.geometry_sets.iter().enumerate() {
            if set.sketch.0 == 0 {
                return Err(format!(
                    "mark geometry set {index} sketch {:?}: expected nonzero sketch id",
                    set.sketch
                ));
            }
            if set.curves.is_empty() {
                return Err(format!(
                    "mark geometry set {index}: expected at least one sketch curve"
                ));
            }
            let mut unique = BTreeSet::new();
            for curve in &set.curves {
                if curve.0 == 0 || !unique.insert(*curve) {
                    return Err(format!(
                        "mark geometry set {index} curve {curve:?}: expected unique nonzero curve id"
                    ));
                }
            }
        }
        if recipe.method == MarkMethod::WrapToFace && recipe.faces.is_empty() {
            return Err(
                "mark wrap faces: expected at least one target face for Wrap to Face".into(),
            );
        }
        let mut face_sources = Vec::with_capacity(recipe.faces.len());
        let mut seen_faces = BTreeSet::new();
        for face in &recipe.faces {
            let (face_doc, face_source) = reference_identity(face, 1, "mark face")?;
            if face_doc != document {
                return Err(format!(
                    "mark face document {face_doc:?}: expected body document {document:?}"
                ));
            }
            if !seen_faces.insert(face.as_slice()) {
                return Err("mark faces: expected unique face reference keys".into());
            }
            face_sources.push(face_source);
        }
        Ok(Self {
            recipe,
            body_source,
            face_sources,
        })
    }

    /// Copy authored inputs for persistence; example: `mark.recipe()`.
    pub fn recipe(&self) -> MarkRecipe {
        self.recipe.clone()
    }

    /// Inspect the target solid body reference key; example: `mark.body()`.
    pub fn body(&self) -> &[u8] {
        &self.recipe.body
    }

    /// Inspect the producer feature of the target body; example: `mark.body_source()`.
    pub fn body_source(&self) -> FeatureId {
        self.body_source
    }

    /// Inspect the authored geometry sets; example: `mark.geometry_sets()`.
    pub fn geometry_sets(&self) -> &[MarkGeometrySet] {
        &self.recipe.geometry_sets
    }

    /// Inspect the authored projection method; example: `mark.method()`.
    pub fn method(&self) -> MarkMethod {
        self.recipe.method
    }

    /// Inspect the authored target faces; example: `mark.faces()`.
    pub fn faces(&self) -> &[Vec<u8>] {
        &self.recipe.faces
    }

    /// Inspect whether Automatic Face Chain is enabled; example: `mark.automatic_face_chain()`.
    pub fn automatic_face_chain(&self) -> bool {
        self.recipe.automatic_face_chain
    }

    /// Inspect the projection direction; example: `mark.direction()`.
    pub fn direction(&self) -> MarkDirection {
        self.recipe.direction
    }

    /// Collect feature and sketch read dependencies; example: `mark.reads(snapshot)?`.
    pub fn reads(&self, snapshot: &DocumentSnapshot) -> Result<FeatureReadFootprint, String> {
        let mut keys = BTreeSet::new();
        let mut unattributed = false;
        keys.insert(FeatureReadKey::Feature(self.body_source));
        for &source in &self.face_sources {
            keys.insert(FeatureReadKey::Feature(source));
        }
        for set in &self.recipe.geometry_sets {
            let sketch = snapshot
                .sketches
                .get(set.sketch)
                .map_err(|cause| format!("mark sketch {:?}: {cause}", set.sketch))?;
            keys.extend(sketch.read_footprint().keys().iter().copied());
            unattributed |= sketch.read_footprint().has_unattributed_reads();
        }
        Ok(FeatureReadFootprint::new(keys, unattributed))
    }

    /// Resolve all geometry sets to 3D world polylines for replay.
    /// Example: `mark.resolve_for_replay(snapshot, datums)?`.
    pub fn resolve_for_replay(
        &self,
        snapshot: &DocumentSnapshot,
        datums: &WorkGeometryEvaluation,
    ) -> Result<ResolvedMark, String> {
        let mut resolved_sets = Vec::with_capacity(self.recipe.geometry_sets.len());
        for set in &self.recipe.geometry_sets {
            let sketch = snapshot
                .require_converged_sketch_with_datums(set.sketch, datums)
                .map_err(|cause| format!("mark sketch {}: {cause}", set.sketch.0))?;
            let polylines_3d =
                sample_sketch_curves_3d(sketch.sketch.geometry(), sketch.plane, &set.curves)?;
            resolved_sets.push(ResolvedMarkGeometrySet {
                mark_type: set.mark_type,
                plane: sketch.plane,
                polylines_3d,
            });
        }
        Ok(ResolvedMark {
            body: self.recipe.body.clone(),
            geometry_sets: resolved_sets,
            method: self.recipe.method,
            faces: self.recipe.faces.clone(),
            automatic_face_chain: self.recipe.automatic_face_chain,
            direction: self.recipe.direction,
        })
    }
}

fn sample_sketch_curves_3d(
    geometry: &Sketch2D,
    plane: SketchPlane,
    curves: &[SketchEntityId],
) -> Result<Vec<Vec<[f64; 3]>>, String> {
    let mut polylines = Vec::with_capacity(curves.len());
    for &curve_id in curves {
        let pts_2d = sample_one_curve_2d(geometry, curve_id)?;
        let pts_3d = pts_2d
            .into_iter()
            .map(|pt| plane_to_world(plane, pt))
            .collect();
        polylines.push(pts_3d);
    }
    Ok(polylines)
}

fn sample_one_curve_2d(
    geometry: &Sketch2D,
    curve_id: SketchEntityId,
) -> Result<Vec<[f64; 2]>, String> {
    if let Ok(line) = geometry.line(curve_id) {
        let a = geometry
            .point(line.endpoints[0])
            .map_err(|cause| format!("mark line {curve_id:?}: {cause}"))?;
        let b = geometry
            .point(line.endpoints[1])
            .map_err(|cause| format!("mark line {curve_id:?}: {cause}"))?;
        return Ok(vec![a.position, b.position]);
    }
    if let Ok(circle) = geometry.circle(curve_id) {
        let center = geometry
            .point(circle.center)
            .map_err(|cause| format!("mark circle {curve_id:?}: {cause}"))?
            .position;
        let steps = 32usize;
        let mut pts = Vec::with_capacity(steps + 1);
        for i in 0..=steps {
            let theta = (i as f64 / steps as f64) * std::f64::consts::TAU;
            pts.push([
                center[0] + circle.radius * theta.cos(),
                center[1] + circle.radius * theta.sin(),
            ]);
        }
        return Ok(pts);
    }
    if let Ok(arc) = geometry.arc(curve_id) {
        let center = geometry
            .point(arc.center)
            .map_err(|cause| format!("mark arc {curve_id:?}: {cause}"))?
            .position;
        let start = geometry
            .point(arc.start)
            .map_err(|cause| format!("mark arc {curve_id:?}: {cause}"))?
            .position;
        let end = geometry
            .point(arc.end)
            .map_err(|cause| format!("mark arc {curve_id:?}: {cause}"))?
            .position;
        let radius = ((start[0] - center[0]).hypot(start[1] - center[1])).max(1e-9);
        let a0 = (start[1] - center[1]).atan2(start[0] - center[0]);
        let mut a1 = (end[1] - center[1]).atan2(end[0] - center[0]);
        if arc.counter_clockwise {
            if a1 <= a0 {
                a1 += std::f64::consts::TAU;
            }
        } else if a1 >= a0 {
            a1 -= std::f64::consts::TAU;
        }
        let steps = 20usize;
        let mut pts = Vec::with_capacity(steps + 1);
        for i in 0..=steps {
            let t = i as f64 / steps as f64;
            let a = a0 + (a1 - a0) * t;
            pts.push([center[0] + radius * a.cos(), center[1] + radius * a.sin()]);
        }
        return Ok(pts);
    }
    Err(format!(
        "mark curve {curve_id:?}: expected a line, circle, or arc in the selected sketch"
    ))
}

fn plane_to_world(plane: SketchPlane, pt: [f64; 2]) -> [f64; 3] {
    let recipe = plane.recipe();
    [
        recipe.origin[0] + pt[0] * recipe.x_axis[0] + pt[1] * recipe.y_axis[0],
        recipe.origin[1] + pt[0] * recipe.x_axis[1] + pt[1] * recipe.y_axis[1],
        recipe.origin[2] + pt[0] * recipe.x_axis[2] + pt[1] * recipe.y_axis[2],
    ]
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::identity::{DocumentGuid, SemanticReferenceName};

    fn sample_keys() -> (Vec<u8>, Vec<u8>) {
        let doc: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
        let body = SemanticReferenceName::new(doc, 1, 4, "prism", "body", &[1])
            .unwrap()
            .encode()
            .unwrap();
        let face = SemanticReferenceName::new(doc, 1, 1, "prism", "top", &[1])
            .unwrap()
            .encode()
            .unwrap();
        (body, face)
    }

    #[test]
    fn validates_project_and_wrap_recipes() {
        let (body, face) = sample_keys();
        let project = MarkRecipe {
            version: 1,
            body: body.clone(),
            geometry_sets: vec![MarkGeometrySet {
                sketch: DocumentSketchId(2),
                curves: vec![SketchEntityId(37)],
                mark_type: MarkType::MarkThrough,
            }],
            method: MarkMethod::ProjectToClosestFace,
            faces: Vec::new(),
            automatic_face_chain: false,
            direction: MarkDirection::Positive,
        };
        let def = MarkDefinition::from_recipe(project).unwrap();
        assert_eq!(def.method(), MarkMethod::ProjectToClosestFace);
        assert_eq!(def.geometry_sets().len(), 1);
        assert_eq!(def.geometry_sets()[0].mark_type, MarkType::MarkThrough);

        let wrap_missing_faces = MarkRecipe {
            version: 1,
            body: body.clone(),
            geometry_sets: vec![MarkGeometrySet {
                sketch: DocumentSketchId(2),
                curves: vec![SketchEntityId(37)],
                mark_type: MarkType::MarkSurface,
            }],
            method: MarkMethod::WrapToFace,
            faces: Vec::new(),
            automatic_face_chain: true,
            direction: MarkDirection::Positive,
        };
        assert!(MarkDefinition::from_recipe(wrap_missing_faces).is_err());

        let wrap_ok = MarkRecipe {
            version: 1,
            body,
            geometry_sets: vec![MarkGeometrySet {
                sketch: DocumentSketchId(2),
                curves: vec![SketchEntityId(37)],
                mark_type: MarkType::MarkSurface,
            }],
            method: MarkMethod::WrapToFace,
            faces: vec![face],
            automatic_face_chain: true,
            direction: MarkDirection::Symmetric,
        };
        let wrap_def = MarkDefinition::from_recipe(wrap_ok).unwrap();
        assert!(wrap_def.automatic_face_chain());
        assert_eq!(wrap_def.faces().len(), 1);
    }

    #[test]
    fn rejects_empty_geometry_sets() {
        let (body, _) = sample_keys();
        let recipe = MarkRecipe {
            version: 1,
            body,
            geometry_sets: Vec::new(),
            method: MarkMethod::ProjectToClosestFace,
            faces: Vec::new(),
            automatic_face_chain: false,
            direction: MarkDirection::Positive,
        };
        assert!(MarkDefinition::from_recipe(recipe).is_err());
    }
}
