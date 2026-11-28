// SPDX-License-Identifier: GPL-2.0-only
use super::{
    result_names::require_unique,
    section_body::{bind_section_body, incident_names},
    *,
};
use crate::revolve::{RevolveCandidate, RevolveFaceRole, SurfaceRevolveCandidate};
use martensite_tkbrep::{
    brep_adaptor::{BRepAdaptorCurve, BRepAdaptorSurface},
    topexp::Explorer,
    topods,
};
use martensite_tkg3d::adaptor3d::{Adaptor3dCurve, Adaptor3dSurface};
use martensite_tkmath::geom_abs::{CurveType, SurfaceType};
use oblikovati_model::identity::SemanticReferenceName;

impl BlendNamedTopology {
    pub(in crate::prism::named) fn from_revolve(
        history: &DocumentEditHistory,
        document: DocumentGuid,
        feature: FeatureId,
        sketch_id: u64,
        candidate: RevolveCandidate,
    ) -> Result<Self, String> {
        let achieved_tolerance = Some(candidate.achieved_tolerance());
        let body =
            SemanticReferenceName::new(document, feature.0, 4, "revolve", "body", &[sketch_id])?
                .encode()?;
        let faces = name_revolve_faces(document, feature, sketch_id, "revolve", candidate.faces)?;
        require_unique(&faces)?;
        let edges = match name_periodic_torus_seams(
            document,
            feature,
            sketch_id,
            &candidate.shape,
            &faces,
        )? {
            Some(seams) => seams,
            None => incident_names(
                document,
                feature,
                &candidate.shape,
                ShapeEnum::Edge,
                2,
                &faces,
            )?,
        };
        let vertices = incident_names(
            document,
            feature,
            &candidate.shape,
            ShapeEnum::Vertex,
            3,
            &edges,
        )?;
        let subentities = faces.into_iter().chain(edges).chain(vertices).collect();
        bind_section_body(
            history,
            document,
            body,
            candidate.shape,
            candidate.resolution,
            achieved_tolerance,
            subentities,
        )
    }

    pub(in crate::prism::named) fn from_surface_revolve(
        history: &DocumentEditHistory,
        document: DocumentGuid,
        feature: FeatureId,
        sketch_id: u64,
        candidate: SurfaceRevolveCandidate,
    ) -> Result<Self, String> {
        let achieved_tolerance = Some(candidate.achieved_tolerance());
        let body = SemanticReferenceName::new(
            document,
            feature.0,
            4,
            "surface-revolve",
            "body",
            &[sketch_id],
        )?
        .encode()?;
        let faces = name_revolve_faces(
            document,
            feature,
            sketch_id,
            "surface-revolve",
            candidate.faces,
        )?;
        require_unique(&faces)?;
        let edges = name_surface_revolve_edges(
            document,
            feature,
            sketch_id,
            &candidate.shell,
            &faces,
            candidate.start_trim.as_ref(),
            candidate.end_trim.as_ref(),
        )?;
        let vertices = incident_names(
            document,
            feature,
            &candidate.shell,
            ShapeEnum::Vertex,
            3,
            &edges,
        )?;
        let subentities = faces.into_iter().chain(edges).chain(vertices).collect();
        bind_section_body(
            history,
            document,
            body,
            candidate.shell,
            candidate.resolution,
            achieved_tolerance,
            subentities,
        )
    }
}

fn name_revolve_faces(
    document: DocumentGuid,
    feature: FeatureId,
    sketch_id: u64,
    namespace: &str,
    source_faces: Vec<(RevolveFaceRole, Shape)>,
) -> Result<Vec<(Vec<u8>, Shape)>, String> {
    let mut faces = Vec::with_capacity(source_faces.len());
    for (role, shape) in source_faces {
        let name = match role {
            RevolveFaceRole::Start => {
                SemanticReferenceName::new(document, feature.0, 1, namespace, "start", &[sketch_id])
            }
            RevolveFaceRole::End => {
                SemanticReferenceName::new(document, feature.0, 1, namespace, "end", &[sketch_id])
            }
            RevolveFaceRole::ProfileEdge(curve) => SemanticReferenceName::new(
                document,
                feature.0,
                1,
                namespace,
                "side",
                &[sketch_id, curve],
            ),
        }?;
        faces.push((name.encode()?, shape));
    }
    Ok(faces)
}

