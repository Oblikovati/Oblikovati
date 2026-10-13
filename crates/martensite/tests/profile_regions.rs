// SPDX-License-Identifier: GPL-2.0-only
//! Integration evidence for released kernel arrangement, not an application profile engine.
use rustycad_tkbo::{bop_algo::BuilderFace, brep_algo_api::BRepAlgoAPIBuilderAlgo};
use rustycad_tkbrep::{
    topexp,
    topods::{Shape, edge, face},
    toptools::IndexedMapOfShape,
};
use rustycad_tkernel::{ncollection::List, standard::Failure};
use rustycad_tkg3d::top_abs::{Orientation, ShapeEnum, State};
use rustycad_tkgeombase::gprop::GProps;
use rustycad_tkmath::gp::Plane3;
use rustycad_tktopalgo::{
    brep_builder_api::BRepBuilderAPIMakeFace, brep_check::BRepCheckAnalyzer, brep_gprop::BRepGProp,
    brep_lib, brep_top_adaptor::BRepTopAdaptorFClass2d,
};
#[path = "profile_regions/fixture.rs"]
mod fixture;
#[path = "profile_regions/history.rs"]
mod history;

struct Arrangement {
    sources: IndexedMapOfShape,
    split: IndexedMapOfShape,
    fuse: BRepAlgoAPIBuilderAlgo,
}
#[derive(Debug)]
struct RegionReport {
    area: f64,
    centroid: [f64; 3],
    loops: usize,
    edges: usize,
    original_region_parents: Vec<usize>,
}
fn indexed_region_shapes(shape: &Shape, kind: ShapeEnum) -> Result<IndexedMapOfShape, Failure> {
    let mut found = IndexedMapOfShape::default();
    topexp::map_shapes_of_type(shape, kind, &mut found)?;
    Ok(found)
}
fn arrange_region_fixture(shape: &Shape) -> Result<Arrangement, Failure> {
    let sources = indexed_region_shapes(shape, ShapeEnum::Edge)?;
    let fuse = fuse_region_edges(&sources)?;
    assert!(
        !fuse.has_errors(),
        "expected successful released general fuse"
    );
    let split = indexed_region_shapes(fuse.shape(), ShapeEnum::Edge)?;
    Ok(Arrangement {
        sources,
        split,
        fuse,
    })
}
fn fuse_region_edges(sources: &IndexedMapOfShape) -> Result<BRepAlgoAPIBuilderAlgo, Failure> {
    let mut arguments = List::default();
    for index in 1..=sources.extent() {
        arguments.append(sources.find_key(index)?.clone());
    }
    let mut fuse = BRepAlgoAPIBuilderAlgo::new();
    fuse.set_arguments(&arguments);
    fuse.set_non_destructive(true);
    fuse.set_to_fill_history(true);
    fuse.build()?;
    Ok(fuse)
}
fn orient_split_edges(
    split: &IndexedMapOfShape,
    plane: &rustycad_tkbrep::topods::Face,
) -> Result<List<Shape>, Failure> {
    let mut inputs = List::default();
    for index in 1..=split.extent() {
        let segment = edge(split.find_key(index)?)?;
        brep_lib::build_pcurve_for_edge_on_plane(&segment, plane)?;
        inputs.append(segment.oriented(Orientation::Forward));
        inputs.append(segment.oriented(Orientation::Reversed));
    }
    Ok(inputs)
}
fn build_fixture_regions(arrangement: &Arrangement) -> Result<BuilderFace, Failure> {
    build_regions_from_split(&arrangement.split)
}
fn build_regions_from_split(split: &IndexedMapOfShape) -> Result<BuilderFace, Failure> {
    let plane = BRepBuilderAPIMakeFace::from_plane(Plane3::default())?.face()?;
    let mut builder = BuilderFace::new();
    builder.set_face(&plane);
    builder.set_shapes(&orient_split_edges(split, &plane)?);
    builder.perform()?;
    assert!(
        !builder.options().has_errors(),
        "expected successful released face builder"
    );
    Ok(builder)
}
fn measure_fixture_region(
    shape: &Shape,
    arrangement: &Arrangement,
) -> Result<RegionReport, Failure> {
    let edges = indexed_region_shapes(shape, ShapeEnum::Edge)?;
    let parents = history::original_region_parents(&edges, arrangement)?;
    measure_region_geometry(shape, parents)
}
fn measure_region_geometry(
    shape: &Shape,
    original_region_parents: Vec<usize>,
) -> Result<RegionReport, Failure> {
    assert!(BRepCheckAnalyzer::new(shape, true, false, true)?.is_valid()?);
    let props = region_surface_properties(shape)?;
    let center = props.centre_of_mass();
    let edges = indexed_region_shapes(shape, ShapeEnum::Edge)?;
    Ok(RegionReport {
        area: props.mass(),
        centroid: [center.x(), center.y(), center.z()],
        loops: indexed_region_shapes(shape, ShapeEnum::Wire)?.extent(),
        edges: edges.extent(),
        original_region_parents,
    })
}
fn bounded_region_reports(
    builder: &BuilderFace,
    arrangement: &Arrangement,
) -> Result<Vec<RegionReport>, Failure> {
    let mut reports = Vec::new();
    for shape in builder.areas().iter() {
        let face = face(shape)?;
        let state = BRepTopAdaptorFClass2d::new(&face, 1e-9)?.perform_infinite_point()?;
        if state != State::Out {
            continue;
        }
        reports.push(measure_fixture_region(shape, arrangement)?);
    }
    reports.sort_by(|a, b| {
        a.area
            .total_cmp(&b.area)
            .then(a.centroid[0].total_cmp(&b.centroid[0]))
            .then(a.centroid[1].total_cmp(&b.centroid[1]))
    });
    Ok(reports)
}
fn verify_occt_region(region_receipt: &RegionReport, expected: &[f64]) {
    assert!(
        (region_receipt.area - expected[0]).abs() < 1e-8,
        "{region_receipt:?}: expected {expected:?}"
    );
    for (actual, expected) in region_receipt.centroid.iter().zip(&expected[1..4]) {
        assert!(
            (actual - expected).abs() < 1e-8,
            "{region_receipt:?}: expected centroid {expected:?}"
        );
    }
    assert_eq!(region_receipt.loops, expected[4] as usize);
    assert_eq!(region_receipt.edges, expected[5] as usize);
    assert!(!region_receipt.original_region_parents.is_empty());
}
fn occt_region_rows(case: &str) -> Vec<Vec<f64>> {
    include_str!("profile_regions/occt-8.0.0.csv")
        .lines()
        .filter_map(|line| line.strip_prefix(&format!("{case},")))
        .map(|row| {
            row.split(',')
                .map(|number| number.parse().unwrap())
                .collect()
        })
        .collect()
}
fn check_region_case(case: &str) {
    let arrangement =
        arrange_region_fixture(&fixture::build_region_fixture(case).unwrap()).unwrap();
    let builder = build_fixture_regions(&arrangement).unwrap();
    let reports = bounded_region_reports(&builder, &arrangement).unwrap();
    verify_region_set(case, &reports);
    history::require_region_history_coverage(&arrangement).unwrap();
}
fn verify_region_set(case: &str, reports: &[RegionReport]) {
    let rows = occt_region_rows(case);
    assert!(
        !rows.is_empty(),
        "{case}: expected retained independent OCCT evidence"
    );
    assert_eq!(
        reports.len(),
        rows.len(),
        "{case}: expected complete bounded-region coverage"
    );
    for (region_receipt, row) in reports.iter().zip(rows) {
        verify_occt_region(region_receipt, &row);
    }
}
#[test]
fn released_arrangement_regions_match_each_independent_occt_face() {
    for case in [
        "rectangle",
        "nested",
        "divider",
        "crossing",
        "circle",
        "ellipse",
    ] {
        check_region_case(case);
    }
}
#[test]
fn divider_history_preserves_both_halves_of_the_original_boundary() {
    let arrangement =
        arrange_region_fixture(&fixture::build_region_fixture("divider").unwrap()).unwrap();
    assert_eq!(arrangement.sources.extent(), 5);
    assert_eq!(arrangement.split.extent(), 7);
    let reports =
        bounded_region_reports(&build_fixture_regions(&arrangement).unwrap(), &arrangement)
            .unwrap();
    assert_eq!(reports.len(), 2);
    assert_ne!(
        reports[0].original_region_parents,
        reports[1].original_region_parents
    );
    for region_receipt in reports {
        assert_eq!(region_receipt.original_region_parents.len(), 4);
    }
}

