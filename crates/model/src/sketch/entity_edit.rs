// SPDX-License-Identifier: GPL-2.0-only
use super::{Sketch2D, SketchEntityId, SketchError};
use crate::identity::RecipeIdError;
use std::{collections::BTreeSet, error::Error, fmt};
/// Preserve invalid geometry and exhausted permanent reservations as distinct typed causes.
#[derive(Clone, Debug, PartialEq)]
pub enum SketchEntityEditError {
    Sketch(SketchError),
    Identity(RecipeIdError),
}
impl fmt::Display for SketchEntityEditError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Sketch(cause) => write!(formatter, "sketch entity edit: {cause}"),
            Self::Identity(cause) => write!(formatter, "sketch entity identity: {cause}"),
        }
    }
}
impl Error for SketchEntityEditError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Sketch(cause) => Some(cause),
            Self::Identity(cause) => Some(cause),
        }
    }
}

impl Sketch2D {
    pub(super) fn reserve_edit_identities(
        &self,
        count: usize,
        reserve: &mut impl FnMut() -> Result<SketchEntityId, RecipeIdError>,
    ) -> Result<Vec<SketchEntityId>, SketchEntityEditError> {
        let mut unique = BTreeSet::new();
        let mut reserved = Vec::with_capacity(count);
        for _ in 0..count {
            let id = reserve().map_err(SketchEntityEditError::Identity)?;
            self.validate_edit_identity(id, &mut unique)
                .map_err(SketchEntityEditError::Sketch)?;
            reserved.push(id);
        }
        Ok(reserved)
    }
    fn validate_edit_identity(
        &self,
        id: SketchEntityId,
        unique: &mut BTreeSet<SketchEntityId>,
    ) -> Result<(), SketchError> {
        self.require_new_entity(id)?;
        if !unique.insert(id) {
            return Err(SketchError::Definition(format!(
                "edit identity {id:?}: expected a unique new entity ID"
            )));
        }
        Ok(())
    }
    pub(super) fn copy_entity_roles(
        &mut self,
        source: &Self,
        original: SketchEntityId,
        created: SketchEntityId,
    ) -> Result<(), SketchError> {
        if source.construction_curves.contains(&original) {
            self.set_curve_construction(&[created], true)?;
        }
        if source.centerlines.contains(&original) {
            self.set_line_centerline(created, true)?;
        }
        Ok(())
    }
}
