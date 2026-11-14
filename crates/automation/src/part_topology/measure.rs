// SPDX-License-Identifier: GPL-2.0-only
use super::*;

pub(super) fn query<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    owner: &mut dyn PartTopologyAccess<O, W, F>,
    document: u64,
    encoded: &[u8],
) -> Result<Vec<u8>, Error> {
    let request: MeasureArgs = crate::decode(METHOD_ANALYSIS_MEASURE, encoded)?;
    if request.type_ != "area" {
        return Err(Error(format!(
            "measurement type {:?}: expected supported type \"area\"",
            request.type_
        )));
    }
    let key = crate::reference_bytes::hex_bytes(&request.key_a)?;
    let area = owner
        .face_area(files, document, request.body_index, &key)
        .map_err(Error)?
        * 100.;
    if !area.is_finite() || area <= 0. {
        return Err(Error(format!(
            "face area {area}: expected finite positive mm²"
        )));
    }
    crate::encode(&MeasureResult {
        type_: request.type_,
        value: area,
        unit: "mm²".into(),
    })
}
