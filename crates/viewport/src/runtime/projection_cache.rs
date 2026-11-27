// SPDX-License-Identifier: GPL-2.0-only
//! Bounded reuse of exact point batches for one viewport camera and surface (#4035).
use oblikovati_model::sketch::SketchPlane;
use std::collections::VecDeque;

// Retain point, curve-witness and glyph batches during sketch interaction (#4035).
// Bound retained coordinate and pixel storage to 24 MiB across three entries.
const CAPACITY: usize = 3;
const MAX_BATCH_BYTES: usize = 8 * 1024 * 1024;
const MAX_POINTS: usize =
    MAX_BATCH_BYTES / (std::mem::size_of::<[f64; 2]>() + std::mem::size_of::<[i32; 2]>());

#[derive(Clone, Copy)]
pub(super) struct ProjectionKey {
    pub(super) plane: SketchPlane,
    pub(super) camera: u64,
    pub(super) surface: u64,
}

struct ProjectionBatch {
    key: ProjectionKey,
    points: Vec<[f64; 2]>,
    pixels: Vec<[i32; 2]>,
}

#[derive(Default)]
pub(super) struct ProjectionCache {
    entries: VecDeque<ProjectionBatch>,
}

impl ProjectionCache {
    pub(super) fn clear(&mut self) {
        self.entries.clear();
    }
    #[cfg(test)]
    pub(super) fn retained(&self) -> usize {
        self.entries.len()
    }
    pub(super) fn get(&mut self, key: ProjectionKey, points: &[[f64; 2]]) -> Option<Vec<[i32; 2]>> {
        let index = self
            .entries
            .iter()
            .position(|entry| entry.matches(key, points))?;
        let entry = self.entries.remove(index)?;
        let pixels = entry.pixels.clone();
        self.entries.push_back(entry);
        Some(pixels)
    }
    pub(super) fn store(&mut self, key: ProjectionKey, points: &[[f64; 2]], pixels: &[[i32; 2]]) {
        if points.len() > MAX_POINTS || points.len() != pixels.len() {
            return;
        }
        if self.entries.len() == CAPACITY {
            self.entries.pop_front();
        }
        self.entries.push_back(ProjectionBatch {
            key,
            points: points.to_vec(),
            pixels: pixels.to_vec(),
        });
    }
}

impl ProjectionBatch {
    fn matches(&self, key: ProjectionKey, points: &[[f64; 2]]) -> bool {
        self.key.camera == key.camera
            && self.key.surface == key.surface
            && self.key.plane.same_recipe(key.plane)
            && same_point_bits(&self.points, points)
    }
}

fn same_point_bits(left: &[[f64; 2]], right: &[[f64; 2]]) -> bool {
    left.len() == right.len()
        && left
            .iter()
            .zip(right)
            .all(|(a, b)| a[0].to_bits() == b[0].to_bits() && a[1].to_bits() == b[1].to_bits())
}

#[cfg(test)]
#[path = "../../tests/projection_cache.rs"]
mod tests;
