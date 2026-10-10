// SPDX-License-Identifier: GPL-2.0-only

use std::{collections::BTreeSet, error::Error, fmt};

/// Monotonic IDs for one document-owned recipe namespace, with zero reserved.
/// Keep this owner outside undo snapshots: removing an entity must not rewind it.
#[derive(Debug, Default)]
pub struct RecipeIdAllocator {
    high_water: u64,
}

/// Invalid persisted allocation state or an exhausted identity namespace.
#[derive(Clone, Debug, Eq, PartialEq)]
pub enum RecipeIdError {
    ZeroId,
    DerivedMark { mark: i64, minimum: i64 },
    DuplicateId(u64),
    BeyondHighWater { id: u64, high_water: u64 },
    Exhausted,
}

impl RecipeIdAllocator {
    /// Restore a namespace, preserving the saved mark even after entity deletion.
    /// A legacy file without a mark can reserve only the IDs still present.
    ///
    /// ```
    /// use oblikovati_model::identity::RecipeIdAllocator;
    /// let mut ids = RecipeIdAllocator::restore(Some(10), &[2, 7])?;
    /// assert_eq!(ids.allocate()?, 11);
    /// # Ok::<(), oblikovati_model::identity::RecipeIdError>(())
    /// ```
    pub fn restore(saved_high_water: Option<u64>, live_ids: &[u64]) -> Result<Self, RecipeIdError> {
        let mut seen = BTreeSet::new();
        let mut largest = 0;
        for &id in live_ids {
            validate_restored_id(id, saved_high_water, &mut seen)?;
            largest = largest.max(id);
        }
        Ok(Self {
            high_water: saved_high_water.unwrap_or(largest),
        })
    }

    /// Reserve the next ID permanently, including when a later edit fails.
    ///
    /// ```
    /// use oblikovati_model::identity::RecipeIdAllocator;
    /// let mut ids = RecipeIdAllocator::default();
    /// assert_eq!(ids.allocate()?, 1);
    /// # Ok::<(), oblikovati_model::identity::RecipeIdError>(())
    /// ```
    pub fn allocate(&mut self) -> Result<u64, RecipeIdError> {
        let next = self
            .high_water
            .checked_add(1)
            .ok_or(RecipeIdError::Exhausted)?;
        self.high_water = next;
        Ok(next)
    }

    /// Return the reservation mark that storage must persist with the recipe.
    ///
    /// ```
    /// use oblikovati_model::identity::RecipeIdAllocator;
    /// assert_eq!(RecipeIdAllocator::default().high_water(), 0);
    /// ```
    pub fn high_water(&self) -> u64 {
        self.high_water
    }
}

fn validate_restored_id(
    id: u64,
    high_water: Option<u64>,
    seen: &mut BTreeSet<u64>,
) -> Result<(), RecipeIdError> {
    if id == 0 {
        return Err(RecipeIdError::ZeroId);
    }
    if !seen.insert(id) {
        return Err(RecipeIdError::DuplicateId(id));
    }
    if let Some(high_water) = high_water
        && id > high_water
    {
        return Err(RecipeIdError::BeyondHighWater { id, high_water });
    }
    Ok(())
}

impl fmt::Display for RecipeIdError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::DerivedMark { mark, minimum } => write!(
                formatter,
                "derived table reservation mark {mark}: expected at least {minimum}"
            ),
            Self::ZeroId => write!(formatter, "recipe ID 0: expected a nonzero ID"),
            Self::DuplicateId(id) => write!(
                formatter,
                "recipe ID {id}: expected one occurrence per namespace"
            ),
            Self::BeyondHighWater { id, high_water } => write!(
                formatter,
                "recipe ID {id}: expected at most saved high-water mark {high_water}"
            ),
            Self::Exhausted => write!(
                formatter,
                "recipe high-water mark {}: expected a value below u64::MAX",
                u64::MAX
            ),
        }
    }
}

impl Error for RecipeIdError {}
