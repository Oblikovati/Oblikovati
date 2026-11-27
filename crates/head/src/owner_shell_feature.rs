// SPDX-License-Identifier: GPL-2.0-only
//! Preview and commit a Shell from current named face selections (#4106).
use super::*;
use oblikovati_api::wire::featureargs::Shell;
use oblikovati_gui::GuiShellFeatureRequest;

struct ShellInputs {
    body: Vec<u8>,
    faces: Vec<Vec<u8>>,
    request: Shell,
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
    fn shell_input(&self, request: &GuiShellFeatureRequest) -> Result<ShellInputs, String> {
        self.require_browser_document(request.document)?;
        let workspace = self.files.workspace();
        let session = workspace
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        let picks = self.current_face_appearance_picks(request.document);
        let keys: Vec<_> = picks.iter().map(|pick| pick.key.clone()).collect();
        if keys.is_empty() || keys != request.faces || session.generation() != request.generation {
            return Err(format!(
                "shell faces {:?}, generation {}: expected the current nonempty selection",
                request.faces, request.generation
            ));
        }
        let guid = workspace
            .identity(request.document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let evaluation = self.part_scenes.current(session)?;
        let bodies = evaluation
            .ordered_bodies()
            .filter(|body| !body.is_surface())
            .map(|body| {
                Ok(super::owner_shell_face_body::ShellBodyFaces {
                    body: body.body_reference_key(session.history(), guid)?.to_vec(),
                    faces: body.face_reference_keys(session.history(), guid)?,
                })
            })
            .collect::<Result<Vec<_>, String>>()?;
        let faces: Vec<_> = picks.iter().map(|pick| pick.face.key.clone()).collect();
        let body_key = super::owner_shell_face_body::unique_shell_face_body(&faces, &bodies)?;
        let input = Shell {
            face_refs: Some(keys),
            thickness: request.thickness.clone(),
            ..Default::default()
        };
        Ok(ShellInputs {
            body: body_key,
            faces,
            request: input,
        })
    }
    pub(in crate::owner) fn preview_selected_shell(
        &mut self,
        request: &GuiShellFeatureRequest,
    ) -> Result<bool, String> {
        self.clear_extrusion_preview();
        let input = self.shell_input(request)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        let prepared = oblikovati_automation::prepare_shell(
            session.snapshot(),
            input.body,
            input.faces,
            &input.request,
        )
        .map_err(|cause| cause.to_string())?;
        self.preview_selected_blend(request.document, |candidate, ids| {
            prepared.create(candidate, ids)
        })
    }
    pub(in crate::owner) fn commit_selected_shell(
        &mut self,
        request: &GuiShellFeatureRequest,
    ) -> Result<bool, String> {
        let input = self.shell_input(request)?;
        oblikovati_api::client::Client::new(self.document_host())
            .features()
            .add_feature(&input.request)
            .map_err(|cause| cause.to_string())?;
        self.clear_extrusion_preview();
        Ok(true)
    }
}
