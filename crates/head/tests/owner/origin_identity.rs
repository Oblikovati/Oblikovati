// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn gui_origin_sketch_creation_preserves_the_selected_origin_identity() {
    use oblikovati_model::{
        document_sketch::SketchPlacement, feature::work_plane::WorkPlaneOrigin,
    };
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    owner
        .start_sketch_on_plane_from_gui(oblikovati_gui::GuiOriginPlaneKind::Xy)
        .unwrap();
    let snapshot = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot();
    assert_eq!(
        snapshot.sketches.sketches()[0].placement(),
        SketchPlacement::OriginPlane(WorkPlaneOrigin::XY)
    );
}
