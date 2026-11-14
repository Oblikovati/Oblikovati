// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::contract::Material;
pub(super) fn request<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    encoded: &[u8],
) -> Result<BodyPhysicalPropertiesArgs, Error> {
    let mut request: BodyPhysicalPropertiesArgs =
        crate::decode(METHOD_BODY_PHYSICAL_PROPS, encoded)?;
    request.density_g_cm3 = resolved_density(files, request.density_g_cm3, &request.accuracy)?;
    Ok(request)
}
pub(super) fn mass_request<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    encoded: &[u8],
) -> Result<MassPropertiesArgs, Error> {
    let mut request: MassPropertiesArgs = crate::decode(METHOD_ANALYSIS_MASS_PROPERTIES, encoded)?;
    request.density_g_cm3 = resolved_density(files, request.density_g_cm3, &request.accuracy)?;
    Ok(request)
}
fn resolved_density<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    mut density: f64,
    accuracy: &str,
) -> Result<f64, Error> {
    if !["", "low", "medium", "high"].contains(&accuracy) {
        return Err(Error(format!(
            "property accuracy {accuracy:?}: expected low, medium, high, or empty"
        )));
    }
    if !density.is_finite() {
        return Err(Error(format!("density {density}: expected finite g/cm³")));
    }
    if density == 0. {
        density = part_density(files)?;
    }
    Ok(if density <= 0. { 1. } else { density })
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
