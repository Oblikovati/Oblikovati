// SPDX-License-Identifier: GPL-2.0-only
//! Detached rounded/rectangular machined Slot cuts with exact Boolean provenance.
use super::*;
use martensite_tkg3d::top_abs::ShapeEnum;
use martensite_tkmath::gp::{Axis2, Circle3, Direction3, Point3, Vector3};
use martensite_tkoffset::brep_offset_api::BRepOffsetAPIThruSections;
use martensite_tkprim::brep_prim_api::BRepPrimAPIMakePrism;
use martensite_tktopalgo::brep_builder_api::{
    BRepBuilderAPIMakeEdge, BRepBuilderAPIMakeFace, BRepBuilderAPIMakePolygon,
    BRepBuilderAPIMakeShape, BRepBuilderAPIMakeWire,
};
use oblikovati_model::{
    feature::slot::{ResolvedSlotSeat, SlotShape},
    identity::DerivedReferenceName,
};

const EXTEND_START_CLEARANCE: f64 = 0.4;

pub(in crate::prism::named) struct SlotRequest<'a> {
    pub feature: FeatureId,
    pub face: &'a [u8],
    pub axis: [[f64; 3]; 2],
    pub shape: SlotShape,
    pub length: f64,
    pub width: f64,
    pub rotation_deg: f64,
    pub depth: f64,
    pub seat: ResolvedSlotSeat,
    pub extend_start: bool,
}

