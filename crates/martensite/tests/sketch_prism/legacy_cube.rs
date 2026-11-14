// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use martensite_tkbo::profile_extrusion::{DisplayTolerance, Resolution, ToleranceCap};
use oblikovati_martensite::{
    PrismResolution, certified_sketch_regions, retain_certified_sketch_prism,
};
use oblikovati_model::{
    document_sketch::DocumentSketchId, feature::sketch_prism::SketchPrismSelection,
    history::DocumentSnapshot,
};
use oblikovati_persistence::package::{
    DocumentPackage, LegacyProfileResolver, LegacySketchCandidate,
};

const GO_CUBE: &[u8] = include_bytes!(
    "../../../../architecture/rust-port/receipts/2026-11-14-go-cube-fixture/go-cube.obk"
);

struct GoCubeProfile;
impl LegacyProfileResolver for GoCubeProfile {
    fn resolve(
        &self,
        candidate: &LegacySketchCandidate,
        sketch: DocumentSketchId,
        index: usize,
    ) -> Result<SketchPrismSelection, String> {
        if index != 0 {
            return Err(format!("Go fixture profile index {index}: expected 0"));
        }
        let document = candidate_document(candidate);
        let regions = certified_sketch_regions(&document, sketch, app_resolution())
            .map_err(|e| e.to_string())?;
        let [region] = regions.as_slice() else {
            return Err(format!(
                "fixture regions {}: expected exactly one",
                regions.len()
            ));
        };
        region
            .line_selection(
                document
                    .sketches
                    .get(sketch)
                    .map_err(|e| e.to_string())?
                    .geometry(),
            )
            .map_err(|e| e.to_string())
    }
}
fn candidate_document(candidate: &LegacySketchCandidate) -> DocumentSnapshot {
    let mut document = selected_sketch_document();
    document.sketches = candidate.sketches.clone();
    document.parameters = candidate.parameters.parameters.clone();
    document.work_geometry = Default::default();
    solve_selected_sketches(&mut document);
    document
}
fn app_resolution() -> PrismResolution {
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
fn kernel_resolution() -> Resolution {
    Resolution::new(
        1e-6,
        1e-7,
        1e-9,
        DisplayTolerance {
            chordal: 1e-3,
            angular: 0.1,
        },
        ToleranceCap::Refuse(1e-5),
    )
}
#[test]
fn go_cube_candidate_replays_as_a_certified_solid_and_keeps_fixed_depth() {
    let package = DocumentPackage::decode(GO_CUBE).unwrap();
    let candidate = package
        .prepare_legacy_part_candidate(&GoCubeProfile)
        .unwrap();
    let definition = &candidate.extrusions[0].definition;
    let mut document = candidate_document(&candidate.sketches);
    let first = retain_certified_sketch_prism(&document, definition, kernel_resolution()).unwrap();
    assert_analytic_scalar(first.body.report().properties.volume, 1.0);
    assert_eq!(first.body.report().faces.len(), 6);
    for face in &first.body.report().faces {
        assert_analytic_scalar(face.area, 1.0);
        assert_eq!(face.loops, 1);
    }
    let size = document.parameters.find("size").unwrap();
    document.parameters.set_expression(size, "20 mm").unwrap();
    solve_selected_sketches(&mut document);
    let second = retain_certified_sketch_prism(&document, definition, kernel_resolution()).unwrap();
    assert_analytic_scalar(second.body.report().properties.volume, 4.0);
    assert!(second.body.report().topology.solid);
    assert_eq!(second.body.report().faces.len(), 6);
    assert!(
        second
            .body
            .report()
            .faces
            .iter()
            .all(|face| face.loops == 1)
    );
    let mut areas: Vec<_> = second
        .body
        .report()
        .faces
        .iter()
        .map(|face| face.area)
        .collect();
    areas.sort_by(f64::total_cmp);
    for (actual, expected) in areas.into_iter().zip([2., 2., 2., 2., 4., 4.]) {
        assert_analytic_scalar(actual, expected);
    }
}
