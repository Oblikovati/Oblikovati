// SPDX-License-Identifier: GPL-2.0-only
//! Preview and commit straight holes from a current face selection (#4093).
use super::*;
use oblikovati_api::wire::featureargs::Hole;
use oblikovati_gui::GuiHoleRequest;

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    fn hole_input(&self, request: &GuiHoleRequest) -> Result<(Vec<u8>, Vec<u8>, Hole), String> {
        self.require_browser_document(request.document)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        let face = self.only_selected_named_face(request.document, &request.face)?;
        let face_hex: String = face.key.iter().map(|byte| format!("{byte:02x}")).collect();
        if session.generation() != request.generation || face_hex != request.face {
            return Err(format!(
                "hole face {:?}, generation {}: expected the current selection",
                request.face, request.generation
            ));
        }
        let guid = self
            .files
            .workspace()
            .identity(request.document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let body_key = self.part_scenes.current(session)?.inspect_body_reference(
            face.feature,
            &face.key,
            |body| {
                body.face_topology(session.history(), guid, &face.key)?;
                body.body_reference_key(session.history(), guid)
                    .map(<[u8]>::to_vec)
            },
        )?;
        let input = Hole {
            face_ref: face_hex,
            diameter: request.diameter.clone(),
            depth: request.depth.clone(),
            ..Default::default()
        };
        Ok((body_key, face.key, input))
    }
    pub(in crate::owner) fn preview_selected_hole(
        &mut self,
        request: &GuiHoleRequest,
    ) -> Result<bool, String> {
        let (body, face, input) = self.hole_input(request)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        let prepared = oblikovati_automation::prepare_hole(session.snapshot(), body, face, &input)
            .map_err(|cause| cause.to_string())?;
        self.preview_selected_blend(request.document, |candidate, ids| {
            prepared.create(candidate, ids)
        })
    }
    pub(in crate::owner) fn commit_selected_hole(
        &mut self,
        request: &GuiHoleRequest,
    ) -> Result<bool, String> {
        let (_, _, input) = self.hole_input(request)?;
        oblikovati_api::client::Client::new(self.document_host())
            .features()
            .add_feature(&input)
            .map_err(|cause| cause.to_string())?;
        self.clear_extrusion_preview();
        Ok(true)
    }
}
