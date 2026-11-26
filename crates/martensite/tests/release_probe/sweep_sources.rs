// SPDX-License-Identifier: GPL-2.0-only
#[path = "../../../model/tests/common/document_snapshot.rs"]
mod document_snapshot;

use oblikovati_martensite::{
    PrismResolution, PrismResolutionPolicy, certified_sketch_regions_with_policy,
    sweep::{SweepFaceRole, build_sweep},
};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{
        AuthoredFeature, FeatureId, FeatureReadKey,
        sketch_prism::SketchDirectedCurve,
        sketch_sweep::{SketchSweepDefinition, SketchSweepRecipe},
        work_geometry::WorkGeometryDefinition,
    },
    history::DocumentSnapshot,
    sketch::{Sketch2D, SketchEntityId, SketchPlane},
};
use std::f64::consts::PI;

const PROFILE: DocumentSketchId = DocumentSketchId(1);
const PATH: DocumentSketchId = DocumentSketchId(2);
const PLANE: FeatureId = FeatureId(1);
const SECTION: SketchEntityId = SketchEntityId(3);
const SPINE: SketchEntityId = SketchEntityId(7);

#[test]
fn saved_sweep_rereads_its_path_from_the_current_document() {
    let (mut snapshot, sweep) = fixture();
    let recipe = serde_json::to_vec(sweep.recipe()).unwrap();
    let restored =
        SketchSweepDefinition::from_recipe(serde_json::from_slice(&recipe).unwrap()).unwrap();
    assert_eq!(restored, sweep);
    assert_eq!(
        sweep
            .reads(&snapshot)
            .unwrap()
            .keys()
            .iter()
            .copied()
            .collect::<Vec<_>>(),
        [
            FeatureReadKey::Feature(PLANE),
            FeatureReadKey::Sketch(1),
            FeatureReadKey::Sketch(2)
        ],
    );
    assert_sweep(&snapshot, &restored, 3.);
    snapshot
        .sketches
        .replace_geometry(PATH, path(4., SPINE))
        .unwrap();
    solve(&mut snapshot);
    assert_sweep(&snapshot, &restored, 4.);
}

#[test]
fn disconnected_authored_path_cannot_return_a_partial_solid() {
    let (mut snapshot, sweep) = fixture();
    let mut geometry = path(3., SPINE);
    let [start, end] = [8, 9].map(SketchEntityId);
    geometry.add_point(start, [0., 5.]).unwrap();
    geometry.add_point(end, [0., 6.]).unwrap();
    let extra = SketchEntityId(10);
    geometry.add_line(extra, [start, end]).unwrap();
    snapshot.sketches.replace_geometry(PATH, geometry).unwrap();
    solve(&mut snapshot);
    let mut recipe = sweep.recipe().clone();
    recipe.path.push(directed(extra));
    let sweep = SketchSweepDefinition::from_recipe(recipe).unwrap();
    let datums = snapshot
        .work_geometry
        .evaluate_work_geometry(&snapshot.parameters)
        .unwrap();
    let failure = build_sweep(sweep.resolve(&snapshot, &datums).unwrap(), resolution())
        .err()
        .expect("expected a disconnected-path failure");
    assert_eq!(
        failure.entity(),
        &oblikovati_martensite::EvaluationEntity::SketchEntity(extra.0)
    );
}

#[test]
fn saved_sweep_declines_a_replaced_path_even_at_identical_coordinates() {
    let (mut snapshot, sweep) = fixture();
    snapshot
        .sketches
        .replace_geometry(PATH, path(3., SketchEntityId(8)))
        .unwrap();
    solve(&mut snapshot);
    let datums = snapshot
        .work_geometry
        .evaluate_work_geometry(&snapshot.parameters)
        .unwrap();
    let error = sweep.resolve(&snapshot, &datums).unwrap_err().to_string();
    assert!(
        error.contains("DocumentSketchId(2)") && error.contains("SketchEntityId(7)"),
        "{error}"
    );
}

#[test]
fn sweep_recipe_declines_an_unknown_version_and_duplicate_path_members() {
    let (_, sweep) = fixture();
    let mut recipe = sweep.recipe().clone();
    recipe.version += 1;
    assert!(SketchSweepDefinition::from_recipe(recipe).is_err());
    let mut recipe: SketchSweepRecipe = sweep.recipe().clone();
    recipe.path.push(recipe.path[0]);
    assert!(SketchSweepDefinition::from_recipe(recipe).is_err());
}

