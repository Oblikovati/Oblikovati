// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{GuiBrowserFeature, GuiBrowserSketch};
fn browser() -> GuiModelBrowser {
    GuiModelBrowser {
        sketches: vec![GuiBrowserSketch {
            id: DocumentSketchId(1),
            inspection: Ok(oblikovati_api::wire::SketchInfo {
                plane: "XY".into(),
                ..Default::default()
            }),
            name: "Sketch1".into(),
            visible: false,
            consumed_by: Some(FeatureId(7)),
            line_style: Default::default(),
            dimensions_shown: true,
        }],
        features: vec![GuiBrowserFeature {
            id: FeatureId(7),
            name: "Extrude1".into(),
            suppressed: false,
            extended: None,
            kind: "Extrusion",
        }],
        ..Default::default()
    }
}
fn origin(name: &str) -> RelatedRow {
    origin_row_named(name).unwrap()
}
#[test]
fn a_sketch_stands_on_its_plane_and_center_point_and_makes_its_feature() {
    let (parents, children) = relations_of(&browser(), RelatedRow::Sketch(DocumentSketchId(1)));
    assert_eq!(parents, [origin("XY Plane"), origin(CENTER_POINT)]);
    assert_eq!(children, [RelatedRow::Feature(FeatureId(7))]);
}
#[test]
fn a_feature_comes_from_a_plane_and_its_sketch_and_nothing_comes_from_it() {
    let (parents, children) = relations_of(&browser(), RelatedRow::Feature(FeatureId(7)));
    assert_eq!(
        parents,
        [origin("XY Plane"), RelatedRow::Sketch(DocumentSketchId(1))]
    );
    assert!(children.is_empty());
}
#[test]
fn only_the_plane_a_sketch_stands_on_has_children() {
    let (parents, children) = relations_of(&browser(), origin("XY Plane"));
    assert!(parents.is_empty());
    assert_eq!(
        children,
        [
            RelatedRow::Sketch(DocumentSketchId(1)),
            RelatedRow::Feature(FeatureId(7))
        ]
    );
    let (parents, children) = relations_of(&browser(), origin("YZ Plane"));
    assert!(parents.is_empty() && children.is_empty());
}
#[test]
fn a_row_reads_as_its_browser_name() {
    assert_eq!(row_name(&browser(), origin("XZ Plane")), "XZ Plane");
    assert_eq!(
        row_name(&browser(), RelatedRow::Feature(FeatureId(7))),
        "Extrude1"
    );
    assert_eq!(row_name(&browser(), RelatedRow::Feature(FeatureId(9))), "");
}

#[test]
fn document_change_closes_relationships_before_docking() {
    let mut dialog = RelationshipsDialog::default();
    dialog.request(RelatedRow::Origin(0));
    dialog.bind_document(Some(1));
    let state = GuiCommandState {
        document: Some(2),
        ..Default::default()
    };
    dialog.sync_before_docking(&state, false);
    assert!(dialog.shown.is_none());
    assert!(!dialog.dock_state.is_requested());
}
