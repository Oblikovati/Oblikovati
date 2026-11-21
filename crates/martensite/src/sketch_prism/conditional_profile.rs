// SPDX-License-Identifier: GPL-2.0-only
//! Conditional source attribution using the same profile input converter as region discovery.
use super::regions::input::profile;
use martensite_tkbo::profile_extrusion::ProfileCurve;
use oblikovati_model::{
    history::{ConditionalPointDisplacement, ConditionalProjectedSketchRoot},
    sketch::{Sketch2D, SketchEntityId, SketchPlane},
};
use std::collections::BTreeMap;

/// One exact converted profile line and its converter-selected endpoints.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ConditionalProfileLine {
    pub curve: SketchEntityId,
    pub endpoints: [SketchEntityId; 2],
}
/// One converted profile vertex and the authored points represented by it.
#[derive(Clone, Debug, PartialEq)]
pub struct ConditionalProfileVertex {
    pub vertex: SketchEntityId,
    pub sources: Vec<SketchEntityId>,
    pub point: [f64; 2],
    pub displacement_upper: f64,
}
/// Actual converter identities tied to conditional root evidence, not feature admission.
pub struct ConditionalProfileMapping<'r, 'e, 's> {
    _root: &'r ConditionalProjectedSketchRoot<'e, 's>,
    lines: Vec<ConditionalProfileLine>,
    vertices: Vec<ConditionalProfileVertex>,
}
impl ConditionalProfileMapping<'_, '_, '_> {
    /// Read every converted profile line; example: `mapping.lines()`.
    pub fn lines(&self) -> &[ConditionalProfileLine] {
        &self.lines
    }
    /// Read vertex source identities and conditional bounds; example: `mapping.vertices()`.
    pub fn vertices(&self) -> &[ConditionalProfileVertex] {
        &self.vertices
    }
}
/// Attribute the complete line profile to the conditional root using the production converter.
/// Example: `conditional_profile_mapping(&root)?` before checking a tolerance policy.
/// This does not certify interval arithmetic, kernel admission, or an output solid.
pub fn conditional_profile_mapping<'r, 'e, 's>(
    root: &'r ConditionalProjectedSketchRoot<'e, 's>,
) -> Result<ConditionalProfileMapping<'r, 'e, 's>, String> {
    let projection = root.projection();
    let resolved = projection
        .snapshot()
        .require_converged_resolved_sketch(projection.sketch())
        .map_err(|e| e.to_string())?;
    let errors = root.canonical_point_displacements()?;
    let (lines, vertices) = convert(root.root().sketch(), resolved.plane, &errors)?;
    Ok(ConditionalProfileMapping {
        _root: root,
        lines,
        vertices,
    })
}
fn convert(
    sketch: &Sketch2D,
    plane: SketchPlane,
    errors: &[ConditionalPointDisplacement],
) -> Result<(Vec<ConditionalProfileLine>, Vec<ConditionalProfileVertex>), String> {
    let (input, sources) = profile(sketch, plane).map_err(|error| error.to_string())?;
    let bounds: BTreeMap<_, _> = errors
        .iter()
        .map(|entry| (entry.point, entry.upper))
        .collect();
    if bounds.len() != errors.len() {
        return Err("point bounds: expected unique source identities".into());
    }
    let lines = input
        .curves
        .into_iter()
        .map(convert_line)
        .collect::<Result<Vec<_>, _>>()?;
    let vertices = input
        .vertices
        .into_iter()
        .map(|vertex| {
            let ids = sources.get(&vertex.id.0).ok_or_else(|| {
                format!(
                    "vertex {:?}: expected converter source identities",
                    vertex.id
                )
            })?;
            let ids: Vec<_> = ids.iter().copied().map(SketchEntityId).collect();
            let upper = source_upper(&ids, &bounds)?;
            Ok(ConditionalProfileVertex {
                vertex: SketchEntityId(vertex.id.0),
                sources: ids,
                point: vertex.point,
                displacement_upper: upper,
            })
        })
        .collect::<Result<Vec<_>, String>>()?;
    Ok((lines, vertices))
}
fn convert_line(curve: ProfileCurve) -> Result<ConditionalProfileLine, String> {
    match curve {
        ProfileCurve::Line { id, start, end } => Ok(ConditionalProfileLine {
            curve: SketchEntityId(id.0),
            endpoints: [SketchEntityId(start.0), SketchEntityId(end.0)],
        }),
        other => Err(format!(
            "profile curve {other:?}: expected a line for conditional input bounds"
        )),
    }
}
fn source_upper(
    ids: &[SketchEntityId],
    bounds: &BTreeMap<SketchEntityId, f64>,
) -> Result<f64, String> {
    if ids.is_empty() {
        return Err("vertex sources []: expected at least one authored point".into());
    }
    ids.iter().try_fold(0.0_f64, |upper, id| {
        let value = *bounds
            .get(id)
            .ok_or_else(|| format!("point {id:?}: expected conditional displacement evidence"))?;
        if !value.is_finite() || value < 0. {
            return Err(format!(
                "point {id:?} bound {value:?}: expected finite nonnegative displacement"
            ));
        }
        Ok(upper.max(value))
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use oblikovati_model::sketch::{PlanarConstraint, SketchConstraintId};

    fn fixture() -> (Sketch2D, Vec<ConditionalPointDisplacement>) {
        let mut sketch = Sketch2D::default();
        for (id, point) in [
            (1, [0., 0.]),
            (2, [0., 0.]),
            (3, [1., 0.]),
            (4, [1., 1.]),
            (5, [0., 0.]),
            (6, [0., 0.]),
        ] {
            sketch.add_point(SketchEntityId(id), point).unwrap();
        }
        sketch
            .add_constraint(
                SketchConstraintId(1),
                PlanarConstraint::Coincident {
                    points: [SketchEntityId(1), SketchEntityId(2)],
                },
            )
            .unwrap();
        sketch
            .add_constraint(
                SketchConstraintId(2),
                PlanarConstraint::Coincident {
                    points: [SketchEntityId(2), SketchEntityId(6)],
                },
            )
            .unwrap();
        for (id, points) in [(10, [2, 3]), (11, [3, 4]), (12, [1, 4]), (13, [5, 4])] {
            sketch
                .add_line(SketchEntityId(id), points.map(SketchEntityId))
                .unwrap();
        }
        sketch
            .set_curve_construction(&[SketchEntityId(12)], true)
            .unwrap();
        let errors = (1..=6)
            .map(|id| ConditionalPointDisplacement {
                point: SketchEntityId(id),
                upper: id as f64 / 10.,
            })
            .collect();
        (sketch, errors)
    }
    #[test]
    fn mapping_uses_converter_identity_and_all_nonconstruction_curves() {
        let (sketch, errors) = fixture();
        let (lines, vertices) = convert(&sketch, SketchPlane::xy(), &errors).unwrap();
        assert_eq!(
            lines.iter().map(|line| line.curve.0).collect::<Vec<_>>(),
            [10, 11, 13]
        );
        assert_eq!(lines[0].endpoints, [SketchEntityId(1), SketchEntityId(3)]);
        let first = vertices
            .iter()
            .find(|vertex| vertex.vertex == SketchEntityId(1))
            .unwrap();
        assert_eq!(
            first.sources,
            [SketchEntityId(1), SketchEntityId(2), SketchEntityId(6)]
        );
        assert_eq!(first.displacement_upper, 0.6);
        assert!(
            vertices
                .iter()
                .any(|vertex| vertex.vertex == SketchEntityId(5)
                    && vertex.sources == [SketchEntityId(5)])
        );
    }
    #[test]
    fn missing_duplicate_and_inconsistent_input_evidence_declines() {
        let (mut sketch, mut errors) = fixture();
        errors.pop();
        assert!(convert(&sketch, SketchPlane::xy(), &errors).is_err());
        errors.push(errors[0]);
        assert!(convert(&sketch, SketchPlane::xy(), &errors).is_err());
        sketch.set_position(SketchEntityId(2), [0.1, 0.]).unwrap();
        assert!(convert(&sketch, SketchPlane::xy(), &errors[..4]).is_err());
    }
    #[test]
    fn nonfinite_and_negative_source_bounds_decline() {
        let (sketch, mut errors) = fixture();
        for value in [f64::NAN, f64::INFINITY, -1.] {
            errors[0].upper = value;
            assert!(convert(&sketch, SketchPlane::xy(), &errors).is_err());
        }
    }
    #[test]
    fn curved_profile_declines_instead_of_dropping_geometry() {
        let (mut sketch, errors) = fixture();
        sketch
            .add_circle(SketchEntityId(20), SketchEntityId(1), 0.5)
            .unwrap();
        let error = convert(&sketch, SketchPlane::xy(), &errors).unwrap_err();
        assert!(error.contains("expected a line"), "{error}");
    }
}
