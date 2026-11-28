// SPDX-License-Identifier: GPL-2.0-only
//! Analytic evaluation for topology-aware extended Work Plane definitions (#3918).

use super::{WorkPlaneExtendedDefinition, WorkPlaneSurfaceReference, WorkPlaneTopologyRef};
use crate::{
    feature::{
        FeatureId,
        work_axis::WorkAxisLine,
        work_plane::{WorkPlaneError, WorkPlaneFaceReference, WorkPlaneResolver},
    },
    parameter::ParameterCollection,
    sketch::{SketchError, SketchPlane, SketchPlaneOptions, normalize_direction, point_difference},
};

/// Extended resolver providing named B-rep vertices, straight edges, torus, and tangent surfaces.
pub trait WorkPlaneExtendedResolver: WorkPlaneResolver {
    fn topology_vertex(
        &self,
        reference: &WorkPlaneTopologyRef,
    ) -> Result<[f64; 3], WorkPlaneError> {
        Err(WorkPlaneError::BodyReference(format!(
            "vertex key {:?}: expected a current named-vertex resolver",
            reference.key()
        )))
    }
    fn topology_edge(
        &self,
        reference: &WorkPlaneTopologyRef,
    ) -> Result<WorkAxisLine, WorkPlaneError> {
        Err(WorkPlaneError::BodyReference(format!(
            "edge key {:?}: expected a current named-edge resolver",
            reference.key()
        )))
    }
    fn torus_midplane(
        &self,
        surface: &WorkPlaneSurfaceReference,
    ) -> Result<SketchPlane, WorkPlaneError> {
        Err(WorkPlaneError::BodyReference(format!(
            "torus face key {:?}: expected a current torus midplane resolver",
            surface.face.key()
        )))
    }
    fn tangent_through_edge(
        &self,
        surface: &WorkPlaneSurfaceReference,
        line: WorkAxisLine,
    ) -> Result<SketchPlane, WorkPlaneError> {
        Err(WorkPlaneError::BodyReference(format!(
            "tangent face key {:?}, line {line:?}: expected a current tangent surface resolver",
            surface.face.key()
        )))
    }
    fn tangent_through_point(
        &self,
        surface: &WorkPlaneSurfaceReference,
        point: [f64; 3],
    ) -> Result<SketchPlane, WorkPlaneError> {
        Err(WorkPlaneError::BodyReference(format!(
            "tangent face key {:?}, point {point:?}: expected a current tangent surface resolver",
            surface.face.key()
        )))
    }
    fn tangent_parallel_to_plane(
        &self,
        surface: &WorkPlaneSurfaceReference,
        plane: SketchPlane,
    ) -> Result<SketchPlane, WorkPlaneError> {
        Err(WorkPlaneError::BodyReference(format!(
            "tangent face key {:?}, plane {plane:?}: expected a current tangent surface resolver",
            surface.face.key()
        )))
    }
}

struct PlainPlaneResolver<'a, R: WorkPlaneResolver>(&'a R);

impl<R: WorkPlaneResolver> WorkPlaneResolver for PlainPlaneResolver<'_, R> {
    fn axis(&self, feature: FeatureId) -> Result<WorkAxisLine, WorkPlaneError> {
        self.0.axis(feature)
    }
    fn plane(&self, feature: FeatureId) -> Result<SketchPlane, WorkPlaneError> {
        self.0.plane(feature)
    }
    fn face(&self, reference: &WorkPlaneFaceReference) -> Result<SketchPlane, WorkPlaneError> {
        self.0.face(reference)
    }
    fn point(&self, feature: FeatureId) -> Result<[f64; 3], WorkPlaneError> {
        self.0.point(feature)
    }
}

impl<R: WorkPlaneResolver> WorkPlaneExtendedResolver for PlainPlaneResolver<'_, R> {}

impl WorkPlaneExtendedDefinition {
    /// Evaluate without body-topology queries (works for pure datum/origin references).
    pub fn evaluate_plain(
        &self,
        parameters: &ParameterCollection<u64>,
        resolver: &impl WorkPlaneResolver,
    ) -> Result<SketchPlane, WorkPlaneError> {
        self.evaluate(parameters, &PlainPlaneResolver(resolver))
    }