#[test]
fn single_closed_curves_expose_the_recorded_general_fuse_limit() {
    for case in ["single-circle", "single-ellipse"] {
        let source = fixture::build_region_fixture(case).unwrap();
        let edges = indexed_region_shapes(&source, ShapeEnum::Edge).unwrap();
        assert_eq!(edges.extent(), 1);
        let fuse = fuse_region_edges(&edges).unwrap();
        assert!(fuse.has_errors());
        let mut failures = Vec::new();
        fuse.dump_errors(&mut failures).unwrap();
        let failures = String::from_utf8(failures).unwrap();
        let expected = include_str!("profile_regions/occt-8.0.0-declines.csv");
        assert!(
            expected
                .lines()
                .any(|line| line == format!("{case},{}", failures.trim()))
        );
        assert!(fuse.shape().is_null());
    }
}

fn exact_seed_state(
    face: &rustycad_tkbrep::topods::Face,
    seed: [f64; 2],
) -> Result<State, Failure> {
    use rustycad_tkmath::gp::Point2;
    use rustycad_tktopalgo::brep_class::BRepClassFaceClassifier;
    // tol:metric fixture precision; this test does not choose application tolerances.
    Ok(BRepClassFaceClassifier::with_face_point2d(
        face,
        &Point2::new(seed[0], seed[1]),
        1e-9,
        false,
        1e-9,
    )?
    .state())
}
fn region_surface_properties(shape: &Shape) -> Result<GProps, Failure> {
    let mut props = GProps::default();
    BRepGProp::surface_properties(shape, &mut props, false, false)?;
    Ok(props)
}
fn classify_region_seed(
    builder: &BuilderFace,
    seed: [f64; 2],
) -> Result<Vec<(State, f64)>, Failure> {
    let mut matched = Vec::new();
    for shape in builder.areas().iter() {
        let face = face(shape)?;
        if BRepTopAdaptorFClass2d::new(&face, 1e-9)?.perform_infinite_point()? != State::Out {
            continue;
        }
        let state = exact_seed_state(&face, seed)?;
        if state == State::Out {
            continue;
        }
        matched.push((state, region_surface_properties(shape)?.mass()));
    }
    Ok(matched)
}
#[test]
fn analytic_region_seeds_distinguish_nested_regions_and_shared_boundary() {
    let arrangement =
        arrange_region_fixture(&fixture::build_region_fixture("nested").unwrap()).unwrap();
    let builder = build_fixture_regions(&arrangement).unwrap();
    assert_eq!(
        classify_region_seed(&builder, [2., 1.5]).unwrap(),
        vec![(State::In, 2.)]
    );
    assert_eq!(
        classify_region_seed(&builder, [0.5, 0.5]).unwrap(),
        vec![(State::In, 10.)]
    );
    assert!(classify_region_seed(&builder, [8., 8.]).unwrap().is_empty());
    let boundary = classify_region_seed(&builder, [1., 1.5]).unwrap();
    assert_eq!(boundary.len(), 2);
    assert!(boundary.iter().all(|(state, _)| *state == State::On));
}
#[path = "profile_regions/nonempty.rs"]
mod nonempty;