impl NamedPrismSnapshot {
    pub(in crate::prism::named) fn slot_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        request: &SlotRequest<'_>,
    ) -> Result<BlendCandidate, String> {
        let source = self.blend_sources(history, document)?;
        let body = self.body_reference_key(history, document)?.to_vec();
        build_slot_candidate(
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

pub(super) fn build_slot_candidate(
    input: BlendInputTopology<'_>,
    history: &DocumentEditHistory,
    document: DocumentGuid,
    request: &SlotRequest<'_>,
) -> Result<BlendCandidate, String> {
    let tools = build_slot_tools(document, request, input.resolution)?;
    super::solid_boolean::build_boolean_candidate(
        input,
        history,
        document,
        tools,
        oblikovati_model::feature::sketch_prism::PrismBoolean::Cut,
        "slot",
    )
    .map_err(|failure| failure.to_string())
}

/// Build the exact B-rep cut tools for a machined Slot (bore prism + optional seat).
pub(crate) fn build_slot_tools(
    document: DocumentGuid,
    request: &SlotRequest<'_>,
    resolution: Option<PrismResolution>,
) -> Result<Vec<super::solid_boolean::BooleanTool>, String> {
    let SlotRequest {
        feature,
        face: face_key,
        axis,
        shape,
        length,
        width,
        rotation_deg,
        depth,
        seat,
        extend_start,
    } = *request;
    if !length.is_finite()
        || !width.is_finite()
        || !depth.is_finite()
        || length <= 0.0
        || width <= 0.0
        || depth <= 0.0
        || length + 1e-9 < width
    {
        return Err(format!(
            "slot length {length}, width {width}, depth {depth}: expected positive dimensions with length >= width"
        ));
    }
    let start_ext = if extend_start {
        EXTEND_START_CLEARANCE
    } else {
        0.0
    };
    let frame = slot_frame(axis, rotation_deg)?;
    let bore = build_slot_prism_tool(
        document,
        feature,
        face_key,
        &frame,
        shape,
        length,
        width,
        -start_ext,
        depth + start_ext,
        "slot",
        resolution,
    )?;
    let mut tools = vec![bore];
    match seat {
        ResolvedSlotSeat::None => {}
        ResolvedSlotSeat::Counterbore {
            width: sw,
            depth: sd,
        }
        | ResolvedSlotSeat::Spotface {
            width: sw,
            depth: sd,
        } => {
            if sw <= width || sd <= 0.0 {
                return Err(format!(
                    "slot seat width {sw}, depth {sd}: expected width > {width} and positive depth"
                ));
            }
            let seat_len = length + (sw - width);
            let seat_tool = build_slot_prism_tool(
                document,
                feature,
                face_key,
                &frame,
                shape,
                seat_len,
                sw,
                -start_ext - 0.02,
                sd + start_ext + 0.02,
                "slot-seat",
                resolution,
            )?;
            tools.push(seat_tool);
        }
        ResolvedSlotSeat::Countersink {
            width: sw,
            angle_deg,
        } => {
            if sw <= width || !angle_deg.is_finite() || angle_deg <= 0.0 || angle_deg >= 180.0 {
                return Err(format!(
                    "slot countersink width {sw}, angle {angle_deg}: expected width > {width} and angle in (0, 180)"
                ));
            }
            let half_tan = (angle_deg * 0.5).to_radians().tan();
            let cone_h = ((sw - width) * 0.5) / half_tan;
            let extra = start_ext + 0.02;
            let top_w = sw + 2.0 * extra * half_tan;
            let top_l = length + (top_w - width);
            let top_wire = build_slot_wire(&frame, shape, top_l, top_w, -extra)?;
            let bot_wire = build_slot_wire(&frame, shape, length, width, cone_h)?;
            let tol = resolution.map_or(1e-5, |r| r.maximum_tolerance);
            let mut loft = BRepOffsetAPIThruSections::new(true, true, tol);
            loft.add_wire(&top_wire);
            loft.add_wire(&bot_wire);
            let seat_shape = loft
                .shape()
                .map_err(|error| format!("slot countersink loft: {error}"))?
                .clone();
            let sources = name_shape_faces(&seat_shape, document, feature, face_key, "slot-seat")?;
            tools.push(super::solid_boolean::BooleanTool {
                shape: seat_shape,
                sources,
                resolution,
            });
        }
    }
    Ok(tools)
}

struct SlotFrame {
    origin: [f64; 3],
    u: [f64; 3],
    v: [f64; 3],
    z: [f64; 3],
}

impl SlotFrame {
    fn pt(&self, x: f64, y: f64, z: f64) -> Point3 {
        Point3::new(
            self.origin[0] + x * self.u[0] + y * self.v[0] + z * self.z[0],
            self.origin[1] + x * self.u[1] + y * self.v[1] + z * self.z[1],
            self.origin[2] + x * self.u[2] + y * self.v[2] + z * self.z[2],
        )
    }
}

fn slot_frame(axis: [[f64; 3]; 2], rotation_deg: f64) -> Result<SlotFrame, String> {
    let [origin, dir] = axis;
    let z_dir = Direction3::new(dir[0], dir[1], dir[2]).map_err(|e| e.to_string())?;
    let ax2 = Axis2::new(Point3::new(origin[0], origin[1], origin[2]), z_dir)
        .map_err(|e| e.to_string())?;
    let base_u = ax2.x_direction();
    let base_v = ax2.y_direction();
    let rad = rotation_deg.to_radians();
    let (s, c) = rad.sin_cos();
    let u = [
        base_u.x() * c + base_v.x() * s,
        base_u.y() * c + base_v.y() * s,
        base_u.z() * c + base_v.z() * s,
    ];
    let v = [
        -base_u.x() * s + base_v.x() * c,
        -base_u.y() * s + base_v.y() * c,
        -base_u.z() * s + base_v.z() * c,
    ];
    Ok(SlotFrame {
        origin,
        u,
        v,
        z: [z_dir.x(), z_dir.y(), z_dir.z()],
    })
}

#[allow(clippy::too_many_arguments)]
fn build_slot_prism_tool(
    document: DocumentGuid,
    feature: FeatureId,
    face_key: &[u8],
    frame: &SlotFrame,
    shape: SlotShape,
    length: f64,
    width: f64,
    z_start: f64,
    extrude_depth: f64,
    prefix: &str,
    resolution: Option<PrismResolution>,
) -> Result<super::solid_boolean::BooleanTool, String> {
    let wire = build_slot_wire(frame, shape, length, width, z_start)?;
    let face: Shape = BRepBuilderAPIMakeFace::from_wire(&wire, true)
        .map_err(|error| format!("slot profile face: {error}"))?
        .face()
        .map_err(|error| format!("slot profile face: {error}"))?
        .into();
    let vec = Vector3::new(
        frame.z[0] * extrude_depth,
        frame.z[1] * extrude_depth,
        frame.z[2] * extrude_depth,
    );
    let mut prism = BRepPrimAPIMakePrism::new(&face, vec, false, true)
        .map_err(|error| format!("slot prism: {error}"))?;
    let solid = prism
        .shape()
        .map_err(|error| format!("slot prism: {error}"))?
        .clone();
    let sources = name_shape_faces(&solid, document, feature, face_key, prefix)?;
    Ok(super::solid_boolean::BooleanTool {
        shape: solid,
        sources,
        resolution,
    })
}

fn build_slot_wire(
    frame: &SlotFrame,
    shape: SlotShape,
    length: f64,
    width: f64,
    z: f64,
) -> Result<martensite_tkbrep::topods::Wire, String> {
    let hl = length * 0.5;
    let hw = width * 0.5;
    match shape {
        SlotShape::Rectangular => {
            let mut poly = BRepBuilderAPIMakePolygon::new();
            for (x, y) in [(-hl, -hw), (hl, -hw), (hl, hw), (-hl, hw)] {
                poly.add_point(frame.pt(x, y, z))
                    .map_err(|e| format!("slot rect point: {e}"))?;
            }
            poly.close().map_err(|e| format!("slot rect close: {e}"))?;
            poly.wire().map_err(|e| format!("slot rect wire: {e}"))
        }
        SlotShape::Rounded => {
            let half_straight = (hl - hw).max(0.0);
            let z_dir =
                Direction3::new(frame.z[0], frame.z[1], frame.z[2]).map_err(|e| e.to_string())?;
            if half_straight <= 1e-6 {
                let ax = Axis2::new(frame.pt(0.0, 0.0, z), z_dir).map_err(|e| e.to_string())?;
                let circ = Circle3::new(ax, hw).map_err(|e| e.to_string())?;
                let edge = BRepBuilderAPIMakeEdge::from_circ(circ)
                    .map_err(|e| format!("slot circle edge: {e}"))?
                    .edge()
                    .map_err(|e| format!("slot circle edge: {e}"))?;
                return BRepBuilderAPIMakeWire::from_edge(&edge)
                    .map_err(|e| format!("slot circle wire: {e}"))?
                    .wire()
                    .map_err(|e| format!("slot circle wire: {e}"));
            }
            let p_bl = frame.pt(-half_straight, -hw, z);
            let p_br = frame.pt(half_straight, -hw, z);
            let p_tr = frame.pt(half_straight, hw, z);
            let p_tl = frame.pt(-half_straight, hw, z);
            let ax_r =
                Axis2::new(frame.pt(half_straight, 0.0, z), z_dir).map_err(|e| e.to_string())?;
            let ax_l =
                Axis2::new(frame.pt(-half_straight, 0.0, z), z_dir).map_err(|e| e.to_string())?;
            let circ_r = Circle3::new(ax_r, hw).map_err(|e| e.to_string())?;
            let circ_l = Circle3::new(ax_l, hw).map_err(|e| e.to_string())?;
            let e1 = BRepBuilderAPIMakeEdge::from_points(p_bl, p_br)
                .map_err(|e| format!("slot bottom edge: {e}"))?
                .edge()
                .map_err(|e| format!("slot bottom edge: {e}"))?;
            let e2 = BRepBuilderAPIMakeEdge::from_circ_points(circ_r, p_br, p_tr)
                .map_err(|e| format!("slot right arc: {e}"))?
                .edge()
                .map_err(|e| format!("slot right arc: {e}"))?;
            let e3 = BRepBuilderAPIMakeEdge::from_points(p_tr, p_tl)
                .map_err(|e| format!("slot top edge: {e}"))?
                .edge()
                .map_err(|e| format!("slot top edge: {e}"))?;
            let e4 = BRepBuilderAPIMakeEdge::from_circ_points(circ_l, p_tl, p_bl)
                .map_err(|e| format!("slot left arc: {e}"))?
                .edge()
                .map_err(|e| format!("slot left arc: {e}"))?;
            let mut wire = BRepBuilderAPIMakeWire::new();
            for edge in [&e1, &e2, &e3, &e4] {
                wire.add_edge(edge)
                    .map_err(|e| format!("slot stadium wire: {e}"))?;
            }
            wire.wire().map_err(|e| format!("slot stadium wire: {e}"))
        }
    }
}

fn name_shape_faces(
    shape: &Shape,
    document: DocumentGuid,
    feature: FeatureId,
    parent: &[u8],
    prefix: &str,
) -> Result<Vec<(Vec<u8>, Shape)>, String> {
    let faces = super::solid_boolean::entities(shape, ShapeEnum::Face)?;
    faces
        .into_iter()
        .enumerate()
        .map(|(idx, face)| {
            let role = format!("{prefix}-face-{idx}");
            let name =
                DerivedReferenceName::new(document, feature.0, 1, &role, &[parent.to_vec()])?
                    .encode()?;
            Ok((name, face))
        })
        .collect()
}
