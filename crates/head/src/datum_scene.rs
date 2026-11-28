// SPDX-License-Identifier: GPL-2.0-only
//! Per-feature work-plane display output and local presentation declines.
use oblikovati_model::feature::FeatureId;
use oblikovati_viewport::{ViewportDatumAxis, ViewportDatumPlane};

/// One evaluated user work axis prepared for frame-level projection.
#[derive(Clone, Debug, PartialEq)]
pub struct HeadWorkAxisSegment {
    pub id: FeatureId,
    pub name: String,
    pub endpoints: [[f64; 3]; 2],
    pub selected: bool,
}

/// Current reference overlays and failures. A decline does not hide healthy planes.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct HeadDatumPlaneUpdate {
    pub planes: Vec<ViewportDatumPlane>,
    pub axes: Vec<ViewportDatumAxis>,
    pub declines: Vec<(FeatureId, String)>,
}
