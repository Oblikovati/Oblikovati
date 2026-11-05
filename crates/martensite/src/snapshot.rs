// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    ReleaseProbeProperties, analytic_probe_properties, build_probe, require_extents,
    require_valid_probe,
};
use rustycad_tkbrep::topods::Shape;
use rustycad_tktopalgo::brep_builder_api::{BRepBuilderAPICopy, BRepBuilderAPIMakeShape};
use std::{marker::PhantomData, rc::Rc};
/// Retained release-probe B-rep with detached topology and geometry.
/// This diagnostic boundary stays on its owner thread; full feature validity and naming remain open.
/// ```compile_fail
/// fn require_send<T: Send>() {}
/// require_send::<oblikovati_martensite::ReleaseProbeSnapshot>();
/// ```
pub struct ReleaseProbeSnapshot {
    shape: Shape,
    extents: [f64; 3],
    owner: PhantomData<Rc<()>>,
}
/// Retain an isolated release-probe result in working units.
/// Example: `let snapshot = retain_box_release([2., 3., 4.])?;` then query `snapshot.properties()?`.
pub fn retain_box_release(extents: [f64; 3]) -> Result<ReleaseProbeSnapshot, String> {
    require_extents(extents)?;
    let source = build_probe(extents)?;
    ReleaseProbeSnapshot::capture(&source, extents)
}
impl ReleaseProbeSnapshot {
    fn capture(source: &Shape, extents: [f64; 3]) -> Result<Self, String> {
        if source.is_null() {
            return Err(format!(
                "probe snapshot {extents:?}: expected a non-null B-rep source"
            ));
        }
        require_valid_probe(source, extents)?;
        let shape = detached_shape(source, extents)?;
        require_valid_probe(&shape, extents)?;
        analytic_probe_properties(&shape, extents)?;
        Ok(Self {
            shape,
            extents,
            owner: PhantomData,
        })
    }
    /// Query the analytic B-rep without reading a mesh; example: `snapshot.properties()?`.
    pub fn properties(&self) -> Result<ReleaseProbeProperties, String> {
        analytic_probe_properties(&self.shape, self.extents)
    }
    /// Create another detached snapshot; example: `let isolated = snapshot.detached()?;`.
    pub fn detached(&self) -> Result<Self, String> {
        Self::capture(&self.shape, self.extents)
    }
}
fn detached_shape(source: &Shape, extents: [f64; 3]) -> Result<Shape, String> {
    let context = format!("probe copy {extents:?}");
    let mut copier = copied_kernel_shape(source, &context)?;
    copier
        .shape()
        .cloned()
        .map_err(|error| format!("{context}: expected copied B-rep: {error}"))
}
pub(crate) fn copied_kernel_shape(
    source: &Shape,
    context: &str,
) -> Result<BRepBuilderAPICopy, String> {
    crate::checked_shape::copy_shape(source, context).map_err(|cause| cause.to_string())
}
#[cfg(test)]
#[path = "../tests/snapshot.rs"]
mod tests;
