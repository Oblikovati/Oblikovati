// SPDX-License-Identifier: GPL-2.0-only
//! Analytic area of a face selected from the current evaluated body.
use super::*;
use oblikovati_model::feature::FeatureId;

impl DesktopPartScenes {
    /// Read the selected face's exact area; example: `scenes.face_area_mm2(files, document, feature, role)?`.
    pub(crate) fn face_area_mm2<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
        &self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
        feature: FeatureId,
        face: &[u8],
    ) -> Result<f64, String> {
        let workspace = files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let evaluated = self.current(session)?;
        let body = evaluated
            .body(feature)
            .ok_or_else(|| format!("feature {feature:?}: expected a current solid"))?;
        face_area(body.face_area(session.history(), guid, face)?)
    }
}
fn face_area(area: f64) -> Result<f64, String> {
    let area = area * 100.;
    if !area.is_finite() || area <= 0. {
        return Err(format!("face area {area:?}: expected finite positive mm²"));
    }
    Ok(area)
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn area_conversion_requires_one_finite_positive_face() {
        assert_eq!(face_area(2.5).unwrap(), 250.);
        assert!(face_area(f64::NAN).is_err());
        assert!(face_area(0.).is_err());
    }
}
