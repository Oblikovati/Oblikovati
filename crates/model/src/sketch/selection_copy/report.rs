// SPDX-License-Identifier: GPL-2.0-only
use super::{CopyIdentities, PreparedSelectionCopy, SketchEntityId, SketchError};
pub use crate::sketch::SketchEntityEditError as SketchCopyError;
/// Selection copies in input order and complete source/copy relations, including defining points.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SketchCopyReport {
    pub copies: Vec<SketchEntityId>,
    pub relations: Vec<(SketchEntityId, SketchEntityId)>,
}

impl PreparedSelectionCopy {
    pub(super) fn report(
        &self,
        identities: &CopyIdentities,
    ) -> Result<SketchCopyReport, SketchError> {
        let copies = self
            .selected
            .iter()
            .map(|id| identities.get(*id))
            .collect::<Result<_, _>>()?;
        Ok(SketchCopyReport {
            copies,
            relations: identities.relations(&self.sources)?,
        })
    }
}
