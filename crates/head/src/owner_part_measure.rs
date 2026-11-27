// SPDX-License-Identifier: GPL-2.0-only
//! Measure named solid faces and edges using analytic Martensite reports.
use super::*;
use oblikovati_gui::GuiMeasurePanel;
use oblikovati_model::feature::FeatureId;

const SELECT_GEOMETRY: &str = "Select a solid face or edge to measure";

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
        vec![SELECT_GEOMETRY.into()]
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
        self.part_inspection_generation(document)?;
        let rows = self.preselected_measure_rows(document)?;
        self.set_part_measure(document, rows)
    }
    pub(super) fn measure_selected_face(
        &mut self,
        document: u64,
        key: &str,
    ) -> Result<bool, String> {
        self.checked_selected_named_face(document, key)?;
        self.start_part_measure()?;
        Ok(true)
    }
    fn preselected_measure_rows(&self, document: u64) -> Result<Vec<String>, String> {
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        if !session.selected_browser_bodies().is_empty() {
            return Ok(vec![SELECT_GEOMETRY.into()]);
        }
        match (
            self.current_face_appearance_picks(document),
            self.chosen_model_edges(document),
        ) {
            ([pick], []) => {
                self.part_measure_face_rows(document, Some((pick.face.feature, &pick.face.key)))
            }
            ([], [_, ..]) if self.chosen_model_face(document).is_none() => {
                self.part_measure_edge_rows(document)
            }
            _ => Ok(vec![SELECT_GEOMETRY.into()]),
        }
    }
    pub(super) fn part_inspection_generation(&self, document: u64) -> Result<u64, String> {
        let kind = self
            .files
            .document_type(document)
            .map_err(|cause| cause.to_string())?;
        if kind != oblikovati_api::types::DOCUMENT_PART {
            return Err(format!(
                "document {document} type {kind:?}: expected a part for inspection"
            ));
        }
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        if session.in_transaction() {
            return Err(format!(
                "document {document}: expected no active transaction for inspection"
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
        if !self.measuring_part(document) {
            return Ok(());
        }
        let selected = picked
            .as_ref()
            .map(|(feature, key)| (*feature, key.as_slice()));
        let rows = self.part_measure_face_rows(document, selected)?;
        self.set_part_measure(document, rows)
    }
    pub(super) fn measuring_part(&self, document: u64) -> bool {
        self.part_measure
            .as_ref()
            .is_some_and(|tool| tool.document == document)
    }
    pub(super) fn measure_part_edges(&mut self, document: u64) -> Result<(), String> {
        if !self.measuring_part(document) {
            return Ok(());
        }
        let rows = self.part_measure_edge_rows(document)?;
        self.set_part_measure(document, rows)
    }
    fn set_part_measure(&mut self, document: u64, rows: Vec<String>) -> Result<(), String> {
        let generation = self.part_inspection_generation(document)?;
        self.part_measure = Some(PartMeasure {
            document,
            generation,
            rows,
        });
        Ok(())
    }
    fn part_measure_face_rows(
        &self,
        document: u64,
        picked: Option<(FeatureId, &[u8])>,
    ) -> Result<Vec<String>, String> {
        let Some((feature, face)) = picked else {
            return Ok(vec![SELECT_GEOMETRY.into()]);
        };
        let area = self
            .part_scenes
            .face_area_mm2(&self.files, document, feature, face)?;
        let mut rows = vec!["Selection 1: Face".into(), format!("Area {area:.3} mm²")];
        if let Some(radius) =
            self.part_scenes
                .face_cylinder_radius_mm(&self.files, document, feature, face)?
        {
            rows.push(format!("Diameter {:.3} mm", radius * 2.));
            rows.push(format!("Radius {radius:.3} mm"));
        }
        Ok(rows)
    }
    fn part_measure_edge_rows(&self, document: u64) -> Result<Vec<String>, String> {
        let edges = self.chosen_model_edges(document);
        if edges.is_empty() {
            return Ok(vec![SELECT_GEOMETRY.into()]);
        }
        let mut rows = Vec::with_capacity(edges.len() * 2);
        for (index, edge) in edges.iter().enumerate() {
            let length =
                self.part_scenes
                    .edge_length_mm(&self.files, document, edge.feature, &edge.key)?;
            rows.push(format!("Selection {}: Edge", index + 1));
            rows.push(format!("Length {length:.3} mm"));
        }
        if edges.len() == 1 {
            rows.push("Ctrl-click another edge to add its length".into());
        }
        Ok(rows)
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
        assert_eq!(measure.rows_for_generation(5), [SELECT_GEOMETRY]);
    }
}

#[path = "owner_face_properties.rs"]
mod face_properties;
