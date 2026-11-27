// SPDX-License-Identifier: GPL-2.0-only
//! Per-feature work-plane display output and local presentation declines.
use oblikovati_model::feature::FeatureId;
use oblikovati_viewport::ViewportDatumPlane;

/// Current reference overlays and failures. A decline does not hide healthy planes.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct HeadDatumPlaneUpdate {
    pub planes: Vec<ViewportDatumPlane>,
    pub declines: Vec<(FeatureId, String)>,
}
