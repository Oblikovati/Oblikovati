// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_martensite::{PrismResolution, certified_sketch_regions};
use oblikovati_model::sketch::{SketchEntityId, SketchSpline2D, SketchSplineKind};
fn resolution() -> PrismResolution {
    PrismResolution {
        version: 1,
        weld: 1e-6,
        sew: 1e-7,
        angular: 1e-9,
        display_chordal: 1e-3,
        display_angular: 0.1,
        maximum_tolerance: 1e-5,
    }
}
fn ids(boundary: &[oblikovati_martensite::SketchRegionCurveUse]) -> Vec<u64> {
    let mut ids: Vec<_> = boundary.iter().map(|edge| edge.curve.0).collect();
    ids.sort_unstable();
    ids
}
#[test]
fn closed_sketch_regions_preserve_boundary_curves_and_nested_holes() {
    let mut document = selected_sketch_document();
    let regions = certified_sketch_regions(&document, SKETCH, resolution()).unwrap();
    assert_eq!(regions.len(), 1);
    assert_eq!(ids(&regions[0].outer), [101, 102, 103, 104]);
    assert!(regions[0].holes.is_empty());
    let selection = regions[0]
        .line_selection(document.sketches.get(SKETCH).unwrap().geometry())
        .unwrap();
    let definition = oblikovati_model::feature::sketch_prism::SketchPrismDefinition::new(
        SKETCH,
        selection,
        [0., 0., 2.].map(oblikovati_model::feature::work_geometry::DatumLength::Constant),
    )
    .unwrap();
    let kernel_resolution = martensite_tkbo::profile_extrusion::Resolution::new(
        1e-6,
        1e-7,
        1e-9,
        martensite_tkbo::profile_extrusion::DisplayTolerance {
            chordal: 1e-3,
            angular: 0.1,
        },
        martensite_tkbo::profile_extrusion::ToleranceCap::Refuse(1e-5),
    );
    let body = oblikovati_martensite::retain_certified_sketch_prism(
        &document,
        &definition,
        kernel_resolution,
    )
    .unwrap();
    assert_analytic_scalar(body.body.report().properties.volume, 24.);
    add_fixed_sketch_hole(&mut document);
    solve_selected_sketches(&mut document);
    let regions = certified_sketch_regions(&document, SKETCH, resolution()).unwrap();
    assert_eq!(regions.len(), 2);
    let ring = regions
        .iter()
        .find(|region| !region.holes.is_empty())
        .unwrap();
    assert_eq!(ids(&ring.outer), [101, 102, 103, 104]);
    assert_eq!(ids(&ring.holes[0]), [111, 112, 113, 114]);
}
#[test]
fn construction_curves_are_excluded_and_unconverted_splines_decline() {
    let mut document = selected_sketch_document();
    add_fixed_sketch_hole(&mut document);
    let mut sketch = document.sketches.get(SKETCH).unwrap().geometry().clone();
    sketch
        .set_curve_construction(&[111, 112, 113, 114].map(SketchEntityId), true)
        .unwrap();
    document.sketches.replace_geometry(SKETCH, sketch).unwrap();
    solve_selected_sketches(&mut document);
    assert_eq!(
        certified_sketch_regions(&document, SKETCH, resolution())
            .unwrap()
            .len(),
        1
    );
    let mut sketch = document.sketches.get(SKETCH).unwrap().geometry().clone();
    sketch
        .add_spline(
            SketchSpline2D::new(
                SketchEntityId(200),
                vec![SketchEntityId(1), SketchEntityId(2)],
                false,
                SketchSplineKind::Control,
            )
            .unwrap(),
        )
        .unwrap();
    document.sketches.replace_geometry(SKETCH, sketch).unwrap();
    solve_selected_sketches(&mut document);
    let error = certified_sketch_regions(&document, SKETCH, resolution()).unwrap_err();
    assert_eq!(
        error.kind(),
        oblikovati_martensite::EvaluationFailureKind::Unsupported
    );
    assert_eq!(error.sketch(), Some(SKETCH));
}

#[test]
fn line_recipe_refuses_a_curved_region_with_a_named_decline() {
    let mut sketch = oblikovati_model::sketch::Sketch2D::default();
    sketch.add_point(SketchEntityId(1), [0., 0.]).unwrap();
    sketch
        .add_circle(SketchEntityId(2), SketchEntityId(1), 2.)
        .unwrap();
    let region = oblikovati_martensite::SketchRegionBoundary {
        outer: vec![oblikovati_martensite::SketchRegionCurveUse {
            curve: SketchEntityId(2),
            reversed: false,
        }],
        holes: vec![],
    };
    let error = region.line_selection(&sketch).unwrap_err();
    assert_eq!(
        error.kind(),
        oblikovati_martensite::EvaluationFailureKind::Unsupported
    );
    assert!(error.to_string().contains("SketchEntityId(2)"));
}
