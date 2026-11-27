// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{contract::Material, types::PhysicalProperties};

pub(super) fn model_density<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    encoded: &[u8],
) -> Result<f64, Error> {
    super::no_arguments(METHOD_MODEL_PHYSICAL_PROPERTIES, encoded)?;
    part_density(files)
}

pub(super) fn model_properties(
    properties: MassPropertiesResult,
) -> Result<PhysicalProperties, Error> {
    let result = PhysicalProperties {
        mass: properties.mass_g,
        volume: properties.volume_mm3 / 1000.,
        area: properties.surface_area_mm2 / 100.,
        density: properties.density_g_cm3,
        centroid: [
            properties.centroid_x_mm / 10.,
            properties.centroid_y_mm / 10.,
            properties.centroid_z_mm / 10.,
        ],
    };
    require_finite_model_properties(&result)?;
    Ok(result)
}
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

fn require_finite_model_properties(properties: &PhysicalProperties) -> Result<(), Error> {
    let values = [
        properties.mass,
        properties.volume,
        properties.area,
        properties.density,
        properties.centroid[0],
        properties.centroid[1],
        properties.centroid[2],
    ];
    if values.into_iter().all(f64::is_finite) {
        return Ok(());
    }
    Err(Error(format!(
        "model physical properties {properties:?}: expected finite API values"
    )))
}
