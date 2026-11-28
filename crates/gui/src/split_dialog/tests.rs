// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::document_sketch::DocumentSketchId;
use oblikovati_model::feature::hole::HoleOriginPlane;
use oblikovati_model::sketch::SketchEntityId;

#[test]
fn split_dialog_mode_switch_and_all_faces_clear() {
    let mut dialog = SplitDialog::default();
    let state = GuiCommandState {
        document: Some(1),
        document_generation: Some(1),
        browser: Some(crate::GuiModelBrowser {
            bodies: vec![crate::GuiBrowserBody {
                feature: FeatureId(1),
                name: "Solid1".into(),
                key: "0102".into(),
                index: 1,
                faces: 14,
                visible: true,
                appearance: String::new(),
                physical: None,
            }],
            ..Default::default()
        }),
        ..Default::default()
    };
    let mut actions = Vec::new();
    dialog.open(&state, None);
    assert_eq!(dialog.mode, SplitMode::Faces);
    assert!(!dialog.is_complete());

    dialog.set_all_faces(true, &state, &mut actions);
    assert!(dialog.all_faces);
    assert_eq!(dialog.displayed_face_count(), 0);

    dialog.accept_tool(
        SplitTool::SketchCurve {
            sketch: DocumentSketchId(2),
            entity: SketchEntityId(3),
        },
        None,
        0,
        &state,
        &mut actions,
    );
    assert_eq!(dialog.displayed_face_count(), 14);
    assert!(dialog.is_complete());

    dialog.set_all_faces(false, &state, &mut actions);
    assert_eq!(dialog.displayed_face_count(), 0);
    assert!(!dialog.is_complete());

    dialog.switch_mode(SplitMode::Solid, &mut actions);
    assert_eq!(dialog.mode, SplitMode::Solid);
    assert!(dialog.body.is_none());
    assert!(!dialog.is_complete());
    assert_eq!(dialog.keep_side, SplitKeepSide::Both);
}

#[test]
fn split_dialog_parses_origin_and_work_plane_browser_refs() {
    assert_eq!(
        controls::parse_browser_plane_tool("origin/v1/xy"),
        Some(SplitTool::OriginPlane(HoleOriginPlane::Xy))
    );
    assert_eq!(
        controls::parse_browser_plane_tool("datum/v1/7"),
        Some(SplitTool::WorkPlane(FeatureId(7)))
    );
}
