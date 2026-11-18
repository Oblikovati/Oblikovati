// SPDX-License-Identifier: GPL-2.0-only
//! Dispatch typed blends through the document's current replay owner.
use super::*;

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileContext<'_, O, W, F> {
    pub(super) fn add_blend(&mut self, args: AddFeatureArgs) -> Result<Vec<u8>, Error> {
        let replay: Option<&mut dyn crate::PartUpdateAccess<O, W, F>> =
            match self.part_update.as_deref_mut() {
                Some(owner) => Some(owner),
                None => self
                    .part_topology
                    .as_deref_mut()
                    .map(|owner| owner as &mut dyn crate::PartUpdateAccess<O, W, F>),
            };
        if args.kind == featureargs::KIND_FILLET {
            return crate::fillet::apply_wire(self.files, self.fillet, replay, args);
        }
        crate::chamfer::apply(self.files, self.fillet, replay, args)
    }
}
