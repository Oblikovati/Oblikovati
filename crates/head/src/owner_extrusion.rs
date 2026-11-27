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
        let snapshot = session.snapshot();
        let evaluated = oblikovati_automation::evaluated_extrude_distance(snapshot, distance);
        let engineering = evaluated.as_ref().ok().map(|&length| {
            let parameters = &snapshot.parameters;
            use oblikovati_model::quantity::{Dimension, Quantity};
            oblikovati_gui::GuiParameterEngineering {
                nominal: Quantity::new(length, Dimension::Length),
                units: std::sync::Arc::new(parameters.units().clone()),
                default_precision: parameters.settings().linear_dimension_precision.clamp(0, 8)
                    as u8,
            }
        });
        let error = evaluated
            .and_then(|_| {
                second_distance.map_or(Ok(()), |distance| {
                    oblikovati_automation::validate_extrude_distance(snapshot, distance)
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
                engineering,
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
    pub(super) fn extrusion_draft_request(
        &self,
        draft: &oblikovati_gui::GuiExtrudePreview,
    ) -> Result<oblikovati_api::wire::featureargs::Extrude, String> {
        use oblikovati_gui::GuiExtrudeReferenceExtent::{FromTo, ThroughAll, ToFace};
        if matches!(draft.reference_extent, None | Some(ThroughAll)) {
            let mut request = self.extrusion_request(
                draft.document,
                draft.sketch,
                draft.profile,
                &draft.distance,
                draft.direction,
                draft.second_distance.as_deref(),
            )?;
            if let Some(operation) = draft.operation {
                request.operation = operation.to_string().to_ascii_lowercase();
            }
            if draft.reference_extent.is_some() {
                request.extent = "through-all".into();
            }
            return Ok(request);
        }
        let (extent, from, to) = match &draft.reference_extent {
            Some(ToFace { to }) => ("to-face", "", to),
            Some(FromTo { from, to }) => ("from-to", from.as_str(), to),
            _ => return Err("extrusion extent: expected a face reference".into()),
        };
        Ok(oblikovati_api::wire::featureargs::Extrude {
            sketch_index: self.extrusion_sketch_index(draft.document, draft.sketch)?,
            profile_index: draft.profile,
            extent: extent.into(),
            from_face: from.into(),
            to_face: to.clone(),
            ..Default::default()
        })
    }
    pub(super) fn commit_extrusion_draft(
        &mut self,
        draft: &oblikovati_gui::GuiExtrudePreview,
        body_name: Option<&str>,
    ) -> Result<bool, String> {
        let session = self
            .files
            .workspace()
            .session(draft.document)
            .map_err(|cause| cause.to_string())?;
        if self.extrusion_draft_preview.as_ref() != Some(draft)
            || draft.generation != Some(session.generation())
        {
            return Err("extrusion draft: expected its current certified preview".into());
        }
        let request = self.extrusion_draft_request(draft)?;
        self.clear_extrusion_preview();
        let client = oblikovati_api::client::Client::new(self.document_host());
        match body_name {
            Some(name) => create_named_solid(&client, &request, name),
            None => client.features().add_feature(&request).map(|_| ()),
        }
        .map_err(|cause| cause.to_string())?;
        Ok(true)
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

fn create_named_solid<T: oblikovati_api::client::Caller>(
    client: &oblikovati_api::client::Client<T>,
    request: &oblikovati_api::wire::featureargs::Extrude,
    name: &str,
) -> Result<(), oblikovati_api::Error> {
    use oblikovati_api::wire::TransactionBeginArgs;
    let before = current_solid_bodies(client)?;
    let transactions = client.transactions();
    transactions.begin(&TransactionBeginArgs {
        label: "Create extrusion".into(),
    })?;
    match extrude_and_name(client, request, name, &before) {
        Ok(()) => transactions.end().map(|_| ()),
        Err(cause) => match transactions.abort() {
            Ok(_) => Err(cause),
            Err(abort) => Err(oblikovati_api::Error(format!(
                "{cause}. Aborting the extrusion also failed: {abort}"
            ))),
        },
    }
}

fn current_solid_bodies<T: oblikovati_api::client::Caller>(
    client: &oblikovati_api::client::Client<T>,
) -> Result<Vec<oblikovati_api::wire::BodyInfo>, oblikovati_api::Error> {
    client
        .body()
        .list(&oblikovati_api::json::EmptyObject {})
        .map(|result| result.bodies.unwrap_or_default())
}

fn extrude_and_name<T: oblikovati_api::client::Caller>(
    client: &oblikovati_api::client::Client<T>,
    request: &oblikovati_api::wire::featureargs::Extrude,
    name: &str,
    before: &[oblikovati_api::wire::BodyInfo],
) -> Result<(), oblikovati_api::Error> {
    client.features().add_feature(request)?;
    let after = current_solid_bodies(client)?;
    let mut added = after
        .iter()
        .filter(|body| before.iter().all(|original| original.key != body.key));
    let (Some(body), None) = (added.next(), added.next()) else {
        return Err(oblikovati_api::Error(format!(
            "extrude Body Name {name:?}: expected one new solid"
        )));
    };
    client
        .body()
        .rename(&oblikovati_api::wire::BodyRenameArgs {
            body_index: body.index,
            name: name.into(),
        })?;
    Ok(())
}
