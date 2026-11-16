// SPDX-License-Identifier: GPL-2.0-only
use serde::Deserialize;

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields, rename_all = "camelCase")]
pub(super) struct LegacySketch {
    pub id: u64,
    pub name: String,
    #[serde(default)]
    pub seq: u64,
    pub plane: LegacyPlane,
    #[serde(default)]
    pub points: Vec<LegacyPoint>,
    #[serde(default)]
    pub entities: Vec<LegacyEntity>,
    #[serde(default)]
    pub constraints: Vec<LegacyConstraint>,
    #[serde(default)]
    pub dimensions: Vec<LegacyDimension>,
}
#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields, rename_all = "camelCase")]
pub(super) struct LegacyPlane {
    pub origin: [f64; 3],
    pub x_axis: [f64; 3],
    pub y_axis: [f64; 3],
}
#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
pub(super) struct LegacyPoint {
    pub id: u64,
    pub x: f64,
    pub y: f64,
    #[serde(default)]
    pub standalone: bool,
}
#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
pub(super) struct LegacyEntity {
    pub id: u64,
    pub kind: String,
    pub points: Vec<u64>,
    #[serde(default)]
    pub construction: bool,
    #[serde(default)]
    pub centerline: bool,
}
#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
pub(super) struct LegacyConstraint {
    pub kind: String,
    pub points: Vec<u64>,
}
#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
pub(super) struct LegacyDimension {
    pub kind: String,
    pub points: Vec<u64>,
    pub expression: String,
    #[serde(default)]
    pub orientation: String,
}
