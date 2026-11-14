// SPDX-License-Identifier: GPL-2.0-only
//! Per-document default sketch names follow Go's monotonic session sequence.
use oblikovati_model::{document_sketch::DocumentSketches, identity::RecipeIdError};

pub(crate) fn next_name(
    start: u64,
    sketches: &DocumentSketches,
) -> Result<(String, u64), RecipeIdError> {
    let mut ordinal = start;
    loop {
        let following = ordinal.checked_add(1).ok_or(RecipeIdError::Exhausted)?;
        let name = format!("Sketch{ordinal}");
        if sketches
            .sketches()
            .iter()
            .all(|sketch| sketch.name() != name)
        {
            return Ok((name, following));
        }
        ordinal = following;
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn exhausted_sequence_declines_without_wrapping() {
        assert!(matches!(
            next_name(u64::MAX, &DocumentSketches::default()),
            Err(RecipeIdError::Exhausted)
        ));
    }
}
