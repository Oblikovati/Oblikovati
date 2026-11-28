// SPDX-License-Identifier: GPL-2.0-only
//! Detached face-draft construction using Martensite `BRepOffsetAPIDraftAngle` (#3918).
use super::*;
use martensite_tkbrep::topods;
use martensite_tkernel::message::ProgressRange;
use martensite_tkmath::gp::{Direction3, Plane3, Point3};
use martensite_tkoffset::brep_offset_api::BRepOffsetAPIDraftAngle;
use martensite_tktopalgo::brep_check::BRepCheckAnalyzer;
use oblikovati_model::feature::draft::{DraftAngleMode, DraftMode, validate_angle_deg};
use std::collections::BTreeSet;

/// Resolved inputs for detached Martensite face-draft construction.
pub(in crate::prism::named) struct DraftRequest<'a> {
    pub mode: DraftMode,
    pub pull_direction: &'a [u8],
    pub parting_plane: Option<([f64; 3], [f64; 3])>,
    pub faces: &'a [Vec<u8>],
    pub angle_deg: f64,
    pub angle_mode: DraftAngleMode,
    pub direction_flipped: bool,
}

impl NamedPrismSnapshot {
    /// Construct a detached drafted solid candidate from named faces (#3918).
    pub(in crate::prism::named) fn draft_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        request: &DraftRequest<'_>,
    ) -> Result<BlendCandidate, String> {
        let source = self.blend_sources(history, document)?;
        let body = self.body_reference_key(history, document)?.to_vec();
        build_draft_candidate(
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

pub(super) fn build_draft_candidate(
    input: BlendInputTopology<'_>,
    history: &DocumentEditHistory,
    document: DocumentGuid,
    request: &DraftRequest<'_>,
) -> Result<BlendCandidate, String> {
    validate_draft_inputs(&input.source, request)?;
    let mut copier = crate::checked_shape::copy_shape(input.body, "draft input")
        .map_err(|cause| cause.to_string())?;
    let sources = copy_named_sources(&mut copier, input.source)?;
    let body = copier.shape().map_err(|cause| cause.to_string())?.clone();

    let pull_shape = named_face_source(&sources, request.pull_direction, "pull direction")?;
    let pull_frame = super::super::face_topology::planar_shape_frame(&body, pull_shape)?;
    let mut pull_vec = pull_frame.normal();
    if request.direction_flipped {
        pull_vec = pull_vec.map(|c| -c);
    }
    let pull_dir = Direction3::new(pull_vec[0], pull_vec[1], pull_vec[2])
        .map_err(|cause| format!("draft pull direction: {cause}"))?;
    let neutral_plane = resolve_neutral_plane(request, pull_frame.recipe().origin, pull_vec)?;
    let angle_rad = request.angle_deg.to_radians();

    let mut maker = BRepOffsetAPIDraftAngle::with_shape(&body)
        .map_err(|cause| format!("draft init: {cause}; errors during the part update"))?;
    for key in request.faces {
        let shape = named_face_source(&sources, key, "draft face")?;
        let face = topods::face(shape)
            .map_err(|cause| format!("draft face {key:?}: expected a face: {cause}"))?;
        maker
            .add(&face, pull_dir, angle_rad, &neutral_plane, true)
            .map_err(|cause| {
                format!(
                    "draft face {key:?}, angle {} deg: {cause}; errors during the part update",
                    request.angle_deg
                )
            })?;
        let add_ok = maker.add_done().map_err(|cause| cause.to_string())?;
        if !add_ok {
            let status = maker.status().ok();
            return Err(format!(
                "draft face {key:?}: cannot draft face ({status:?}); errors during the part update"
            ));
        }
    }
    maker.build(&ProgressRange::default()).map_err(|cause| {
        format!(
            "draft angle {} deg: build failed ({cause}); errors during the part update",
            request.angle_deg
        )
    })?;
    if !maker.is_done() {
        let status = maker.status().ok();
        return Err(format!(
            "draft angle {} deg: operation did not complete ({status:?}); errors during the part update",
            request.angle_deg
        ));
    }
    let shape = maker
        .shape()
        .map_err(|cause| {
            format!(
                "draft angle {} deg: expected a completed shape ({cause}); errors during the part update",
                request.angle_deg
            )
        })?
        .clone();
    crate::checked_shape::require_geometry(&shape, "draft").map_err(|cause| {
        format!("draft geometry validation failed ({cause}); errors during the part update")
    })?;
    let valid = BRepCheckAnalyzer::new(&shape, true, false, false)
        .and_then(|analyzer| analyzer.is_valid())
        .map_err(|cause| {
            format!("draft BRep check failed ({cause}); errors during the part update")
        })?;
    if !valid {
        return Err(format!(
            "draft angle {} deg: resulting solid failed topology validation; errors during the part update",
            request.angle_deg
        ));
    }
    let achieved = if input.resolution.is_some() {
        Some(super::tolerances::check_budget(
            &body,
            &shape,
            input.resolution,
            "draft",
        )?)
    } else {
        None
    };
    let records = draft_history_records(&maker, &sources)?;
    Ok(BlendCandidate {
        resolution: input.resolution,
        achieved_tolerance: achieved,
        shape,
        history: records,
        source_bodies: input.source_bodies,
        body_parent: input.body_parent,
        primary_body: input.primary_body,
        inherited_deleted: input.inherited_deleted,
        inherited_splits: input.inherited_splits,
        document,
        scope: history.evaluation_scope(),
    })
}

fn resolve_neutral_plane(
    request: &DraftRequest<'_>,
    pull_origin: [f64; 3],
    pull_vec: [f64; 3],
) -> Result<Plane3, String> {
    let (origin, normal) = match request.mode {
        DraftMode::FixedEdge | DraftMode::FixedPlane => (pull_origin, pull_vec),
        DraftMode::PartingTool => {
            let (part_origin, part_normal) = request
                .parting_plane
                .ok_or("draft parting tool: expected a resolved parting plane")?;
            let alignment = (part_normal[0] * pull_vec[0]
                + part_normal[1] * pull_vec[1]
                + part_normal[2] * pull_vec[2])
                .abs();
            // When the parting plane is transverse to the pull direction, use it directly.
            // When parallel to pull_dir (e.g. XZ origin plane on a Z-extruded block), anchor the
            // neutral plane at the parting plane's origin normal to the pull direction.
            if alignment > 1e-3 {
                (part_origin, part_normal)
            } else {
                (part_origin, pull_vec)
            }
        }
    };
    let dir = Direction3::new(normal[0], normal[1], normal[2])
        .map_err(|cause| format!("draft neutral plane normal: {cause}"))?;
    Plane3::new(Point3::new(origin[0], origin[1], origin[2]), dir)
        .map_err(|cause| format!("draft neutral plane: {cause}"))
}

fn draft_history_records(
    maker: &BRepOffsetAPIDraftAngle,
    sources: &[(Vec<u8>, Shape)],
) -> Result<Vec<BlendHistoryRecord>, String> {
    let mut records = Vec::with_capacity(sources.len());
    for (name, source) in sources {
        let modified = match maker.modified_shape(source) {
            Ok(image) if !image.is_same(source) => vec![image],
            _ => Vec::new(),
        };
        records.push(BlendHistoryRecord {
            name: name.clone(),
            source: source.clone(),
            generated: Vec::new(),
            modified,
            deleted: false,
            transient: false,
        });
    }
    Ok(records)
}

fn validate_draft_inputs(
    sources: &[(Vec<u8>, Shape)],
    request: &DraftRequest<'_>,
) -> Result<(), String> {
    validate_angle_deg(request.angle_deg)?;
    let _ = request.angle_mode;
    let unique: BTreeSet<_> = request.faces.iter().collect();
    if request.faces.is_empty() || unique.len() != request.faces.len() {
        return Err(format!(
            "draft faces {}, unique {}: expected unique nonempty faces",
            request.faces.len(),
            unique.len()
        ));
    }
    let _ = named_face_source(sources, request.pull_direction, "pull direction")?;
    for key in request.faces {
        if key.as_slice() == request.pull_direction {
            return Err(
                "draft face matches pull direction face; errors during the part update".into(),
            );
        }
        let _ = named_face_source(sources, key, "draft face")?;
    }
    Ok(())
}

fn named_face_source<'a>(
    sources: &'a [(Vec<u8>, Shape)],
    key: &[u8],
    label: &str,
) -> Result<&'a Shape, String> {
    let matches: Vec<_> = sources
        .iter()
        .filter(|(name, _)| name == key)
        .map(|(_, shape)| shape)
        .collect();
    let [shape] = matches.as_slice() else {
        return Err(format!(
            "draft {label} {key:?}, matches {}: expected exactly one named face in the source body",
            matches.len()
        ));
    };
    topods::face(shape)
        .map_err(|cause| format!("draft {label} {key:?}: expected a face: {cause}"))?;
    Ok(shape)
}
