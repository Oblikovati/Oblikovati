// SPDX-License-Identifier: GPL-2.0-only
//! Measure a selected solid face using its analytic Martensite report.
use super::*;
use oblikovati_gui::GuiMeasurePanel;
use oblikovati_model::feature::FeatureId;

const SELECT_FACE: &str = "Select a solid face to measure its area";

pub(super) struct PartMeasure {
    pub(super) document: u64,
    generation: u64,
    rows: Vec<String>,
}
impl PartMeasure {
    fn rows_for_generation(&self, generation: u64) -> Vec<String> {
        if self.generation == generation {
            return self.rows.clone();
        }
        vec![SELECT_FACE.into()]
    }
}
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn start_part_measure(&mut self) -> Result<(), String> {
        let document = self
            .files
            .workspace()
            .active_document()
            .ok_or("Measure: expected an active part")?;
        let generation = self.part_measure_generation(document)?;
        self.part_measure = Some(PartMeasure {
            document,
            generation,
            rows: vec![SELECT_FACE.into()],
        });
        Ok(())
    }
    fn part_measure_generation(&self, document: u64) -> Result<u64, String> {
        let kind = self
            .files
            .document_type(document)
            .map_err(|cause| cause.to_string())?;
        if kind != oblikovati_api::types::DOCUMENT_PART {
            return Err(format!(
                "document {document} type {kind:?}: expected a part for Measure"
            ));
        }
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        if session.in_transaction() {
            return Err(format!(
                "document {document}: expected no active transaction for Measure"
            ));
        }
        Ok(session.generation())
    }
    pub(super) fn gui_part_measure(&self) -> Option<GuiMeasurePanel> {
        let measure = self.part_measure.as_ref()?;
        if self.files.workspace().active_document() != Some(measure.document)
            || self.active_sketch_edit().is_some()
        {
            return None;
        }
        let session = self.files.workspace().session(measure.document).ok()?;
        Some(GuiMeasurePanel {
            rows: measure.rows_for_generation(session.generation()),
        })
    }
    pub(super) fn measure_part_face(
        &mut self,
        document: u64,
        picked: Option<(FeatureId, Vec<u8>)>,
    ) -> Result<(), String> {
        if self
            .part_measure
            .as_ref()
            .is_none_or(|measure| measure.document != document)
        {
            return Ok(());
        }
        let rows = self.part_measure_rows(document, picked)?;
        let generation = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .generation();
        self.part_measure = Some(PartMeasure {
            document,
            generation,
            rows,
        });
        Ok(())
    }
    fn part_measure_rows(
        &self,
        document: u64,
        picked: Option<(FeatureId, Vec<u8>)>,
    ) -> Result<Vec<String>, String> {
        let Some((feature, face)) = picked else {
            return Ok(vec![SELECT_FACE.into()]);
        };
        let area = self
            .part_scenes
            .face_area_mm2(&self.files, document, feature, &face)?;
        Ok(vec![
            "Selection 1: Face".into(),
            format!("Area {area:.3} mm²"),
        ])
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn changed_model_generation_discards_stale_measurement_rows() {
        let measure = PartMeasure {
            document: 1,
            generation: 4,
            rows: vec!["Area 100.000 mm²".into()],
        };
        assert_eq!(measure.rows_for_generation(4), measure.rows);
        assert_eq!(measure.rows_for_generation(5), [SELECT_FACE]);
    }
}
