// SPDX-License-Identifier: GPL-2.0-only
//! Detached cylindrical, seated, and drill-point cuts with exact Boolean provenance for Hole (#4093).
use super::*;
use martensite_tkg3d::top_abs::ShapeEnum;
use martensite_tkmath::gp::{Axis1, Axis2, Direction3, Point3};
use martensite_tkprim::{
    brep_prim_api::{BRepPrimAPIMakeCylinder, BRepPrimAPIMakeOneAxis, BRepPrimAPIMakeRevol},
    brep_sweep::BRepSweepRevol,
};
use martensite_tktopalgo::{
    brep_builder_api::{
        BRepBuilderAPIMakeFace, BRepBuilderAPIMakePolygon, BRepBuilderAPIMakeShape,
    },
    brep_lib,
};
use oblikovati_model::{
    feature::hole::{HoleDrillPoint, ResolvedHoleSeat},
    identity::DerivedReferenceName,
};

const EXTEND_START_CLEARANCE: f64 = 0.4;

pub(in crate::prism::named) struct HoleRequest<'a> {
    pub feature: FeatureId,
    pub face: &'a [u8],
    pub axis: [[f64; 3]; 2],
    pub diameter: f64,
    pub depth: f64,
    pub seat: ResolvedHoleSeat,
    pub drill_point: HoleDrillPoint,
    pub extend_start: bool,
    pub finite_depth: bool,
}

impl NamedPrismSnapshot {
    pub(in crate::prism::named) fn hole_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        request: &HoleRequest<'_>,
    ) -> Result<BlendCandidate, String> {
        let source = self.blend_sources(history, document)?;
        let body = self.body_reference_key(history, document)?.to_vec();
        build_hole_candidate(
            BlendInputTopology {
                body: &self.snapshot.shape,
                source,
                source_bodies: vec![body.clone()],
                body_parent: body.clone(),
                primary_body: body,
                inherited_deleted: Vec::new(),
                inherited_splits: Default::default(),
                resolution: self.certification.as_ref().map(|value| value.resolution),
            },
            history,
            document,
            request,
        )
    }
}

pub(super) fn build_hole_candidate(
    input: BlendInputTopology<'_>,
    history: &DocumentEditHistory,
    document: DocumentGuid,
    request: &HoleRequest<'_>,
) -> Result<BlendCandidate, String> {
    let tool = build_hole_tool(document, request, input.resolution)?;
    super::solid_boolean::build_boolean_candidate(
        input,
        history,
        document,
        vec![tool],
        oblikovati_model::feature::sketch_prism::PrismBoolean::Cut,
        "hole",
    )
    .map_err(|failure| failure.to_string())
}

/// Build the single exact B-rep cut tool for a Hole (plain cylinder or revolved seated/angled solid).
pub(crate) fn build_hole_tool(
    document: DocumentGuid,
    request: &HoleRequest<'_>,
    resolution: Option<PrismResolution>,
) -> Result<super::solid_boolean::BooleanTool, String> {
    let HoleRequest {
        feature,
        face: face_key,
        axis,
        diameter,
        depth,
        seat,
        drill_point,
        extend_start,
        finite_depth,
    } = *request;
    if !diameter.is_finite() || diameter <= 0. || !depth.is_finite() || depth <= 0. {
        return Err(format!(
            "hole diameter {diameter}, depth {depth}: expected finite positive lengths"
        ));
    }
    let start_ext = if extend_start {
        EXTEND_START_CLEARANCE
    } else {
        0.0
    };
    let has_seat = !matches!(seat, ResolvedHoleSeat::None);
    let has_angled_tip = finite_depth && matches!(drill_point, HoleDrillPoint::Angle(_));
    if !has_seat && !has_angled_tip {
        let shifted_axis = [shift(axis[0], axis[1], -start_ext), axis[1]];
        let mut maker = cylinder(shifted_axis, diameter, depth + start_ext)?;
        let shape = maker.shape().map_err(|error| error.to_string())?.clone();
        let sources = cylinder_faces(&mut maker, document, feature, face_key, "hole")?;
        return Ok(super::solid_boolean::BooleanTool {
            shape,
            sources,
            resolution: extend_start.then_some(resolution).flatten(),
        });
    }
    build_revolved_hole_tool(document, request, start_ext, resolution)
}

