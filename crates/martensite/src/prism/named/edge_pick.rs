// SPDX-License-Identifier: GPL-2.0-only
//! Exact edge-to-segment queries for viewport selection (#4045).
use super::*;
use martensite_tkmath::gp::Point3;
use martensite_tktopalgo::{
    brep_builder_api::BRepBuilderAPIMakeEdge, brep_extrema::BRepExtremaDistShapeShape,
};
use oblikovati_model::identity::ReferenceEnvelope;

/// One named edge's minimum distance to a finite view segment.
#[derive(Clone, Debug, PartialEq)]
pub struct NamedEdgeHit {
    pub key: Vec<u8>,
    pub distance: f64,
    pub reach: f64,
}

impl NamedSolid {
    /// Sample one named edge for display; example: `body.edge_display_segments(&history, document, key, 0.01)?`.
    pub fn edge_display_segments(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
        deflection: f64,
    ) -> Result<Vec<[[f64; 3]; 2]>, String> {
        use martensite_tkbrep::{brep_adaptor::BRepAdaptorCurve, topods};
        use martensite_tkg3d::adaptor3d::Adaptor3dCurve;
        let shape = self.native_edge(history, document, key)?;
        let edge = topods::edge(shape).map_err(|cause| cause.to_string())?;
        let curve = BRepAdaptorCurve::with_edge(&edge).map_err(|cause| cause.to_string())?;
        let range = [
            curve.first_parameter().map_err(|cause| cause.to_string())?,
            curve.last_parameter().map_err(|cause| cause.to_string())?,
        ];
        let parameters = super::discretize::curve_parameters(shape, range, deflection)?;
        let points = parameters
            .into_iter()
            .map(|parameter| {
                curve
                    .value(parameter)
                    .map(|point| [point.x(), point.y(), point.z()])
                    .map_err(|cause| cause.to_string())
            })
            .collect::<Result<Vec<_>, _>>()?;
        if points.iter().flatten().any(|value| !value.is_finite()) {
            return Err(format!(
                "edge {key:?} display points {points:?}: expected finite coordinates"
            ));
        }
        Ok(points.windows(2).map(|pair| [pair[0], pair[1]]).collect())
    }
    /// Query named edges on a forward view ray over the analytic body bounds.
    /// Example: `body.edge_hits_on_ray(&history, document, [point, direction], tolerance)?`.
    pub fn edge_hits_on_ray(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        [point, direction]: [[f64; 3]; 2],
        tolerance: f64,
    ) -> Result<Vec<NamedEdgeHit>, String> {
        if !tolerance.is_finite() || tolerance <= 0. {
            return Err(format!(
                "edge pick tolerance {tolerance}: expected a finite positive length"
            ));
        }
        let Some(end) = ray_end(
            self.native_shape(history, document)?,
            point,
            direction,
            tolerance,
        )?
        else {
            return Ok(Vec::new());
        };
        self.edge_hits_on_segment(history, document, [point, end])
    }
    /// Query analytic edge distances in deterministic depth order.
    /// Example: `body.edge_hits_on_segment(&history, document, [near, far])?`.
    pub fn edge_hits_on_segment(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        segment: [[f64; 3]; 2],
    ) -> Result<Vec<NamedEdgeHit>, String> {
        let (ray, direction) = segment_edge(segment)?;
        let mut hits = Vec::new();
        for key in self.subentity_reference_keys(history, document)? {
            if ReferenceEnvelope::decode(&key)
                .map_err(|cause| cause.to_string())?
                .kind
                != 2
            {
                continue;
            }
            let edge = self.native_edge(history, document, &key)?;
            hits.push(edge_distance(edge, &ray, &key, segment[0], direction)?);
        }
        hits.sort_by(|a, b| {
            a.reach
                .total_cmp(&b.reach)
                .then(a.distance.total_cmp(&b.distance))
                .then(a.key.cmp(&b.key))
        });
        Ok(hits)
    }

    fn native_edge<'a>(
        &'a self,
        history: &'a DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<&'a Shape, String> {
        match self {
            Self::Prism(body) => body
                .subentity_bindings
                .as_ref()
                .ok_or("edge selection: expected certified subentities")?
                .bind(history, document, key, 2)
                .map(|entry| &entry.shape),
            Self::Fillet(body) => body.native_edge(history, document, key),
        }
    }
}

