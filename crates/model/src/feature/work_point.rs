// SPDX-License-Identifier: GPL-2.0-only
//! Analytic work-point constructions: vertex/loop references, grounded points, and intersections (#3918).
use crate::{
    document_sketch::SketchFrameTopologyOrigin,
    feature::{
        FeatureReadFootprint, FeatureReadKey,
        work_axis::{WorkAxisLine, WorkAxisReference},
        work_geometry::{DatumLength, WorkGeometryError, plane_key, validate_position},
        work_plane::{WorkPlaneReference, WorkPlaneResolver},
    },
    parameter::ParameterCollection,
    sketch::SketchPlane,
};

const PARALLEL_EPSILON: f64 = 1.0e-9;
const SKEW_LINE_TOLERANCE: f64 = 1.0e-5;

/// Authored analytic work-point construction source.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub enum WorkPointConstruction {
    OnVertex(SketchFrameTopologyOrigin),
    LoopCenter(SketchFrameTopologyOrigin),
    Grounded([DatumLength; 3]),
    TwoLines {
        first: WorkAxisReference,
        second: WorkAxisReference,
    },
    PlaneAndLine {
        plane: WorkPlaneReference,
        line: WorkAxisReference,
    },
    ThreePlanes([WorkPlaneReference; 3]),
}

/// Versioned serialized recipe for an analytic work-point construction.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct WorkPointRecipe {
    pub version: u32,
    pub construction: WorkPointConstruction,
}

/// Validated authored work-point definition.
#[derive(Clone, Debug, PartialEq)]
pub struct WorkPointDefinition {
    construction: WorkPointConstruction,
}

impl WorkPointDefinition {
    /// Construct and validate a work-point definition; example: `WorkPointDefinition::new(construction)?`.
    pub fn new(construction: WorkPointConstruction) -> Result<Self, WorkGeometryError> {
        if let WorkPointConstruction::Grounded(coordinates) = construction {
            validate_position(coordinates)?;
        }
        Ok(Self { construction })
    }

    /// Inspect the authored construction variant; example: `definition.construction()`.
    pub fn construction(&self) -> &WorkPointConstruction {
        &self.construction
    }

    /// Capture versioned authored source; example: `definition.recipe()`.
    pub fn recipe(&self) -> WorkPointRecipe {
        WorkPointRecipe {
            version: 1,
            construction: self.construction.clone(),
        }
    }

    /// Restore and validate a serialized recipe; example: `WorkPointDefinition::from_recipe(recipe)?`.
    pub fn from_recipe(recipe: WorkPointRecipe) -> Result<Self, WorkGeometryError> {
        if recipe.version != 1 {
            return Err(WorkGeometryError::RecipeVersion(recipe.version));
        }
        Self::new(recipe.construction)
    }

    /// Compare exact authored recipe bits without derived geometry; example: `a.same_recipe(&b)`.
    pub fn same_recipe(&self, other: &Self) -> bool {
        match (&self.construction, &other.construction) {
            (WorkPointConstruction::Grounded(a), WorkPointConstruction::Grounded(b)) => a
                .iter()
                .zip(*b)
                .all(|(left, right)| left.same_recipe(right)),
            _ => self.construction == other.construction,
        }
    }

    /// True when this work point depends on named solid topology or planar face references.
    pub fn requires_named_part_replay(&self) -> bool {
        match &self.construction {
            WorkPointConstruction::OnVertex(_) | WorkPointConstruction::LoopCenter(_) => true,
            WorkPointConstruction::PlaneAndLine { plane, .. } => is_face_plane(plane),
            WorkPointConstruction::ThreePlanes(planes) => planes.iter().any(is_face_plane),
            WorkPointConstruction::Grounded(_) | WorkPointConstruction::TwoLines { .. } => false,
        }
    }

