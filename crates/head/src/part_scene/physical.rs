// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::wire::MassPropertiesResult;
use oblikovati_martensite::AnalyticBodyProperties;
impl DesktopPartScenes {
    pub(super) fn body_physical_properties<
        O: TransactionObserver,
        W: WorkspaceObserver,
        F: DocumentFiles,
    >(
        &self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
        index: Option<i64>,
        density: f64,
    ) -> Result<MassPropertiesResult, String> {
        let workspace = files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let evaluation = self.current(session)?;
        let bodies: Vec<_> = evaluation.ordered_bodies().collect();
        if let Some(index) = index {
            let body = selected_body(&bodies, index)?;
            let key = body.body_reference_key(session.history(), guid)?;
            return converted_properties(
                body.physical_properties(session.history(), guid, key)?,
                density,
            );
        }
        match oblikovati_martensite::named_part_properties(session.history(), guid, bodies)? {
            Some(properties) => converted_properties(properties, density),
            None => Ok(empty_properties(density)),
        }
    }
}
fn converted_properties(
    properties: AnalyticBodyProperties,
    density: f64,
) -> Result<MassPropertiesResult, String> {
    // Database lengths are cm; the wire contract reports mm, grams and g·mm².
    let [cx, cy, cz] = properties
        .geometry
        .centroid
        .map(|coordinate| coordinate * 10.);
    let tensor = properties
        .inertia
        .map(|row| row.map(|value| value * density * 100.));
    let result = MassPropertiesResult {
        volume_mm3: properties.geometry.volume * 1000.,
        surface_area_mm2: properties.area * 100.,
        mass_g: properties.geometry.volume * density,
        density_g_cm3: density,
        centroid_x_mm: cx,
        centroid_y_mm: cy,
        centroid_z_mm: cz,
        inertia_xx_gmm2: tensor[0][0],
        inertia_yy_gmm2: tensor[1][1],
        inertia_zz_gmm2: tensor[2][2],
        inertia_xy_gmm2: tensor[0][1],
        inertia_yz_gmm2: tensor[1][2],
        inertia_zx_gmm2: tensor[2][0],
        principal_moments_gmm2: properties
            .principal_moments
            .map(|value| value * density * 100.),
        principal_axes: properties.principal_axes,
    };
    finite_conversion(&result)?;
    Ok(result)
}
fn finite_conversion(result: &MassPropertiesResult) -> Result<(), String> {
    let values = [
        result.volume_mm3,
        result.surface_area_mm2,
        result.mass_g,
        result.density_g_cm3,
        result.centroid_x_mm,
        result.centroid_y_mm,
        result.centroid_z_mm,
        result.inertia_xx_gmm2,
        result.inertia_yy_gmm2,
        result.inertia_zz_gmm2,
        result.inertia_xy_gmm2,
        result.inertia_yz_gmm2,
        result.inertia_zx_gmm2,
    ];
    if values
        .into_iter()
        .chain(result.principal_moments_gmm2)
        .any(|value| !value.is_finite())
    {
        return Err(format!(
            "converted body properties {result:?}: expected finite API units"
        ));
    }
    Ok(())
}

fn selected_body<'a>(
    bodies: &[&'a NamedPrismSnapshot],
    index: i64,
) -> Result<&'a NamedPrismSnapshot, String> {
    usize::try_from(index)
        .ok()
        .and_then(|index| bodies.get(index))
        .copied()
        .ok_or_else(|| format!("body index {index}: expected 0 <= index < {}", bodies.len()))
}
fn empty_properties(density: f64) -> MassPropertiesResult {
    MassPropertiesResult {
        density_g_cm3: density,
        principal_axes: [[1., 0., 0.], [0., 1., 0.], [0., 0., 1.]],
        ..Default::default()
    }
}
