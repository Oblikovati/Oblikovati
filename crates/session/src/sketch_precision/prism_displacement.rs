// SPDX-License-Identifier: GPL-2.0-only
//! Session-bound source enclosures; they do not authorize a prism operation.
use super::SessionConditionalSketch;
use crate::TransactionObserver;
use oblikovati_model::{
    feature::{
        FeatureId, part::PartFeatureDefinition, sketch_prism::SketchPrismDefinition,
        work_geometry::DatumLength,
    },
    history::ConditionalProjectedSketchRoot,
    sketch::{ConditionalParameterBound, ConditionalSketchRoot, conditional_parameter_bound},
};

/// One local displacement component with its source and conditional bounds.
#[derive(Clone)]
pub struct ConditionalPrismAxis {
    source: DatumLength,
    bounds: [f64; 2],
    parameter: Option<ConditionalParameterBound>,
}
impl ConditionalPrismAxis {
    /// Inspect the authored source; example: `axis.source()`.
    pub fn source(&self) -> DatumLength {
        self.source
    }
    /// Inspect the conditional bounds in working units; example: `axis.bounds()`.
    pub fn bounds(&self) -> [f64; 2] {
        self.bounds
    }
    /// Inspect parameter provenance when present; example: `axis.parameter()`.
    pub fn parameter(&self) -> Option<&ConditionalParameterBound> {
        self.parameter.as_ref()
    }
}

/// Source bounds retained under the owning session borrow, not a feature certificate.
///
/// ```compile_fail
/// use oblikovati_session::{DocumentSession, TransactionObserver};
/// use oblikovati_model::{document_sketch::DocumentSketchId, feature::FeatureId, sketch::SketchEntityId};
/// fn no_mutation<O: TransactionObserver>(session: &mut DocumentSession<O>) {
///     let sketch = session.conditional_canonical_sketch(DocumentSketchId(1), SketchEntityId(1)).unwrap();
///     let bounds = sketch.conditional_prism_displacement(FeatureId(1), [Some([0., 0.]), Some([0., 0.]), None]).unwrap();
///     session.undo().unwrap();
///     let _ = bounds.axes();
/// }
/// ```
pub struct ConditionalPrismDisplacement<'e, 's, O> {
    _evidence: &'e SessionConditionalSketch<'s, O>,
    feature: FeatureId,
    axes: [ConditionalPrismAxis; 3],
}
impl<O> ConditionalPrismDisplacement<'_, '_, O> {
    /// Inspect the feature identity; example: `result.feature()`.
    pub fn feature(&self) -> FeatureId {
        self.feature
    }
    /// Inspect local X, Y and normal source bounds; example: `result.axes()[2]`.
    pub fn axes(&self) -> &[ConditionalPrismAxis; 3] {
        &self.axes
    }
}

