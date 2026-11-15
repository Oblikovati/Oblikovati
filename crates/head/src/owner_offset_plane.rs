// SPDX-License-Identifier: GPL-2.0-only
//! Create offset datums through the typed work-plane API.
use super::*;
use oblikovati_gui::{GuiOffsetPlaneBase, GuiOriginPlaneKind};

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn publish_offset_plane_validation(
        &mut self,
        document: u64,
        offset: &str,
    ) -> Result<bool, String> {
        self.require_browser_document(document)?;
        let error = self.validate_plane_offset(document, offset).err();
        let generation = self
            .files
            .workspace()
            .session(document)
            .map_err(|e| e.to_string())?
            .generation();
        self.offset_plane_validation = Some(oblikovati_gui::GuiOffsetPlaneValidation {
            generation,
            document,
            offset: offset.to_owned(),
            error,
        });
        Ok(true)
    }
    pub(super) fn create_offset_plane_from_selection(
        &mut self,
        document: u64,
        base: GuiOffsetPlaneBase,
        offset: &str,
    ) -> Result<bool, String> {
        self.require_browser_document(document)?;
        self.validate_plane_offset(document, offset)?;
        let reference = self.offset_plane_reference(document, base)?;
        let request = oblikovati_api::wire::CreateWorkPlaneArgs {
            kind: oblikovati_api::types::WORK_PLANE_OFFSET.0.into_owned(),
            refs: Some(vec![reference]),
            offset: offset.to_owned(),
            ..Default::default()
        };
        oblikovati_api::client::Client::new(self.document_host())
            .work_planes()
            .create(&request)
            .map_err(|error| error.to_string())?;
        Ok(true)
    }
    fn validate_plane_offset(&self, document: u64, offset: &str) -> Result<(), String> {
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|e| e.to_string())?;
        let value = session
            .snapshot()
            .parameters
            .evaluate_scalar_expression(offset, oblikovati_model::quantity::Dimension::Length)
            .map_err(|e| e.to_string())?;
        if value.value == 0.0 {
            return Err(format!("offset {offset:?}: expected a nonzero length"));
        }
        Ok(())
    }
    fn offset_plane_reference(
        &mut self,
        document: u64,
        base: GuiOffsetPlaneBase,
    ) -> Result<String, String> {
        match base {
            GuiOffsetPlaneBase::Origin(plane) => Ok(origin_reference(plane).into()),
            GuiOffsetPlaneBase::WorkPlane(id) => self.selected_plane_reference(document, id),
            GuiOffsetPlaneBase::SelectedFace => {
                let face = self
                    .chosen_model_face(document)
                    .ok_or("offset plane: expected a selected face")?;
                self.part_scenes.selected_face_reference(
                    &mut self.files,
                    document,
                    face.feature,
                    face.role,
                )
            }
        }
    }
    fn selected_plane_reference(
        &mut self,
        document: u64,
        id: oblikovati_model::feature::FeatureId,
    ) -> Result<String, String> {
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|e| e.to_string())?;
        let name = session
            .snapshot()
            .work_geometry
            .get(id)
            .map_err(|e| e.to_string())?
            .name()
            .to_owned();
        let planes = oblikovati_api::client::Client::new(self.document_host())
            .work_planes()
            .list()
            .map_err(|e| e.to_string())?;
        let matching: Vec<_> = planes
            .planes
            .unwrap_or_default()
            .into_iter()
            .filter(|plane| plane.name == name && !plane.is_origin)
            .collect();
        let [plane] = matching.as_slice() else {
            return Err(format!(
                "work plane {id:?} named {name:?}: expected exactly one API reference"
            ));
        };
        Ok(plane.ref_.clone())
    }
}
fn origin_reference(plane: GuiOriginPlaneKind) -> &'static str {
    use oblikovati_api::types::{WORK_REF_XY_PLANE, WORK_REF_XZ_PLANE, WORK_REF_YZ_PLANE};
    match plane {
        GuiOriginPlaneKind::Xy => WORK_REF_XY_PLANE,
        GuiOriginPlaneKind::Xz => WORK_REF_XZ_PLANE,
        GuiOriginPlaneKind::Yz => WORK_REF_YZ_PLANE,
    }
}
