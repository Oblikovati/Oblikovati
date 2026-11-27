// SPDX-License-Identifier: GPL-2.0-only
//! Route drawing creation through the current source part owner.
use super::*;

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileContext<'_, O, W, F> {
    pub(super) fn drawing_view_command(
        &mut self,
        method: &str,
        request: &[u8],
    ) -> Result<Vec<u8>, Error> {
        if method == METHOD_DRAWING_SET_MODEL_REFERENCE {
            let result = crate::drawing_reference::apply(self.files, decode(method, request)?)?;
            self.synchronize_drawing_references();
            return Ok(result);
        }
        let replay: Option<&mut dyn crate::PartUpdateAccess<O, W, F>> =
            match self.part_update.as_deref_mut() {
                Some(owner) => Some(owner),
                None => self
                    .part_topology
                    .as_deref_mut()
                    .map(|owner| owner as &mut dyn crate::PartUpdateAccess<O, W, F>),
            };
        if crate::drawing_annotations::ANNOTATION_METHODS.contains(&method) {
            return crate::drawing_annotations::apply(
                self.files,
                self.drawing_views,
                replay,
                method,
                request,
            );
        }
        if method == METHOD_DRAWING_VIEWS_ADD_BASE {
            let result = crate::drawing_views::add_base(
                self.files,
                self.drawing_views,
                replay,
                decode(method, request)?,
            )?;
            self.synchronize_drawing_references();
            return Ok(result);
        }
        crate::drawing_views::collection::apply(
            self.files,
            self.drawing_views,
            replay,
            method,
            request,
        )
    }
}