    /// Evaluate exact analytic construction plane; example: `definition.evaluate(&params, &resolver)?`.
    pub fn evaluate(
        &self,
        parameters: &ParameterCollection<u64>,
        resolver: &impl WorkPlaneExtendedResolver,
    ) -> Result<SketchPlane, WorkPlaneError> {
        match self {
            Self::MidplaneTwoPlanes { first, second } => midplane_between(
                first.clone().frame(resolver)?,
                second.clone().frame(resolver)?,
            ),
            Self::ThreePoints { points } => SketchPlane::through_points(
                points[0].resolve(resolver)?,
                points[1].resolve(resolver)?,
                points[2].resolve(resolver)?,
                SketchPlaneOptions::default(),
            )
            .map_err(WorkPlaneError::Frame),
            Self::ParallelThroughPoint { base, point } => {
                let plane = base.clone().frame(resolver)?;
                let mut recipe = plane.recipe();
                recipe.origin = point.resolve(resolver)?;
                SketchPlane::from_recipe(recipe, plane.options()).map_err(WorkPlaneError::Frame)
            }
            Self::NormalToAxisThroughPoint { axis, point } => {
                let line = axis.resolve(resolver)?;
                let origin = point.resolve(resolver)?;
                SketchPlane::from_origin_normal(
                    origin,
                    line.direction(),
                    SketchPlaneOptions::default(),
                )
                .map_err(WorkPlaneError::Frame)
            }
            Self::AngleToPlaneAroundEdge { base, edge, angle } => {
                let plane = base.clone().frame(resolver)?;
                let line = edge.resolve(resolver)?;
                let radians = angle.resolve(parameters)?;
                plane
                    .at_angle_about_axis(line.origin(), line.direction(), radians)
                    .map_err(WorkPlaneError::Frame)
            }
            Self::TwoCoplanarEdges { first, second } => {
                two_coplanar_edges(first.resolve(resolver)?, second.resolve(resolver)?)
            }
            Self::MidplaneOfTorus { surface } => resolver.torus_midplane(surface),
            Self::TangentThroughEdge { surface, edge } => {
                let line = edge.resolve(resolver)?;
                resolver.tangent_through_edge(surface, line)
            }
            Self::TangentThroughPoint { surface, point } => {
                let position = point.resolve(resolver)?;
                resolver.tangent_through_point(surface, position)
            }
            Self::TangentParallelToPlane { surface, plane } => {
                let reference = plane.clone().frame(resolver)?;
                resolver.tangent_parallel_to_plane(surface, reference)
            }
        }
    }
}

fn midplane_between(
    first: SketchPlane,
    second: SketchPlane,
) -> Result<SketchPlane, WorkPlaneError> {
    let n1 = first.normal();
    let n2 = second.normal();
    let dot = dot3(n1, n2);
    let o1 = first.recipe().origin;
    let o2 = second.recipe().origin;
    if dot.abs() >= 1.0 - 1e-7 {
        let separation = dot3(point_difference(o1, o2), n1);
        if separation.abs() <= 1e-9 {
            return Err(WorkPlaneError::Frame(SketchError::Definition(
                "midplane references: expected two distinct parallel planes".into(),
            )));
        }
        let mut recipe = first.recipe();
        recipe.origin = std::array::from_fn(|i| 0.5 * (o1[i] + o2[i]));
        return SketchPlane::from_recipe(recipe, first.options()).map_err(WorkPlaneError::Frame);
    }
    let line_dir = normalize_direction(cross3(n1, n2)).map_err(WorkPlaneError::Frame)?;
    let d1 = dot3(n1, o1);
    let d2 = dot3(n2, o2);
    let denom = 1.0 - dot * dot;
    let c1 = (d1 - d2 * dot) / denom;
    let c2 = (d2 - d1 * dot) / denom;
    let origin = std::array::from_fn(|i| c1 * n1[i] + c2 * n2[i]);
    let aligned_n2 = if dot >= 0.0 {
        n2
    } else {
        [-n2[0], -n2[1], -n2[2]]
    };
    let bisector_normal = normalize_direction(std::array::from_fn(|i| n1[i] + aligned_n2[i]))
        .map_err(WorkPlaneError::Frame)?;
    let y_axis =
        normalize_direction(cross3(bisector_normal, line_dir)).map_err(WorkPlaneError::Frame)?;
    SketchPlane::new(origin, line_dir, y_axis, SketchPlaneOptions::default())
        .map_err(WorkPlaneError::Frame)
}

fn two_coplanar_edges(
    first: WorkAxisLine,
    second: WorkAxisLine,
) -> Result<SketchPlane, WorkPlaneError> {
    let o1 = first.origin();
    let o2 = second.origin();
    let d1 = first.direction();
    let d2 = second.direction();
    let cross = cross3(d1, d2);
    let cross_len = (cross[0] * cross[0] + cross[1] * cross[1] + cross[2] * cross[2]).sqrt();
    let delta = point_difference(o1, o2);
    if cross_len > 1e-7 {
        let normal = cross.map(|v| v / cross_len);
        let skew = dot3(delta, normal).abs();
        if skew > 1e-5 {
            return Err(WorkPlaneError::Frame(SketchError::Definition(format!(
                "two coplanar edges skew distance {skew}: expected coplanar straight edges"
            ))));
        }
        return SketchPlane::from_axis_directions(o1, d1, d2, SketchPlaneOptions::default())
            .map_err(WorkPlaneError::Frame);
    }
    let proj = dot3(delta, d1);
    let perp = std::array::from_fn(|i| delta[i] - proj * d1[i]);
    let perp_dir = normalize_direction(perp).map_err(|_| {
        WorkPlaneError::Frame(SketchError::Definition(
            "two coplanar edges: expected non-collinear distinct parallel edges".into(),
        ))
    })?;
    SketchPlane::new(o1, d1, perp_dir, SketchPlaneOptions::default()).map_err(WorkPlaneError::Frame)
}

fn dot3(a: [f64; 3], b: [f64; 3]) -> f64 {
    a[0] * b[0] + a[1] * b[1] + a[2] * b[2]
}

fn cross3(a: [f64; 3], b: [f64; 3]) -> [f64; 3] {
    [
        a[1] * b[2] - a[2] * b[1],
        a[2] * b[0] - a[0] * b[2],
        a[0] * b[1] - a[1] * b[0],
    ]
}
