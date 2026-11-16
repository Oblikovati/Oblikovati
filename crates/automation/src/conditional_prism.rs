// SPDX-License-Identifier: GPL-2.0-only
//! Compose conditional prism input evidence without authorizing a feature.
use oblikovati_martensite::{
    ConditionalProfileLine, ConditionalProfileVertex, conditional_profile_mapping,
};
use oblikovati_model::{
    feature::{FeatureId, part::PartFeatureDefinition},
    history::{ConditionalPointDisplacement, DocumentSnapshot},
    sketch::ConditionalSketchRoot,
};
use oblikovati_session::{
    ConditionalPrismAxis, PrismPrecisionObligation, SessionConditionalSketch, TransactionObserver,
};
use std::collections::BTreeSet;

/// A single-root assessment held under the source session borrow, not a certificate.
///
/// ```compile_fail
/// use oblikovati_automation::conditional_prism::conditional_prism_inputs;
/// use oblikovati_model::{document_sketch::DocumentSketchId, feature::FeatureId, sketch::SketchEntityId};
/// use oblikovati_session::{DocumentSession, TransactionObserver};
/// fn no_edit<O: TransactionObserver>(session: &mut DocumentSession<O>, bounds: &[[f64; 2]]) {
///     let sketch = session.conditional_canonical_sketch(DocumentSketchId(1), SketchEntityId(1)).unwrap();
///     let inputs = conditional_prism_inputs(&sketch, FeatureId(1), bounds, [Some([0., 0.]), Some([0., 0.]), None]).unwrap();
///     session.undo().unwrap();
///     let _ = inputs.vertices();
/// }
/// ```
pub struct ConditionalPrismInputs<'e, 's, O> {
    _sketch: &'e SessionConditionalSketch<'s, O>,
    feature: FeatureId,
    root: ConditionalSketchRoot,
    lines: Vec<ConditionalProfileLine>,
    vertices: Vec<ConditionalProfileVertex>,
    swept: Vec<ConditionalPointDisplacement>,
    axes: [ConditionalPrismAxis; 3],
    obligations: Vec<PrismPrecisionObligation>,
}
impl<O> ConditionalPrismInputs<'_, '_, O> {
    /// Inspect the feature identity; example: `result.feature()`.
    pub fn feature(&self) -> FeatureId {
        self.feature
    }
    /// Inspect the single conditional root; example: `result.root()`.
    pub fn root(&self) -> &ConditionalSketchRoot {
        &self.root
    }
    /// Inspect all converted normal lines; example: `result.lines()`.
    pub fn lines(&self) -> &[ConditionalProfileLine] {
        &self.lines
    }
    /// Inspect converted vertices and their source bounds; example: `result.vertices()`.
    pub fn vertices(&self) -> &[ConditionalProfileVertex] {
        &self.vertices
    }
    /// Inspect swept bounds for authored points; example: `result.swept_points()`.
    pub fn swept_points(&self) -> &[ConditionalPointDisplacement] {
        &self.swept
    }
    /// Inspect displacement-source assumptions; example: `result.axes()`.
    pub fn axes(&self) -> &[ConditionalPrismAxis; 3] {
        &self.axes
    }
    /// Inspect the unresolved admission obligations; example: `result.obligations()`.
    pub fn obligations(&self) -> &[PrismPrecisionObligation] {
        &self.obligations
    }
}

/// Assess one prism from one session-owned root without admitting a kernel operation.
/// Example: `conditional_prism_inputs(&sketch, feature, &bounds, constants)?`.
pub fn conditional_prism_inputs<'e, 's, O: TransactionObserver>(
    sketch: &'e SessionConditionalSketch<'s, O>,
    feature: FeatureId,
    bounds: &[[f64; 2]],
    constants: [Option<[f64; 2]>; 3],
) -> Result<ConditionalPrismInputs<'e, 's, O>, String> {
    let projection = sketch.projection()?;
    let root = projection.conditional_root_auto(bounds, &[])?;
    let mapping = conditional_profile_mapping(&root)?;
    let displacement = sketch.conditional_prism_displacement(feature, constants)?;
    require_converted_selection(projection.snapshot(), feature, mapping.lines())?;
    let swept = displacement.swept_points_with_root(&root)?;
    let obligations = sketch.prism_precision_obligations_with_root(feature, &root)?;
    Ok(ConditionalPrismInputs {
        _sketch: sketch,
        feature,
        root: root.root().clone(),
        lines: mapping.lines().to_vec(),
        vertices: mapping.vertices().to_vec(),
        swept: swept.points().to_vec(),
        axes: std::array::from_fn(|axis| displacement.axes()[axis].clone()),
        obligations,
    })
}

fn require_converted_selection(
    snapshot: &DocumentSnapshot,
    feature: FeatureId,
    lines: &[ConditionalProfileLine],
) -> Result<(), String> {
    let authored = snapshot
        .work_geometry
        .get(feature)
        .map_err(|error| error.to_string())?;
    let PartFeatureDefinition::SketchPrism(prism) = authored.definition() else {
        return Err(format!(
            "feature {feature:?}: expected an authored sketch prism"
        ));
    };
    let mut seen = BTreeSet::new();
    for edge in std::iter::once(&prism.selection().outer)
        .chain(&prism.selection().holes)
        .flatten()
    {
        if !seen.insert(edge.curve)
            || lines.iter().filter(|line| line.curve == edge.curve).count() != 1
        {
            return Err(format!(
                "feature {feature:?} curve {:?}: expected one distinct converted profile line",
                edge.curve
            ));
        }
    }
    Ok(())
}

#[cfg(test)]
mod tests;
