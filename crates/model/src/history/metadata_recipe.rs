// SPDX-License-Identifier: GPL-2.0-only
use super::DocumentSnapshot;
impl DocumentSnapshot {
    /// Compare non-geometric recipe state for history, refresh and save guards.
    /// Example: `before.same_metadata_recipe(&after)` cannot omit a newly authored asset assignment.
    pub fn same_metadata_recipe(&self, other: &Self) -> bool {
        self.properties.same_recipe(&other.properties)
            && self.assets == other.assets
            && self.attachments == other.attachments
            && self.sketch_settings == other.sketch_settings
            && self.resources == other.resources
            && self.interests == other.interests
            && self.attributes.same_recipe(&other.attributes)
    }
}