fn name_surface_revolve_edges(
    document: DocumentGuid,
    feature: FeatureId,
    sketch_id: u64,
    shell: &Shape,
    faces: &[(Vec<u8>, Shape)],
    start_trim: Option<&Shape>,
    end_trim: Option<&Shape>,
) -> Result<Vec<(Vec<u8>, Shape)>, String> {
    let start_edges = wire_edges(start_trim)?;
    let end_edges = wire_edges(end_trim)?;
    let mut unique = IndexedMapOfShape::default();
    topexp::map_shapes_of_type(shell, ShapeEnum::Edge, &mut unique)
        .map_err(|cause| format!("surface revolve edges: {cause}"))?;
    let mut named = Vec::with_capacity(unique.extent());
    for index in 1..=unique.extent() {
        let edge = unique
            .find_key(index)
            .map_err(|cause| format!("surface revolve edge {index}: {cause}"))?
            .clone();
        let mut parents: Vec<&[u8]> = faces
            .iter()
            .filter(|(_, face)| face_contains_edge(face, &edge))
            .map(|(key, _)| key.as_slice())
            .collect();
        parents.sort_unstable();
        if parents.is_empty() {
            return Err("surface revolve edge has no named incident face".into());
        }
        let role = if start_edges.iter().any(|item| item.is_same(&edge)) {
            "start-trim"
        } else if end_edges.iter().any(|item| item.is_same(&edge)) {
            "end-trim"
        } else if parents.len() == 1 {
            "seam"
        } else {
            "shared"
        };
        let key = SemanticReferenceName::new(
            document,
            feature.0,
            2,
            "surface-revolve",
            role,
            &edge_sources(sketch_id, &parents),
        )?
        .encode()?;
        named.push((key, edge));
    }
    require_unique(&named)?;
    Ok(named)
}

fn edge_sources(sketch_id: u64, parents: &[&[u8]]) -> Vec<u64> {
    let mut ids = Vec::with_capacity(1 + parents.len());
    ids.push(sketch_id);
    for parent in parents {
        let mut hash = 14695981039346656037_u64;
        for byte in *parent {
            hash ^= u64::from(*byte);
            hash = hash.wrapping_mul(1099511628211);
        }
        ids.push(hash);
    }
    ids
}

fn wire_edges(wire: Option<&Shape>) -> Result<Vec<Shape>, String> {
    let Some(wire) = wire else {
        return Ok(Vec::new());
    };
    Explorer::new(wire, ShapeEnum::Edge, ShapeEnum::Shape)
        .map(|iter| iter.collect())
        .map_err(|cause| format!("surface revolve trim edges: {cause}"))
}

fn face_contains_edge(face: &Shape, edge: &Shape) -> bool {
    Explorer::new(face, ShapeEnum::Edge, ShapeEnum::Shape)
        .ok()
        .is_some_and(|mut iter| iter.any(|candidate| candidate.is_same(edge)))
}

type NamedShapeList = Vec<(Vec<u8>, Shape)>;

/// Name the two periodic seam circles of a full 360-degree torus solid by their geometric
/// orientation relative to the torus revolution axis (parallel vs meridian seam).
fn name_periodic_torus_seams(
    document: DocumentGuid,
    feature: FeatureId,
    sketch_id: u64,
    solid: &Shape,
    faces: &[(Vec<u8>, Shape)],
) -> Result<Option<NamedShapeList>, String> {
    let [(_, face)] = faces else {
        return Ok(None);
    };
    let native_face = topods::face(face).map_err(|cause| cause.to_string())?;
    let surf =
        BRepAdaptorSurface::with_face(&native_face, false).map_err(|cause| cause.to_string())?;
    if surf.get_type().map_err(|cause| cause.to_string())? != SurfaceType::Torus {
        return Ok(None);
    }
    let torus = surf.torus().map_err(|cause| cause.to_string())?;
    let axis = *torus.position().direction().coordinates();
    let all_edges = shapes(solid, ShapeEnum::Edge)?;
    if all_edges.len() != 2 {
        return Ok(None);
    }
    let mut named = Vec::with_capacity(2);
    for edge in all_edges {
        let native_edge = topods::edge(&edge).map_err(|cause| cause.to_string())?;
        let curve = BRepAdaptorCurve::with_edge(&native_edge).map_err(|cause| cause.to_string())?;
        if curve.get_type().map_err(|cause| cause.to_string())? != CurveType::Circle {
            return Ok(None);
        }
        let circle = curve.circle().map_err(|cause| cause.to_string())?;
        let normal = *circle.position().direction().coordinates();
        let alignment = (normal[0] * axis[0] + normal[1] * axis[1] + normal[2] * axis[2]).abs();
        let role = if alignment > 0.5 {
            "seam-parallel"
        } else {
            "seam-meridian"
        };
        let key =
            SemanticReferenceName::new(document, feature.0, 2, "revolve", role, &[sketch_id])?
                .encode()?;
        named.push((key, edge));
    }
    require_unique(&named)?;
    Ok(Some(named))
}