fn fixture() -> (DocumentSnapshot, SketchSweepDefinition) {
    let mut snapshot = document_snapshot::empty_snapshot();
    let frame = SketchPlane::new([0.; 3], [1., 0., 0.], [0., 0., 1.], Default::default()).unwrap();
    snapshot
        .work_geometry
        .append(AuthoredFeature::work_plane(
            PLANE,
            "Path plane",
            WorkGeometryDefinition::Fixed(frame),
        ))
        .unwrap();
    snapshot
        .sketches
        .add(PROFILE, "Section", SketchPlane::xy(), profile())
        .unwrap();
    snapshot
        .sketches
        .add_on_work_plane(PATH, "Path", PLANE, path(3., SPINE))
        .unwrap();
    solve(&mut snapshot);
    let regions = certified_sketch_regions_with_policy(&snapshot, PROFILE, resolution()).unwrap();
    assert_eq!(regions.len(), 1);
    let profile = regions[0].selection();
    let sweep = SketchSweepDefinition::new(PROFILE, profile, PATH, vec![directed(SPINE)]).unwrap();
    (snapshot, sweep)
}

fn profile() -> Sketch2D {
    let mut sketch = Sketch2D::default();
    let center = SketchEntityId(2);
    sketch.add_point(center, [0., 0.]).unwrap();
    sketch.add_circle(SECTION, center, 0.5).unwrap();
    sketch
}

fn path(height: f64, edge: SketchEntityId) -> Sketch2D {
    let mut sketch = Sketch2D::default();
    let [start, end] = [5, 6].map(SketchEntityId);
    sketch.add_point(start, [0., 0.]).unwrap();
    sketch.add_point(end, [0., height]).unwrap();
    sketch.add_line(edge, [start, end]).unwrap();
    sketch
}

fn directed(curve: SketchEntityId) -> SketchDirectedCurve {
    SketchDirectedCurve {
        curve,
        reversed: false,
    }
}

fn solve(snapshot: &mut DocumentSnapshot) {
    snapshot.sketches = snapshot
        .sketches
        .solve_pending(&snapshot.parameters, Default::default(), || false)
        .candidate;
}

fn assert_sweep(snapshot: &DocumentSnapshot, sweep: &SketchSweepDefinition, height: f64) {
    let datums = snapshot
        .work_geometry
        .evaluate_work_geometry(&snapshot.parameters)
        .unwrap();
    let input = sweep.resolve(snapshot, &datums).unwrap();
    let result = build_sweep(input, resolution()).unwrap();
    assert_eq!(result.reported_surface_error(), 0.);
    assert!(result.achieved_tolerance().edge <= 1e-6);
    assert_eq!(
        result.face_roles().collect::<Vec<_>>(),
        [
            SweepFaceRole::Start,
            SweepFaceRole::End,
            SweepFaceRole::Side {
                profile: SECTION,
                path: SPINE
            },
        ]
    );
    assert!((result.properties().unwrap().volume - PI * height / 4.).abs() < 1e-8);
}

fn resolution() -> PrismResolutionPolicy {
    PrismResolutionPolicy::Fixed(PrismResolution {
        version: 1,
        weld: 1e-7,
        sew: 1e-7,
        angular: 1e-9,
        display_chordal: 0.01,
        display_angular: 0.1,
        maximum_tolerance: 1e-6,
    })
}

#[path = "sweep_sources/replay.rs"]
mod replay;

#[path = "sweep_sources/taper.rs"]
mod taper;

#[test]
fn fixed_orientation_translates_along_an_oblique_straight_path() {
    let (mut snapshot, sweep) = fixture();
    let mut geometry = Sketch2D::default();
    let [start, end] = [SketchEntityId(5), SketchEntityId(6)];
    geometry.add_point(start, [0., 0.]).unwrap();
    geometry.add_point(end, [2., 3.]).unwrap();
    geometry.add_line(SPINE, [start, end]).unwrap();
    snapshot.sketches.replace_geometry(PATH, geometry).unwrap();
    solve(&mut snapshot);
    let datums = snapshot
        .work_geometry
        .evaluate_work_geometry(&snapshot.parameters)
        .unwrap();
    let fixed = sweep.with_fixed_orientation();
    let body = build_sweep(fixed.resolve(&snapshot, &datums).unwrap(), resolution()).unwrap();
    let result = body.properties().unwrap();
    assert!((result.volume - 0.75 * PI).abs() < 1e-8);
    for (current, expected) in result.centroid.into_iter().zip([1., 0., 1.5]) {
        assert!((current - expected).abs() < 1e-8, "{result:?}");
    }
}
