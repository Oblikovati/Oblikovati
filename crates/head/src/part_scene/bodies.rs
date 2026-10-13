// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::wire::BodyInfo;
use oblikovati_model::{assets::AssignmentReference, identity::DocumentGuid};
use oblikovati_session::DocumentSession;
impl DesktopPartScenes {
    pub(super) fn body_infos<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
        &self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<Vec<BodyInfo>, String> {
        let workspace = files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let evaluated = self.current(session)?;
        evaluated
            .ordered_bodies()
            .enumerate()
            .map(|(index, body)| {
                let info = native_body_info(session, guid, index, body)?;
                with_display_metadata(files, document, info)
            })
            .collect()
    }
}
fn native_body_info<O: TransactionObserver>(
    session: &DocumentSession<O>,
    guid: DocumentGuid,
    index: usize,
    body: &NamedPrismSnapshot,
) -> Result<BodyInfo, String> {
    let encoded = body.body_reference_key(session.history(), guid)?;
    let report = body.body_report(session.history(), guid, encoded)?;
    let key: String = encoded.iter().map(|byte| format!("{byte:02x}")).collect();
    let reference = AssignmentReference::new(&key, vec![])?;
    let material_id = session
        .snapshot()
        .assets
        .assignments
        .effective_material_id(&reference)
        .unwrap_or_default()
        .into();
    let mut info = topology_info(&report.topology)?;
    info.index = api_count("body index", index)?;
    info.name = format!("Solid{}", index + 1);
    info.key = key;
    info.material_id = material_id;
    Ok(info)
}
fn topology_info(
    topology: &oblikovati_martensite::PrismTopologySummary,
) -> Result<BodyInfo, String> {
    Ok(BodyInfo {
        solid: topology.solid,
        visible: true,
        faces: api_count("faces", topology.faces)?,
        edges: api_count("edges", topology.edges)?,
        vertices: api_count("vertices", topology.vertices)?,
        shells: api_count("shells", topology.shells)?,
        wires: api_count("wires", topology.wires)?,
        ..Default::default()
    })
}
fn api_count(label: &str, count: usize) -> Result<i64, String> {
    i64::try_from(count).map_err(|_| {
        format!("body {label} {count}: expected a nonnegative signed 64-bit API count")
    })
}

fn with_display_metadata<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    document: u64,
    mut info: BodyInfo,
) -> Result<BodyInfo, String> {
    let (name, style) = files
        .body_display_metadata(document, &info.key)
        .map_err(|cause| cause.to_string())?;
    if let Some(name) = name {
        info.name = name.into();
    }
    info.style = style.unwrap_or_default().into();
    Ok(info)
}