    /// Attribute every authored dependency for replay ordering; example: `definition.reads()`.
    pub fn reads(&self) -> FeatureReadFootprint {
        let mut keys = Vec::new();
        match &self.construction {
            WorkPointConstruction::OnVertex(origin) | WorkPointConstruction::LoopCenter(origin) => {
                keys.push(FeatureReadKey::Feature(origin.source()));
            }
            WorkPointConstruction::Grounded(coordinates) => {
                for length in coordinates {
                    if let Some(parameter) = length.parameter() {
                        keys.push(FeatureReadKey::Parameter(parameter));
                    }
                }
            }
            WorkPointConstruction::TwoLines { first, second } => {
                push_axis_key(&mut keys, *first);
                push_axis_key(&mut keys, *second);
            }
            WorkPointConstruction::PlaneAndLine { plane, line } => {
                if let Some(key) = plane_key(plane) {
                    keys.push(key);
                }
                push_axis_key(&mut keys, *line);
            }
            WorkPointConstruction::ThreePlanes(planes) => {
                for plane in planes {
                    if let Some(key) = plane_key(plane) {
                        keys.push(key);
                    }
                }
            }
        }
        FeatureReadFootprint::new(keys, false)
    }

    /// Evaluate exact 3D model coordinates without consulting tessellation; example: `definition.evaluate(&params, &resolver)?`.
    pub fn evaluate(
        &self,
        parameters: &ParameterCollection<u64>,
        resolver: &impl WorkPlaneResolver,
    ) -> Result<[f64; 3], WorkGeometryError> {
        match &self.construction {
            WorkPointConstruction::OnVertex(origin) | WorkPointConstruction::LoopCenter(origin) => {
                resolver.topology_origin(origin)
            }
            WorkPointConstruction::Grounded(coordinates) => {
                resolve_grounded(*coordinates, parameters)
            }
            WorkPointConstruction::TwoLines { first, second } => {
                intersect_two_lines(first.line(resolver)?, second.line(resolver)?)
            }
            WorkPointConstruction::PlaneAndLine { plane, line } => {
                intersect_plane_and_line(plane.clone().frame(resolver)?, line.line(resolver)?)
            }
            WorkPointConstruction::ThreePlanes(planes) => intersect_three_planes([
                planes[0].clone().frame(resolver)?,
                planes[1].clone().frame(resolver)?,
                planes[2].clone().frame(resolver)?,
            ]),
        }
    }
}

fn is_face_plane(reference: &WorkPlaneReference) -> bool {
    matches!(
        reference,
        WorkPlaneReference::Face(_) | WorkPlaneReference::GoPrismFace(_)
    )
}

fn push_axis_key(keys: &mut Vec<FeatureReadKey>, reference: WorkAxisReference) {
    if let WorkAxisReference::Feature(id) = reference {
        keys.push(FeatureReadKey::Feature(id));
    }
}

fn resolve_grounded(
    coordinates: [DatumLength; 3],
    parameters: &ParameterCollection<u64>,
) -> Result<[f64; 3], WorkGeometryError> {
    let mut result = [0.0; 3];
    for (component, length) in coordinates.into_iter().enumerate() {
        result[component] = length.resolve(parameters).map_err(|cause| {
            WorkGeometryError::Coordinate(crate::feature::work_geometry::WorkPointCoordinateError {
                component,
                cause,
            })
        })?;
    }
    Ok(result)
}

/// Exact 3D intersection of two coplanar non-parallel lines.
pub fn intersect_two_lines(
    first: WorkAxisLine,
    second: WorkAxisLine,
) -> Result<[f64; 3], WorkGeometryError> {
    let p1 = first.origin();
    let d1 = first.direction();
    let p2 = second.origin();
    let d2 = second.direction();
    let n = cross(d1, d2);
    let denom = dot(n, n);
    if !denom.is_finite() || denom <= PARALLEL_EPSILON * PARALLEL_EPSILON {
        return Err(WorkGeometryError::BodyReference(
            "two-line work point: lines are parallel or collinear".into(),
        ));
    }
    let delta = sub(p2, p1);
    let s = dot(cross(delta, d2), n) / denom;
    let t = dot(cross(delta, d1), n) / denom;
    let q1 = add_scaled(p1, d1, s);
    let q2 = add_scaled(p2, d2, t);
    let gap = norm(sub(q1, q2));
    if !gap.is_finite() || gap > SKEW_LINE_TOLERANCE {
        return Err(WorkGeometryError::BodyReference(format!(
            "two-line work point: lines are skew (separation {gap:.6})"
        )));
    }
    let mid = std::array::from_fn(|axis| 0.5 * (q1[axis] + q2[axis]));
    require_finite_point(mid, "two-line work point")
}