fn ray_end(
    body: &Shape,
    point: [f64; 3],
    direction: [f64; 3],
    tolerance: f64,
) -> Result<Option<[f64; 3]>, String> {
    use martensite_tkmath::bnd::BoundingBox3;
    use martensite_tktopalgo::brep_bnd_lib::BRepBndLib;
    let length = direction[0].hypot(direction[1]).hypot(direction[2]);
    if !length.is_finite() || length == 0. || point.iter().any(|value| !value.is_finite()) {
        return Err(format!(
            "edge ray {point:?} {direction:?}: expected finite point and nonzero direction"
        ));
    }
    let unit = direction.map(|value| value / length);
    let mut bounds = BoundingBox3::default();
    BRepBndLib::add_optimal(body, &mut bounds, false, false).map_err(|cause| cause.to_string())?;
    if bounds.is_void() {
        return Err("edge ray bounds: expected a nonempty body".into());
    }
    let limits = bounds.limits();
    let minimum = [limits.x_min, limits.y_min, limits.z_min];
    let maximum = [limits.x_max, limits.y_max, limits.z_max];
    let Some(farthest) = expanded_bounds_exit(point, unit, minimum, maximum, tolerance)? else {
        return Ok(None);
    };
    Ok(Some(std::array::from_fn(|axis| {
        point[axis] + unit[axis] * (farthest + tolerance)
    })))
}

fn expanded_bounds_exit(
    point: [f64; 3],
    direction: [f64; 3],
    minimum: [f64; 3],
    maximum: [f64; 3],
    tolerance: f64,
) -> Result<Option<f64>, String> {
    let mut enter: f64 = 0.;
    let mut exit: f64 = f64::INFINITY;
    for axis in 0..3 {
        let low = minimum[axis] - tolerance;
        let high = maximum[axis] + tolerance;
        if !low.is_finite() || !high.is_finite() || low > high {
            return Err(format!(
                "edge bounds {low}..{high}: expected finite ordered bounds"
            ));
        }
        if direction[axis] == 0. {
            if point[axis] < low || point[axis] > high {
                return Ok(None);
            }
            continue;
        }
        let first = (low - point[axis]) / direction[axis];
        let second = (high - point[axis]) / direction[axis];
        enter = enter.max(first.min(second));
        exit = exit.min(first.max(second));
        if enter > exit {
            return Ok(None);
        }
    }
    if !exit.is_finite() {
        return Err(format!("edge ray exit {exit}: expected finite body bounds"));
    }
    Ok(Some(exit))
}

fn segment_edge([start, end]: [[f64; 3]; 2]) -> Result<(Shape, [f64; 3]), String> {
    let delta = std::array::from_fn::<_, 3, _>(|index| end[index] - start[index]);
    let length = delta[0].hypot(delta[1]).hypot(delta[2]);
    if !length.is_finite()
        || length == 0.
        || start.iter().chain(&end).any(|value| !value.is_finite())
    {
        return Err(format!(
            "view segment {start:?}..{end:?}: expected finite distinct points"
        ));
    }
    let point = |value: [f64; 3]| Point3::new(value[0], value[1], value[2]);
    let ray = BRepBuilderAPIMakeEdge::from_points(point(start), point(end))
        .and_then(|maker| maker.edge())
        .map_err(|cause| cause.to_string())?;
    Ok((ray.into_shape(), delta.map(|value| value / length)))
}

fn edge_distance(
    edge: &Shape,
    ray: &Shape,
    key: &[u8],
    start: [f64; 3],
    direction: [f64; 3],
) -> Result<NamedEdgeHit, String> {
    let extrema = BRepExtremaDistShapeShape::with_shapes(edge, ray, Default::default())
        .map_err(|cause| cause.to_string())?;
    let distance = extrema.value().map_err(|cause| cause.to_string())?;
    if !extrema.is_done() || extrema.nb_solution() <= 0 || !distance.is_finite() || distance < 0. {
        return Err(format!(
            "edge {key:?}, distance {distance}, solutions {}: expected completed finite extrema",
            extrema.nb_solution()
        ));
    }
    let mut reach = f64::INFINITY;
    for index in 1..=extrema.nb_solution() {
        let point = extrema
            .point_on_shape2(index)
            .map_err(|cause| cause.to_string())?;
        let coordinates = [point.x(), point.y(), point.z()];
        let value: f64 = (0..3)
            .map(|axis| (coordinates[axis] - start[axis]) * direction[axis])
            .sum();
        if !value.is_finite() {
            return Err(format!(
                "edge {key:?}, ray reach {value}: expected a finite distance"
            ));
        }
        reach = reach.min(value);
    }
    Ok(NamedEdgeHit {
        key: key.to_vec(),
        distance,
        reach,
    })
}