/// Build a single exact revolved solid for a seated and/or angled-drill-point hole.
fn build_revolved_hole_tool(
    document: DocumentGuid,
    request: &HoleRequest<'_>,
    start_ext: f64,
    resolution: Option<PrismResolution>,
) -> Result<super::solid_boolean::BooleanTool, String> {
    let segments = hole_meridian_segments(
        request.diameter,
        request.depth,
        request.seat,
        request.drill_point,
        request.finite_depth,
        start_ext,
    )?;
    let ax2 = axis2(request.axis)?;
    let r_dir = ax2.x_direction();
    let z_dir = ax2.direction();
    let origin = request.axis[0];
    let to_point = |r: f64, z: f64| {
        Point3::new(
            origin[0] + r * r_dir.x() + z * z_dir.x(),
            origin[1] + r * r_dir.y() + z * z_dir.y(),
            origin[2] + r * r_dir.z() + z * z_dir.z(),
        )
    };
    let mut poly = BRepBuilderAPIMakePolygon::new();
    let Some((first, _, _)) = segments.first().copied() else {
        return Err("hole meridian: expected nonempty profile".into());
    };
    poly.add_point(to_point(first.0, first.1))
        .map_err(|error| format!("hole meridian start: {error}"))?;
    let mut named_edges = Vec::with_capacity(segments.len());
    for &(_, p1, role) in &segments[..segments.len() - 1] {
        poly.add_point(to_point(p1.0, p1.1))
            .map_err(|error| format!("hole meridian point: {error}"))?;
        if let Some(role) = role {
            if !poly.added() || poly.edge().is_null() {
                return Err(format!(
                    "hole meridian edge {role}: expected non-degenerate segment"
                ));
            }
            named_edges.push((role, Shape::from(poly.edge().clone())));
        }
    }
    poly.close()
        .map_err(|error| format!("hole meridian close: {error}"))?;
    let wire = poly
        .wire()
        .map_err(|error| format!("hole meridian wire: {error}"))?;
    let face: Shape = BRepBuilderAPIMakeFace::from_wire(&wire, true)
        .map_err(|error| format!("hole meridian face: {error}"))?
        .face()
        .map_err(|error| format!("hole meridian face: {error}"))?
        .into();
    let rev_axis = Axis1::new(Point3::new(origin[0], origin[1], origin[2]), z_dir);
    let validator = BRepPrimAPIMakeRevol::full(&face, rev_axis, false)
        .map_err(|error| format!("hole revolve validation: {error}"))?;
    if !validator.is_done() {
        return Err("hole revolve: expected completed solid revolution".into());
    }
    let mut sweep = BRepSweepRevol::full(&face, rev_axis, false)
        .map_err(|error| format!("hole revolve sweep: {error}"))?;
    let shape = sweep
        .shape()
        .map_err(|error| format!("hole revolve shape: {error}"))?;
    brep_lib::update_inner_tolerances(&shape)
        .map_err(|error| format!("hole revolve tolerances: {error}"))?;
    let mut sources = Vec::with_capacity(named_edges.len());
    for (role, edge) in named_edges {
        let generated = sweep
            .shape_of(&edge)
            .map_err(|error| format!("hole revolve history {role}: {error}"))?;
        if generated.is_null() || generated.shape_type().ok() != Some(ShapeEnum::Face) {
            continue;
        }
        let name = DerivedReferenceName::new(
            document,
            request.feature.0,
            1,
            role,
            &[request.face.to_vec()],
        )?
        .encode()?;
        sources.push((name, generated.clone()));
        if role == "hole-tip-cone" {
            for edge in super::solid_boolean::entities(&generated, ShapeEnum::Edge)? {
                let vertices = super::solid_boolean::entities(&edge, ShapeEnum::Vertex)?;
                if vertices.len() == 1 {
                    let apex_name = DerivedReferenceName::new(
                        document,
                        request.feature.0,
                        2,
                        "hole-tip-apex",
                        &[request.face.to_vec()],
                    )?
                    .encode()?;
                    sources.push((apex_name, edge));
                }
            }
        }
    }
    Ok(super::solid_boolean::BooleanTool {
        shape,
        sources,
        resolution,
    })
}

