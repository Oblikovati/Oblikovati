// SPDX-License-Identifier: GPL-2.0-only
//! Resolve exact vertices and circular-edge centers for sketch frames (#4107).

use martensite_tkbrep::{
    brep::BRepTool,
    brep_adaptor::BRepAdaptorCurve,
    topods::{self, Shape},
};
use martensite_tkg3d::adaptor3d::Adaptor3dCurve;
use martensite_tkmath::{geom_abs, gp::Point3};

pub(super) fn analytic_frame_origin(shape: &Shape, kind: u32) -> Result<[f64; 3], String> {
    let point = match kind {
        2 => circle_center(shape)?,
        3 => BRepTool::pnt(&topods::vertex(shape).map_err(|error| error.to_string())?)
            .map_err(|error| error.to_string())?,
        _ => {
            return Err(format!(
                "frame origin kind {kind}: expected circular edge 2 or vertex 3"
            ));
        }
    };
    let at = [point.x(), point.y(), point.z()];
    if !at.iter().all(|value| value.is_finite()) {
        return Err(format!(
            "frame origin {at:?}: expected finite exact coordinates"
        ));
    }
    Ok(at)
}
fn circle_center(shape: &Shape) -> Result<Point3, String> {
    let edge = topods::edge(shape).map_err(|error| error.to_string())?;
    let curve = BRepAdaptorCurve::with_edge(&edge).map_err(|error| error.to_string())?;
    let kind = curve.get_type().map_err(|error| error.to_string())?;
    if kind != geom_abs::CurveType::Circle {
        return Err(format!(
            "frame origin edge {kind:?}: expected an analytic circle"
        ));
    }
    curve
        .circle()
        .map(|circle| circle.location())
        .map_err(|error| error.to_string())
}

impl super::NamedPrismSnapshot {
    /// Resolve an exact named vertex or circular-edge center; example: `body.frame_origin(history, document, key)?`.
    pub fn frame_origin(
        &self,
        history: &oblikovati_model::history::DocumentEditHistory,
        document: oblikovati_model::identity::DocumentGuid,
        key: &[u8],
    ) -> Result<[f64; 3], String> {
        let kind = oblikovati_model::identity::ReferenceEnvelope::decode(key)
            .map_err(|error| error.to_string())?
            .kind;
        if !matches!(kind, 2 | 3) {
            return Err(format!(
                "frame origin key {key:?}, kind {kind}: expected edge 2 or vertex 3"
            ));
        }
        let bindings = self
            .subentity_bindings
            .as_ref()
            .ok_or("frame origin: expected certified named subentities")?;
        let entity = bindings.bind(history, document, key, kind)?;
        analytic_frame_origin(&entity.shape, kind)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use martensite_tkmath::gp::{Axis2, Circle3, Direction3};
    use martensite_tktopalgo::brep_builder_api::{
        BRepBuilderAPIMakeEdge, BRepBuilderAPIMakeVertex,
    };

    #[test]
    fn circular_arc_uses_analytic_center_not_bounds_center() {
        let center = Point3::new(5., 7., 0.);
        let axes = Axis2::new(center, Direction3::new(0., 0., 1.).unwrap()).unwrap();
        let circle = Circle3::new(axes, 2.).unwrap();
        let edge: Shape = BRepBuilderAPIMakeEdge::from_circ_points(
            circle,
            Point3::new(7., 7., 0.),
            Point3::new(5., 9., 0.),
        )
        .unwrap()
        .edge()
        .unwrap()
        .into();
        assert_eq!(analytic_frame_origin(&edge, 2).unwrap(), [5., 7., 0.]);
    }
    #[test]
    fn vertex_is_exact_and_non_circular_edges_are_rejected() {
        let vertex: Shape = BRepBuilderAPIMakeVertex::new(Point3::new(3., 4., 5.))
            .unwrap()
            .vertex()
            .unwrap()
            .into();
        assert_eq!(analytic_frame_origin(&vertex, 3).unwrap(), [3., 4., 5.]);
        let edge: Shape =
            BRepBuilderAPIMakeEdge::from_points(Point3::new(0., 0., 0.), Point3::new(1., 0., 0.))
                .unwrap()
                .edge()
                .unwrap()
                .into();
        assert!(analytic_frame_origin(&edge, 2).is_err());
    }
}
