// SPDX-License-Identifier: GPL-2.0-only
use super::{CopyIdentities, PreparedSelectionCopy, RecipeIdError, SketchEntityId, SketchError};
use std::{error::Error, fmt};
/// Selection copies in input order and complete source/copy relations, including defining points.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SketchCopyReport {
    pub copies: Vec<SketchEntityId>,
    pub relations: Vec<(SketchEntityId, SketchEntityId)>,
}
/// Preserve invalid geometry and exhausted permanent reservations as distinct typed causes.
#[derive(Clone, Debug, PartialEq)]
pub enum SketchCopyError {
    Sketch(SketchError),
    Identity(RecipeIdError),
}
impl fmt::Display for SketchCopyError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Sketch(cause) => write!(formatter, "sketch selection copy: {cause}"),
            Self::Identity(cause) => write!(formatter, "sketch copy identity: {cause}"),
        }
    }
}
impl Error for SketchCopyError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Sketch(cause) => Some(cause),
            Self::Identity(cause) => Some(cause),
        }
    }
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
