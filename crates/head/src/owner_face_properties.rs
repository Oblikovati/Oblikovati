// SPDX-License-Identifier: GPL-2.0-only
//! Generation-scoped analytic face properties for the desktop.
use super::super::*;
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(in crate::owner) fn open_face_properties(
        &mut self,
        document: u64,
        key: &str,
    ) -> Result<bool, String> {
        let face = self.checked_selected_named_face(document, key)?;
        let generation = self.part_inspection_generation(document)?;
        let area =
            self.part_scenes
                .face_area_mm2(&self.files, document, face.feature, &face.key)?;
        self.face_properties = Some((
            document,
            generation,
            oblikovati_gui::GuiFaceProperties {
                key: key.into(),
                area_mm2: area,
            },
        ));
        Ok(true)
    }
    pub(in crate::owner) fn close_face_properties(&mut self) -> Result<bool, String> {
        Ok(self.face_properties.take().is_some())
    }
    pub(in crate::owner) fn gui_face_properties(
        &self,
    ) -> Option<oblikovati_gui::GuiFaceProperties> {
        let (document, generation, properties) = self.face_properties.as_ref()?;
        if self.files.workspace().active_document() != Some(*document) {
            return None;
        }
        let session = self.files.workspace().session(*document).ok()?;
        (session.generation() == *generation).then(|| properties.clone())
    }
    pub(in crate::owner) fn retire_stale_face_properties(&mut self) -> bool {
        if self.face_properties.is_some() && self.gui_face_properties().is_none() {
            self.face_properties = None;
            return true;
        }
        false
    }
}
