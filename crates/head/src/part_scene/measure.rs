// SPDX-License-Identifier: GPL-2.0-only
//! Analytic area of a face selected from the current evaluated body.
use super::*;
use oblikovati_martensite::{PrismFaceProbe, PrismFaceRole};
use oblikovati_model::feature::FeatureId;

impl DesktopPartScenes {
    /// Read the selected face's exact area; example: `scenes.face_area_mm2(files, document, feature, role)?`.
    pub(crate) fn face_area_mm2<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
        &self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
        feature: FeatureId,
        role: PrismFaceRole,
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
        let key = body.body_reference_key(session.history(), guid)?;
        face_area(&body.body_report(session.history(), guid, key)?.faces, role)
    }
}
fn face_area(report: &[PrismFaceProbe], role: PrismFaceRole) -> Result<f64, String> {
    let mut faces = report.iter().filter(|face| face.role == role);
    let face = faces
        .next()
        .ok_or_else(|| format!("face {role:?}: expected a current face"))?;
    if faces.next().is_some() {
        return Err(format!("face {role:?}: expected a unique face"));
    }
    let area = face.area * 100.;
    if !area.is_finite() || area <= 0. {
        return Err(format!(
            "face {role:?} area {area:?}: expected finite positive mm²"
        ));
    }
    Ok(area)
}

#[cfg(test)]
mod tests {
    use super::*;
    fn face(area: f64) -> PrismFaceProbe {
        PrismFaceProbe {
            role: PrismFaceRole::Start,
            area,
            centroid: [0.; 3],
            loops: 1,
        }
    }
    #[test]
    fn area_conversion_requires_one_finite_positive_face() {
        assert_eq!(face_area(&[face(2.5)], PrismFaceRole::Start).unwrap(), 250.);
        assert!(face_area(&[], PrismFaceRole::Start).is_err());
        assert!(face_area(&[face(1.), face(2.)], PrismFaceRole::Start).is_err());
        assert!(face_area(&[face(f64::NAN)], PrismFaceRole::Start).is_err());
        assert!(face_area(&[face(0.)], PrismFaceRole::Start).is_err());
    }
}
