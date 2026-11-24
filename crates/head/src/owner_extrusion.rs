// SPDX-License-Identifier: GPL-2.0-only
//! Apply desktop extrusion requests through the public typed feature command.
use super::*;
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn validate_extrusion_distance_from_gui(
        &mut self,
        document: u64,
        distance: &str,
        second_distance: Option<&str>,
    ) -> Result<bool, String> {
        if self.files.workspace().active_document() != Some(document) {
            return Err(format!(
                "extrude document {document}: expected the active document"
            ));
        }
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let error = oblikovati_automation::validate_extrude_distance(session.snapshot(), distance)
            .and_then(|()| {
                second_distance.map_or(Ok(()), |distance| {
                    oblikovati_automation::validate_extrude_distance(session.snapshot(), distance)
                        .map_err(|cause| oblikovati_api::Error(format!("Distance B: {cause}")))
                })
            })
            .err()
            .map(|cause| cause.to_string());
        self.extrusion_distance_validation = Some((
            session.generation(),
            oblikovati_gui::GuiExtrudeDistanceValidation {
                generation: session.generation(),
                document,
                distance: distance.into(),
                second_distance: second_distance.map(str::to_owned),
                error,
            },
        ));
        Ok(true)
    }
    pub(super) fn create_extrusion_from_gui(
        &mut self,
        document: u64,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
        profile: i64,
        distance: &str,
        direction: oblikovati_gui::GuiExtrudeDirection,
        second_distance: Option<&str>,
    ) -> Result<bool, String> {
        self.clear_extrusion_preview();
        let request = self.extrusion_request(
            document,
            sketch,
            profile,
            distance,
            direction,
            second_distance,
        )?;
        oblikovati_api::client::Client::new(self.document_host())
            .features()
            .add_feature(&request)
            .map_err(|cause| cause.to_string())?;
        Ok(true)
    }
    pub(super) fn extrusion_request(
        &self,
        document: u64,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
        profile: i64,
        distance: &str,
        direction: oblikovati_gui::GuiExtrudeDirection,
        second_distance: Option<&str>,
    ) -> Result<oblikovati_api::wire::featureargs::Extrude, String> {
        use oblikovati_gui::GuiExtrudeDirection::*;
        if (direction == Asymmetric) != second_distance.is_some()
            || second_distance.is_some_and(|value| value.trim().is_empty())
        {
            return Err(format!(
                "extrusion {direction:?}: expected Distance B only in Asymmetric mode"
            ));
        }
        Ok(oblikovati_api::wire::featureargs::Extrude {
            sketch_index: self.extrusion_sketch_index(document, sketch)?,
            profile_index: profile,
            distance: distance.into(),
            second_distance: second_distance.unwrap_or_default().into(),
            direction: match direction {
                Positive | Asymmetric => "",
                Negative => "negative",
                Symmetric => "symmetric",
            }
            .into(),
            ..Default::default()
        })
    }
    fn extrusion_sketch_index(
        &self,
        document: u64,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
    ) -> Result<i64, String> {
        let workspace = self.files.workspace();
        if workspace.active_document() != Some(document) {
            return Err(format!(
                "extrude document {document}: expected the active document"
            ));
        }
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let index = session
            .snapshot()
            .sketches
            .sketches()
            .iter()
            .position(|row| row.id() == sketch)
            .ok_or_else(|| {
                format!("extrude sketch {sketch:?}: expected a sketch in document {document}")
            })?;
        i64::try_from(index)
            .map_err(|_| format!("sketch index {index}: expected a signed API index"))
    }
}
