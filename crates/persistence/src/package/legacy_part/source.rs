// SPDX-License-Identifier: GPL-2.0-only
use serde::Deserialize;

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
pub(super) struct LegacyFeature {
    pub kind: String,
    pub name: String,
    #[serde(default)]
    pub seq: u64,
    #[serde(default)]
    pub suppressed: bool,
    pub extrude: LegacyExtrude,
}
#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
pub(super) struct LegacyExtrude {
    pub sketch: usize,
    #[serde(default)]
    pub profile: Option<usize>,
    #[serde(default)]
    pub profiles: Vec<usize>,
    pub operation: String,
    #[serde(default)]
    pub extent: String,
    #[serde(default)]
    pub direction: String,
    pub distance: f64,
}
