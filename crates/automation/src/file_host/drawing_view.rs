// SPDX-License-Identifier: GPL-2.0-only
//! Route drawing creation through the current source part owner.
use super::*;

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileContext<'_, O, W, F> {
    pub(super) fn add_drawing_base(&mut self, request: AddBaseViewArgs) -> Result<Vec<u8>, Error> {
        let replay: Option<&mut dyn crate::PartUpdateAccess<O, W, F>> =
            match self.part_update.as_deref_mut() {
                Some(owner) => Some(owner),
                None => self
                    .part_topology
                    .as_deref_mut()
                    .map(|owner| owner as &mut dyn crate::PartUpdateAccess<O, W, F>),
            };
        crate::drawing_views::add_base(self.files, self.drawing_views, replay, request)
    }
}