/// Conditional error bounds for points swept by one session-owned prism input.
pub struct ConditionalPrismSweptPoints<'b, 'e, 's, O> {
    _displacement: &'b ConditionalPrismDisplacement<'e, 's, O>,
    root: ConditionalSketchRoot,
    points: Vec<oblikovati_model::history::ConditionalPointDisplacement>,
}
impl<O> ConditionalPrismSweptPoints<'_, '_, '_, O> {
    /// Read point identities and their conditional swept displacement; example: `result.points()`.
    pub fn points(&self) -> &[oblikovati_model::history::ConditionalPointDisplacement] {
        &self.points
    }
    /// Read the root used to calculate the points; example: `result.root()`.
    pub fn root(&self) -> &ConditionalSketchRoot {
        &self.root
    }
}
impl<'e, 's, O: TransactionObserver> ConditionalPrismDisplacement<'e, 's, O> {
    /// Compose source and sketch errors for all points on a canonical-plane sweep.
    /// Example: `displacement.swept_points(&sketch_box)?`.
    /// This bound is conditional and does not authorize a kernel operation.
    pub fn swept_points<'b>(
        &'b self,
        sketch_box: &[[f64; 2]],
    ) -> Result<ConditionalPrismSweptPoints<'b, 'e, 's, O>, String> {
        let projection = self._evidence.projection()?;
        let root = projection.conditional_root_auto(sketch_box, &[])?;
        self.swept_points_with_root(&root)
    }
    /// Reuse a root from this exact session evidence; example: `displacement.swept_points_with_root(&root)?`.
    /// A root from a different snapshot is refused even when its numeric values match.
    pub fn swept_points_with_root<'b>(
        &'b self,
        root: &ConditionalProjectedSketchRoot<'_, '_>,
    ) -> Result<ConditionalPrismSweptPoints<'b, 'e, 's, O>, String> {
        self._evidence.require_root(root)?;
        let definition = self._evidence.prism_definition(self.feature)?;
        let center = definition
            .resolve(self._evidence.session.snapshot())
            .map_err(|e| e.to_string())?
            .displacement;
        let bounds = std::array::from_fn(|axis| self.axes[axis].bounds());
        let points = root.canonical_swept_point_displacements(center, bounds)?;
        Ok(ConditionalPrismSweptPoints {
            _displacement: self,
            root: root.root().clone(),
            points,
        })
    }
}
impl<'s, O: TransactionObserver> SessionConditionalSketch<'s, O> {
    /// Bound a prism's source components under explicit constant assumptions.
    /// Example: `evidence.conditional_prism_displacement(feature, assumptions)?`.
    /// Parameters use conditional authored bounds; this does not qualify extrusion.
    pub fn conditional_prism_displacement<'e>(
        &'e self,
        feature: FeatureId,
        constants: [Option<[f64; 2]>; 3],
    ) -> Result<ConditionalPrismDisplacement<'e, 's, O>, String> {
        let definition = self.prism_definition(feature)?;
        let axes = self.displacement_axes(definition, constants)?;
        require_nonzero_normal(axes[2].bounds)?;
        Ok(ConditionalPrismDisplacement {
            _evidence: self,
            feature,
            axes,
        })
    }
    pub(super) fn prism_definition(
        &self,
        feature: FeatureId,
    ) -> Result<&SketchPrismDefinition, String> {
        let authored = self
            .session
            .snapshot()
            .work_geometry
            .get(feature)
            .map_err(|error| error.to_string())?;
        let PartFeatureDefinition::SketchPrism(definition) = authored.definition() else {
            return Err(format!(
                "feature {feature:?}: expected an authored sketch prism"
            ));
        };
        if definition.sketch() != self.sketch {
            return Err(format!(
                "feature {feature:?} sketch {:?}: expected {:?}",
                definition.sketch(),
                self.sketch
            ));
        }
        Ok(definition)
    }
    fn displacement_axes(
        &self,
        definition: &SketchPrismDefinition,
        constants: [Option<[f64; 2]>; 3],
    ) -> Result<[ConditionalPrismAxis; 3], String> {
        let snapshot = self.session.snapshot();
        let resolved = definition
            .resolve(snapshot)
            .map_err(|error| error.to_string())?;
        let mut axes = Vec::with_capacity(3);
        for (index, source) in definition.displacement().into_iter().enumerate() {
            let flipped = index == 2 && definition.normal_is_reversed();
            let value = resolved.displacement[index];
            let mut axis = axis_bound(
                source,
                constants[index],
                if flipped { -value } else { value },
                &snapshot.parameters,
            )?;
            if flipped {
                axis.bounds = [-axis.bounds[1], -axis.bounds[0]];
            }
            axes.push(axis);
        }
        axes.try_into()
            .map_err(|_| "displacement axes: expected three bounds".to_owned())
    }
}
fn require_nonzero_normal(bounds: [f64; 2]) -> Result<(), String> {
    if bounds[0] <= 0. && bounds[1] >= 0. {
        return Err(format!(
            "normal displacement {bounds:?}: expected bounds excluding zero"
        ));
    }
    Ok(())
}
fn axis_bound(
    source: DatumLength,
    assumption: Option<[f64; 2]>,
    value: f64,
    parameters: &oblikovati_model::parameter::ParameterCollection<u64>,
) -> Result<ConditionalPrismAxis, String> {
    let (bounds, parameter) = match source {
        DatumLength::Constant(_) => (
            assumption
                .ok_or_else(|| format!("constant {source:?}: expected explicit source bounds"))?,
            None,
        ),
        DatumLength::Parameter(id) if assumption.is_none() => {
            let bound = conditional_parameter_bound(parameters, id)?;
            (bound.bounds(), Some(bound))
        }
        DatumLength::Parameter(id) => {
            return Err(format!(
                "parameter {id}: expected no caller constant bounds"
            ));
        }
    };
    if !bounds.iter().all(|number| number.is_finite())
        || bounds[0] > bounds[1]
        || value < bounds[0]
        || value > bounds[1]
    {
        return Err(format!(
            "source {source:?} value {value:?} and bounds {bounds:?}: expected finite ordered bounds containing the current value"
        ));
    }
    Ok(ConditionalPrismAxis {
        source,
        bounds,
        parameter,
    })
}

#[cfg(test)]
mod tests {
    use super::require_nonzero_normal;

    #[test]
    fn normal_bounds_refuse_zero_and_allow_both_directions() {
        for bounds in [[0., 1.], [-1., 0.], [-1., 1.]] {
            assert!(require_nonzero_normal(bounds).is_err());
        }
        for bounds in [[1., 2.], [-2., -1.]] {
            assert!(require_nonzero_normal(bounds).is_ok());
        }
    }
}
