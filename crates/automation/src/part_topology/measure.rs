// SPDX-License-Identifier: GPL-2.0-only
use super::*;

pub(super) fn query<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    owner: &mut dyn PartTopologyAccess<O, W, F>,
    document: u64,
    encoded: &[u8],
) -> Result<Vec<u8>, Error> {
    let request: MeasureArgs = crate::decode(METHOD_ANALYSIS_MEASURE, encoded)?;
    if !matches!(request.type_.as_str(), "area" | "length") {
        return Err(Error(format!(
            "measurement type {:?}: expected supported type \"area\" or \"length\"",
            request.type_
        )));
    }
    let key = crate::reference_bytes::hex_bytes(&request.key_a)?;
    let (value, unit) = if request.type_ == "area" {
        (
            owner
                .face_area(files, document, request.body_index, &key)
                .map_err(Error)?
                * 100.,
            "mm²",
        )
    } else {
        (
            owner
                .edge_length(files, document, request.body_index, &key)
                .map_err(Error)?
                * 10.,
            "mm",
        )
    };
    if !value.is_finite() || value <= 0. {
        return Err(Error(format!(
            "measurement {} {value}: expected finite positive {unit}",
            request.type_
        )));
    }
    crate::encode(&MeasureResult {
        type_: request.type_,
        value,
        unit: unit.into(),
    })
}
