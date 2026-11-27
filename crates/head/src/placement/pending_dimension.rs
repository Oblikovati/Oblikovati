// SPDX-License-Identifier: GPL-2.0-only
//! Native placement data kept while the user chooses a dimension expression.
use oblikovati_model::{
    history::{SketchLineChainCreation, SketchPlacementFieldLock},
    sketch::SketchPlacementRecipe,
};

#[derive(Clone)]
pub(crate) enum PendingDimensionPlacement {
    Recipe {
        recipe: SketchPlacementRecipe,
        locks: Vec<Option<SketchPlacementFieldLock>>,
    },
    Line {
        input: SketchLineChainCreation,
        pixel: Option<[i32; 2]>,
    },
}

#[derive(Clone)]
pub(crate) struct PendingDimensionChoice {
    pub id: u64,
    pub index: usize,
    pub placement: PendingDimensionPlacement,
}
