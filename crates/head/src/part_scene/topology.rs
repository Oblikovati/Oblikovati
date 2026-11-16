// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::wire::{BodyTopology, ReferenceKeysResult, TopologyRef};
use oblikovati_martensite::PrismSurfaceKind;
use oblikovati_model::identity::{DocumentGuid, ReferenceEnvelope};
use oblikovati_session::DocumentSession;

impl DesktopPartScenes {
    pub(super) fn topology_keys<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
        &self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<ReferenceKeysResult, String> {
        let workspace = files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let bodies = self
            .current(session)?
            .ordered_bodies()
            .map(|body| body_topology(session, guid, body))
            .collect::<Result<Vec<_>, _>>()?;
        Ok(ReferenceKeysResult {
            bodies: Some(bodies),
        })
    }
}
fn body_topology<O: TransactionObserver>(
    session: &DocumentSession<O>,
    guid: DocumentGuid,
    body: &NamedSolid,
) -> Result<BodyTopology, String> {
    let history = session.history();
    let mut faces = Vec::new();
    let mut edges = Vec::new();
    let mut vertices = Vec::new();
    for key in body.face_reference_keys(history, guid)? {
        if ReferenceEnvelope::decode(&key)
            .map_err(|cause| cause.to_string())?
            .kind
            != 1
        {
            continue;
        }
        let face = body.face_topology(history, guid, &key)?;
        faces.push(TopologyRef {
            key: super::bodies::body_reference_text(&key),
            point: Some(face.point.to_vec()),
            kind: surface_name(face.surface).into(),
        });
    }
    for key in body.subentity_reference_keys(history, guid)? {
        let kind = ReferenceEnvelope::decode(&key)
            .map_err(|cause| cause.to_string())?
            .kind;
        let point = body.subentity_point(history, guid, &key)?;
        let target = match kind {
            2 => &mut edges,
            3 => &mut vertices,
            _ => {
                return Err(format!(
                    "topology key kind {kind}: expected edge 2 or vertex 3"
                ));
            }
        };
        target.push(TopologyRef {
            key: super::bodies::body_reference_text(&key),
            point: Some(point.to_vec()),
            kind: String::new(),
        });
    }
    Ok(BodyTopology {
        faces: Some(faces),
        edges: Some(edges),
        vertices: Some(vertices),
    })
}
fn surface_name(kind: PrismSurfaceKind) -> &'static str {
    match kind {
        PrismSurfaceKind::Plane => "plane",
        PrismSurfaceKind::Cylinder => "cylinder",
        PrismSurfaceKind::Cone => "cone",
        PrismSurfaceKind::Sphere => "sphere",
        PrismSurfaceKind::Torus => "torus",
        PrismSurfaceKind::Spline => "spline",
    }
}
