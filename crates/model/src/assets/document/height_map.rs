// SPDX-License-Identifier: GPL-2.0-only
//! A portable grayscale height map for an appearance, independent of its source file.

/// Downsampled height data embedded in the owning Part recipe.
#[derive(Clone, Debug, PartialEq, Eq, serde::Deserialize, serde::Serialize)]
pub struct AppearanceHeightMap {
    name: String,
    samples: Vec<u8>,
}

impl AppearanceHeightMap {
    pub const SIDE: usize = 32;

    /// Construct an embedded map from a file name and 32×32 row-major samples.
    /// For example, `AppearanceHeightMap::new("bump.png".into(), vec![0; 1024])`.
    pub fn new(name: String, samples: Vec<u8>) -> Result<Self, String> {
        let map = Self { name, samples };
        map.validate()?;
        Ok(map)
    }

    /// Show the source file name in the editor, such as `map.name()`.
    pub fn name(&self) -> &str {
        &self.name
    }

    /// Read the stored samples in row-major order, such as `map.samples()[0]`.
    pub fn samples(&self) -> &[u8] {
        &self.samples
    }

    pub(super) fn validate(&self) -> Result<(), String> {
        if self.name.is_empty() || self.name.contains(['/', '\\', '\0']) {
            return Err(format!(
                "height map name {:?}: expected a plain file name",
                self.name
            ));
        }
        if self.samples.len() != Self::SIDE * Self::SIDE {
            return Err(format!(
                "height map {:?} has {} samples: expected {}",
                self.name,
                self.samples.len(),
                Self::SIDE * Self::SIDE
            ));
        }
        Ok(())
    }
}
