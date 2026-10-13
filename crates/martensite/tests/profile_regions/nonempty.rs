// SPDX-License-Identifier: GPL-2.0-only
//! Diagnostic cardinality contract, not the production sketch-region API.
use super::*;
use rustycad_tkbo::bop_algo::{BOPAlgoBuilder, BOPAlgoPaveFiller, BuilderHooks};
struct NonemptyRegionBuilder(BOPAlgoBuilder);
impl BuilderHooks for NonemptyRegionBuilder {
    fn general_fuse(&mut self) -> &mut BOPAlgoBuilder {
        &mut self.0
    }
    fn check_data(&mut self, filler: &BOPAlgoPaveFiller) {
        // PaveFiller already rejects empty/null inputs. Do not change mapped Builder::CheckData.
        self.0.options().report().merge(filler.options().report());
    }
}
fn nonempty_arrangement(sources: &IndexedMapOfShape) -> NonemptyRegionBuilder {
    let mut arguments = List::default();
    for index in 1..=sources.extent() {
        arguments.append(sources.find_key(index).unwrap().clone());
    }
    let mut filler = BOPAlgoPaveFiller::new();
    filler.set_arguments(&arguments);
    filler.set_non_destructive(true);
    filler.perform();
    let mut builder = NonemptyRegionBuilder(BOPAlgoBuilder::new());
    builder.0.set_arguments(&arguments);
    builder.0.set_to_fill_history(true);
    builder.perform_internal1(&mut filler).unwrap();
    builder
}
fn nonempty_faces(builder: &NonemptyRegionBuilder) -> BuilderFace {
    assert!(!builder.0.options().has_errors());
    let split = indexed_region_shapes(builder.0.shape(), ShapeEnum::Edge).unwrap();
    build_regions_from_split(&split).unwrap()
}
fn nonempty_region_reports(
    builder: &NonemptyRegionBuilder,
    sources: &IndexedMapOfShape,
) -> Vec<RegionReport> {
    let mut reports = Vec::new();
    for shape in nonempty_faces(builder).areas().iter() {
        let face = face(shape).unwrap();
        if BRepTopAdaptorFClass2d::new(&face, 1e-9)
            .unwrap()
            .perform_infinite_point()
            .unwrap()
            != State::Out
        {
            continue;
        }
        let edges = indexed_region_shapes(shape, ShapeEnum::Edge).unwrap();
        let lineage = nonempty_source_parents(&edges, sources, builder);
        reports.push(measure_region_geometry(shape, lineage).unwrap());
    }
    reports.sort_by(|a, b| {
        a.area
            .total_cmp(&b.area)
            .then(a.centroid[0].total_cmp(&b.centroid[0]))
            .then(a.centroid[1].total_cmp(&b.centroid[1]))
    });
    reports
}
fn nonempty_source_parents(
    edges: &IndexedMapOfShape,
    sources: &IndexedMapOfShape,
    builder: &NonemptyRegionBuilder,
) -> Vec<usize> {
    let mut found = std::collections::BTreeSet::new();
    for index in 1..=edges.extent() {
        let segment = edges.find_key(index).unwrap();
        let lineage: Vec<usize> = (1..=sources.extent())
            .filter(|&parent| {
                nonempty_parent_matches(sources.find_key(parent).unwrap(), segment, builder)
            })
            .collect();
        assert!(!lineage.is_empty());
        found.extend(lineage);
    }
    found.into_iter().collect()
}
fn nonempty_parent_matches(
    source: &Shape,
    segment: &Shape,
    builder: &NonemptyRegionBuilder,
) -> bool {
    source.is_same(segment)
        || builder
            .0
            .modified(source)
            .unwrap()
            .iter()
            .any(|shape| shape.is_same(segment))
}
#[test]
fn nonempty_cardinality_uses_the_same_pipeline_for_all_eight_profile_inputs() {
    for case in [
        "rectangle",
        "nested",
        "divider",
        "crossing",
        "circle",
        "ellipse",
        "single-circle",
        "single-ellipse",
    ] {
        let source = fixture::build_region_fixture(case).unwrap();
        let edges = indexed_region_shapes(&source, ShapeEnum::Edge).unwrap();
        let builder = nonempty_arrangement(&edges);
        let reports = nonempty_region_reports(&builder, &edges);
        let rows: Vec<Vec<f64>> = include_str!("occt-8.0.0-nonempty.csv")
            .lines()
            .filter_map(|line| line.strip_prefix(&format!("{case},")))
            .map(|line| {
                line.split(',')
                    .map(|value| value.parse().unwrap())
                    .collect()
            })
            .collect();
        assert_eq!(reports.len(), rows.len(), "case {case}");
        assert!(!rows.is_empty());
        for (report, row) in reports.iter().zip(rows) {
            verify_occt_region(report, &row);
        }
    }
}
#[test]
fn empty_inputs_remain_a_named_decline_with_no_output_shape() {
    let builder = nonempty_arrangement(&IndexedMapOfShape::default());
    assert!(builder.0.options().has_errors());
    assert!(builder.0.shape().is_null());
    let mut messages = Vec::new();
    builder.0.options().dump_errors(&mut messages).unwrap();
    assert!(
        String::from_utf8(messages)
            .unwrap()
            .contains("TooFewArguments")
    );
}