/// Exact 3D intersection of a plane and a non-parallel line.
pub fn intersect_plane_and_line(
    plane: SketchPlane,
    line: WorkAxisLine,
) -> Result<[f64; 3], WorkGeometryError> {
    let normal = plane.normal();
    let plane_origin = plane.recipe().origin;
    let line_origin = line.origin();
    let direction = line.direction();
    let denom = dot(normal, direction);
    if !denom.is_finite() || denom.abs() <= PARALLEL_EPSILON {
        return Err(WorkGeometryError::BodyReference(
            "plane-and-line work point: line is parallel to plane".into(),
        ));
    }
    let t = dot(normal, sub(plane_origin, line_origin)) / denom;
    let point = add_scaled(line_origin, direction, t);
    require_finite_point(point, "plane-and-line work point")
}

/// Exact 3D intersection of three mutually non-degenerate planes.
pub fn intersect_three_planes(planes: [SketchPlane; 3]) -> Result<[f64; 3], WorkGeometryError> {
    let n0 = planes[0].normal();
    let n1 = planes[1].normal();
    let n2 = planes[2].normal();
    let c12 = cross(n1, n2);
    let det = dot(n0, c12);
    if !det.is_finite() || det.abs() <= PARALLEL_EPSILON {
        return Err(WorkGeometryError::BodyReference(
            "three-plane work point: planes do not intersect at a unique point".into(),
        ));
    }
    let d0 = dot(n0, planes[0].recipe().origin);
    let d1 = dot(n1, planes[1].recipe().origin);
    let d2 = dot(n2, planes[2].recipe().origin);
    let c20 = cross(n2, n0);
    let c01 = cross(n0, n1);
    let point =
        std::array::from_fn(|axis| (d0 * c12[axis] + d1 * c20[axis] + d2 * c01[axis]) / det);
    require_finite_point(point, "three-plane work point")
}

fn require_finite_point(point: [f64; 3], label: &str) -> Result<[f64; 3], WorkGeometryError> {
    if !point.iter().all(|value| value.is_finite()) {
        return Err(WorkGeometryError::BodyReference(format!(
            "{label}: expected finite intersection coordinates"
        )));
    }
    Ok(point)
}

fn dot(a: [f64; 3], b: [f64; 3]) -> f64 {
    a[0] * b[0] + a[1] * b[1] + a[2] * b[2]
}

fn cross(a: [f64; 3], b: [f64; 3]) -> [f64; 3] {
    [
        a[1] * b[2] - a[2] * b[1],
        a[2] * b[0] - a[0] * b[2],
        a[0] * b[1] - a[1] * b[0],
    ]
}

fn sub(a: [f64; 3], b: [f64; 3]) -> [f64; 3] {
    [a[0] - b[0], a[1] - b[1], a[2] - b[2]]
}

fn add_scaled(origin: [f64; 3], dir: [f64; 3], scale: f64) -> [f64; 3] {
    [
        origin[0] + dir[0] * scale,
        origin[1] + dir[1] * scale,
        origin[2] + dir[2] * scale,
    ]
}

