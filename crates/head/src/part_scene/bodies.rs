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
/// What the browser's folder shows of one solid.
pub(crate) struct BodyRow {
    pub feature: oblikovati_model::feature::FeatureId,
    pub name: String,
    pub key: String,
    /// Where the API's body list counts it.
    pub index: i64,
    pub faces: usize,
}
impl DesktopPartScenes {
    /// Each solid and the feature that made it, for the browser's folder.
    /// Example: `scenes.body_rows(files, document)?` reads Extrude1 as `Solid1`.
    pub(crate) fn body_rows<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
        &self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<Vec<BodyRow>, String> {
        let workspace = files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let evaluated = self.current(session)?;
        let key_of = |body: &NamedSolid| {
            body.body_reference_key(session.history(), guid)
                .map(body_reference_text)
        };
        let ordered: Vec<String> = evaluated
            .ordered_bodies()
            .map(&key_of)
            .collect::<Result<_, _>>()?;
        let mut rows = Vec::new();
        for record in evaluated.health() {
            let Some(body) = evaluated.body(record.feature) else {
                continue;
            };
            let key = key_of(body)?;
            let index = ordered
                .iter()
                .position(|known| *known == key)
                .unwrap_or_default();
            let (name, _) = files
                .body_display_metadata(document, &key)
                .map_err(|cause| cause.to_string())?;
            let encoded = body.body_reference_key(session.history(), guid)?;
            let faces = body
                .topology_summary(session.history(), guid, encoded)?
                .faces;
            rows.push(BodyRow {
                feature: record.feature,
                faces,
                name: name.map_or_else(|| format!("Solid{}", index + 1), Into::into),
                key,
                index: api_count("body index", index)?,
            });
        }
        Ok(rows)
    }
}
fn native_body_info<O: TransactionObserver>(
    session: &DocumentSession<O>,
    guid: DocumentGuid,
    index: usize,
    body: &NamedSolid,
) -> Result<BodyInfo, String> {
    let encoded = body.body_reference_key(session.history(), guid)?;
    let topology = body.topology_summary(session.history(), guid, encoded)?;
    let key = body_reference_text(encoded);
    let reference = AssignmentReference::new(&key, vec![])?;
    let material_id = session
        .snapshot()
        .assets
        .assignments
        .effective_material_id(&reference)
        .unwrap_or_default()
        .into();
    let mut info = topology_info(&topology)?;
    info.index = api_count("body index", index)?;
    info.name = format!("Solid{}", index + 1);
    info.visible = session.body_visible(&key);
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

pub(super) fn body_reference_text(encoded: &[u8]) -> String {
    encoded.iter().map(|byte| format!("{byte:02x}")).collect()
}