type MeridianSegment = ((f64, f64), (f64, f64), Option<&'static str>);

fn hole_meridian_segments(
    diameter: f64,
    depth: f64,
    seat: ResolvedHoleSeat,
    drill_point: HoleDrillPoint,
    finite_depth: bool,
    start_ext: f64,
) -> Result<Vec<MeridianSegment>, String> {
    let rb = diameter * 0.5;
    let z0 = -start_ext;
    let mut pts: Vec<((f64, f64), Option<&'static str>)> = vec![((0.0, z0), None)];
    let mut push = |pt: (f64, f64), role: Option<&'static str>| {
        pts.push((pt, role));
    };
    let r_bottom = match seat {
        ResolvedHoleSeat::None => {
            push((rb, z0), Some("hole-start"));
            push((rb, depth), Some("hole-wall"));
            rb
        }
        ResolvedHoleSeat::Counterbore {
            diameter: ds,
            depth: hs,
        }
        | ResolvedHoleSeat::Spotface {
            diameter: ds,
            depth: hs,
        } => {
            if ds <= diameter || hs <= 0. {
                return Err(format!(
                    "hole seat diameter {ds}, depth {hs}: expected diameter > {diameter} and positive depth"
                ));
            }
            let rs = ds * 0.5;
            push((rs, z0), Some("hole-seat-start"));
            if hs + 1e-7 < depth {
                push((rs, hs), Some("hole-seat-wall"));
                push((rb, hs), Some("hole-seat-step"));
                push((rb, depth), Some("hole-wall"));
                rb
            } else {
                push((rs, depth), Some("hole-seat-wall"));
                rs
            }
        }
        ResolvedHoleSeat::Countersink {
            diameter: ds,
            angle_deg,
        } => {
            if ds <= diameter || !angle_deg.is_finite() || angle_deg <= 0. || angle_deg >= 180. {
                return Err(format!(
                    "hole countersink diameter {ds}, angle {angle_deg}: expected diameter > {diameter} and angle in (0, 180)"
                ));
            }
            let half_tan = (angle_deg * 0.5).to_radians().tan();
            let cone_h = (ds * 0.5 - rb) / half_tan;
            let r_top = ds * 0.5 + start_ext * half_tan;
            push((r_top, z0), Some("hole-seat-start"));
            if cone_h + 1e-7 < depth {
                push((rb, cone_h), Some("hole-seat-cone"));
                push((rb, depth), Some("hole-wall"));
                rb
            } else {
                let r_end = (ds * 0.5 - depth * half_tan).max(rb);
                push((r_end, depth), Some("hole-seat-cone"));
                r_end
            }
        }
    };
    let z_axis_end = if finite_depth && let HoleDrillPoint::Angle(angle_deg) = drill_point {
        if !angle_deg.is_finite() || angle_deg <= 0. || angle_deg >= 180. {
            return Err(format!(
                "hole drill point angle {angle_deg}: expected angle in (0, 180)"
            ));
        }
        let half_tan = (angle_deg * 0.5).to_radians().tan();
        let tip_h = r_bottom / half_tan;
        let z_tip = depth + tip_h;
        push((0.0, z_tip), Some("hole-tip-cone"));
        z_tip
    } else {
        push((0.0, depth), Some("hole-end"));
        depth
    };
    let mut segments = Vec::with_capacity(pts.len());
    for pair in pts.windows(2) {
        segments.push((pair[0].0, pair[1].0, pair[1].1));
    }
    segments.push(((0.0, z_axis_end), (0.0, z0), None));
    Ok(segments)
}

fn shift(origin: [f64; 3], direction: [f64; 3], distance: f64) -> [f64; 3] {
    std::array::from_fn(|i| origin[i] + direction[i] * distance)
}

fn axis2(axis: [[f64; 3]; 2]) -> Result<Axis2, String> {
    let [origin, vector] = axis;
    let direction =
        Direction3::new(vector[0], vector[1], vector[2]).map_err(|error| error.to_string())?;
    Axis2::new(Point3::new(origin[0], origin[1], origin[2]), direction)
        .map_err(|error| error.to_string())
}

fn cylinder(
    axis: [[f64; 3]; 2],
    diameter: f64,
    depth: f64,
) -> Result<BRepPrimAPIMakeCylinder, String> {
    Ok(BRepPrimAPIMakeCylinder::with_axes(
        axis2(axis)?,
        diameter / 2.,
        depth,
    ))
}

fn cylinder_faces(
    maker: &mut BRepPrimAPIMakeCylinder,
    document: DocumentGuid,
    feature: FeatureId,
    parent: &[u8],
    prefix: &str,
) -> Result<Vec<(Vec<u8>, Shape)>, String> {
    let axis = maker.one_axis();
    let faces = [
        (format!("{prefix}-wall"), axis.lateral_face()),
        (format!("{prefix}-end"), axis.top_face()),
        (format!("{prefix}-start"), axis.bottom_face()),
    ];
    encode_faces(faces, document, feature, parent)
}

fn encode_faces<E: std::fmt::Display>(
    faces: impl IntoIterator<Item = (String, Result<impl Into<Shape>, E>)>,
    document: DocumentGuid,
    feature: FeatureId,
    parent: &[u8],
) -> Result<Vec<(Vec<u8>, Shape)>, String> {
    faces
        .into_iter()
        .map(|(role, face)| {
            let name =
                DerivedReferenceName::new(document, feature.0, 1, &role, &[parent.to_vec()])?
                    .encode()?;
            Ok((
                name,
                face.map_err(|error| format!("hole {role}: expected a tool face: {error}"))?
                    .into(),
            ))
        })
        .collect()
}