fn norm(v: [f64; 3]) -> f64 {
    dot(v, v).sqrt()
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::feature::{
        AuthoredFeature, FeatureId, FeatureSequence, work_geometry::WorkGeometryDefinition,
        work_plane::WorkPlaneOrigin,
    };
    use crate::units::UnitsOfMeasure;

    fn close(a: [f64; 3], b: [f64; 3]) -> bool {
        (0..3).all(|i| (a[i] - b[i]).abs() < 1.0e-9)
    }

    #[test]
    fn two_lines_plane_line_and_three_planes_intersect_exactly() {
        let params = ParameterCollection::new(UnitsOfMeasure::default());
        let mut seq = FeatureSequence::<WorkGeometryDefinition>::default();
        seq.append(AuthoredFeature::datum(
            FeatureId(1),
            "Work Point1",
            WorkGeometryDefinition::WorkPoint(Box::new(
                WorkPointDefinition::new(WorkPointConstruction::TwoLines {
                    first: WorkAxisReference::OriginX,
                    second: WorkAxisReference::OriginY,
                })
                .unwrap(),
            )),
        ))
        .unwrap();
        seq.append(AuthoredFeature::datum(
            FeatureId(2),
            "Work Point2",
            WorkGeometryDefinition::WorkPoint(Box::new(
                WorkPointDefinition::new(WorkPointConstruction::PlaneAndLine {
                    plane: WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
                    line: WorkAxisReference::OriginZ,
                })
                .unwrap(),
            )),
        ))
        .unwrap();
        seq.append(AuthoredFeature::datum(
            FeatureId(3),
            "Work Point3",
            WorkGeometryDefinition::WorkPoint(Box::new(
                WorkPointDefinition::new(WorkPointConstruction::ThreePlanes([
                    WorkPlaneReference::Origin(WorkPlaneOrigin::YZ),
                    WorkPlaneReference::Origin(WorkPlaneOrigin::XZ),
                    WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
                ]))
                .unwrap(),
            )),
        ))
        .unwrap();
        let evaluated = seq.evaluate_work_geometry(&params).unwrap();
        assert!(close(evaluated.point(FeatureId(1)).unwrap(), [0.0; 3]));
        assert!(close(evaluated.point(FeatureId(2)).unwrap(), [0.0; 3]));
        assert!(close(evaluated.point(FeatureId(3)).unwrap(), [0.0; 3]));

        let recipe = seq.work_geometry_recipe().unwrap();
        assert_eq!(recipe.version, 66);
        let restored = FeatureSequence::from_work_geometry_recipe(&recipe).unwrap();
        assert!(seq.same_work_geometry_recipe(&restored));
    }

    #[test]
    fn offset_three_planes_and_degenerate_intersections() {
        let p_xy =
            SketchPlane::from_origin_normal([0.0, 0.0, 3.0], [0.0, 0.0, 1.0], Default::default())
                .unwrap();
        let p_xz =
            SketchPlane::from_origin_normal([0.0, 2.0, 0.0], [0.0, 1.0, 0.0], Default::default())
                .unwrap();
        let p_yz =
            SketchPlane::from_origin_normal([1.0, 0.0, 0.0], [1.0, 0.0, 0.0], Default::default())
                .unwrap();
        assert!(close(
            intersect_three_planes([p_yz, p_xz, p_xy]).unwrap(),
            [1.0, 2.0, 3.0]
        ));
        assert!(intersect_three_planes([p_xy, p_xy, p_yz]).is_err());

        let l1 = WorkAxisLine::new([0.0, 0.0, 0.0], [1.0, 0.0, 0.0]).unwrap();
        let l2 = WorkAxisLine::new([0.0, 0.0, 1.0], [0.0, 1.0, 0.0]).unwrap();
        assert!(intersect_two_lines(l1, l2).is_err());
        assert!(intersect_two_lines(l1, l1).is_err());
        assert!(intersect_plane_and_line(p_xy, l1).is_err());
        let l3 = WorkAxisLine::new([4.0, 5.0, 0.0], [0.0, 0.0, 1.0]).unwrap();
        assert!(close(
            intersect_plane_and_line(p_xy, l3).unwrap(),
            [4.0, 5.0, 3.0]
        ));
    }

    #[test]
    fn grounded_point_roundtrips_and_rejects_non_finite() {
        assert!(
            WorkPointDefinition::new(WorkPointConstruction::Grounded([
                DatumLength::Constant(f64::NAN),
                DatumLength::Constant(0.0),
                DatumLength::Constant(0.0),
            ]))
            .is_err()
        );
        let def = WorkPointDefinition::new(WorkPointConstruction::Grounded([
            DatumLength::Constant(1.5),
            DatumLength::Constant(-2.0),
            DatumLength::Constant(3.25),
        ]))
        .unwrap();
        let restored = WorkPointDefinition::from_recipe(def.recipe()).unwrap();
        assert!(def.same_recipe(&restored));
    }
}
