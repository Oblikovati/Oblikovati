// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::history::{DocumentEditHistory, RecipeReservations};
impl RecipeReservations<'_> {
    /// Reserve a dimension name outside undo state; example: `ids.allocate_dimension_name(sketch, &snapshot.parameters)?`.
    pub fn allocate_dimension_name(
        &mut self,
        sketch: DocumentSketchId,
        parameters: &ParameterCollection<u64>,
    ) -> Result<String, RecipeIdError> {
        self.dimension_names
            .allocate(sketch, self.sketch_ids.high_water(), parameters)
    }
}
impl DocumentEditHistory {
    pub(in crate::history) fn restore_dimension_name_marks(
        &mut self,
        supplied: &BTreeMap<u64, u64>,
    ) -> Result<(), RecipeIdError> {
        for (sketch, minimum) in self.dimension_names.marks() {
            let mark = supplied.get(&sketch).copied().unwrap_or(0);
            if mark < minimum {
                return Err(RecipeIdError::DimensionNameMark {
                    sketch,
                    mark,
                    minimum,
                });
            }
        }
        self.dimension_names = SketchDimensionNameReservations::restore(
            &self.current.sketches,
            &self.current.parameters,
            self.sketch_ids.high_water(),
            Some(supplied),
        )?;
        Ok(())
    }
}
