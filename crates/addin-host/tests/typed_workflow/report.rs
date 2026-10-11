// SPDX-License-Identifier: GPL-2.0-only
#[derive(serde::Deserialize)]
pub(super) struct WorkflowReport {
    pub(super) created: u64,
    pub(super) reopened: u64,
    #[serde(rename = "radiusBits")]
    pub(super) radius_bits: u64,
    pub(super) parameter: String,
}
