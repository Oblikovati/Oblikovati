// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::contract::Material;
pub(super) fn request<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    encoded: &[u8],
) -> Result<BodyPhysicalPropertiesArgs, Error> {
    let mut request: BodyPhysicalPropertiesArgs =
        crate::decode(METHOD_BODY_PHYSICAL_PROPS, encoded)?;
    if !["", "low", "medium", "high"].contains(&request.accuracy.as_str()) {
        return Err(Error(format!(
            "body property accuracy {:?}: expected low, medium, high, or empty",
            request.accuracy
        )));
    }
    if !request.density_g_cm3.is_finite() {
        return Err(Error(format!(
            "body density {}: expected finite g/cm³",
            request.density_g_cm3
        )));
    }
    if request.density_g_cm3 == 0. {
        request.density_g_cm3 = part_density(files)?;
    }
    if request.density_g_cm3 <= 0. {
        request.density_g_cm3 = 1.;
    }
    Ok(request)
}
fn part_density<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
) -> Result<f64, Error> {
    let document = crate::part_owner::active_part(files)?;
    let session = files
        .workspace()
        .session(document)
        .map_err(|cause| Error(cause.to_string()))?;
    let Some(id) = session.snapshot().assets.assignments.part_material_id() else {
        return Ok(1.);
    };
    crate::assets::with_scoped_assets(files, |materials, _| {
        Ok(materials.resolve(id).map_err(Error)?.density())
    })
}
