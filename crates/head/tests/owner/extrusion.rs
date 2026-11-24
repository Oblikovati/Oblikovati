// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::wire::featureargs::Extrude;
use oblikovati_gui::GuiExtrudeDirection::Positive;
use oblikovati_martensite::PrismResolution;
use oblikovati_persistence::package::DocumentPackage;
#[path = "extrusion_direction.rs"]
mod direction;
#[path = "extrusion_extent.rs"]
mod extent;
#[path = "extrusion_suppression.rs"]
mod suppression;

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
fn open_sketch(owner: &mut TestOwner) -> u64 {
    let mut snapshot = crate::scene_fixture::selected_prism_document();
    snapshot.work_geometry = Default::default();
    let mut recipe = snapshot
        .sketches
        .get(crate::scene_fixture::SKETCH)
        .unwrap()
        .geometry()
        .recipe();
    for constraint in &mut recipe.constraints {
        if let oblikovati_model::sketch::PlanarConstraint::Distance { parameter, .. } =
            &mut constraint.relation
        {
            *parameter = 1;
        }
    }
    let geometry = oblikovati_model::sketch::Sketch2D::from_recipe(&recipe).unwrap();
    snapshot
        .sketches
        .replace_geometry(crate::scene_fixture::SKETCH, geometry)
        .unwrap();
    snapshot.parameters.set_expression(1, "10 mm").unwrap();
    crate::scene_fixture::solve_selected_sketches(&mut snapshot);
    let bytes = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n")
        .unwrap()
        .with_parameters(&snapshot.parameters, Default::default())
        .unwrap()
        .with_sketches(&snapshot.sketches, 1, 104)
        .unwrap()
        .with_work_geometry(&snapshot.work_geometry, 0)
        .unwrap()
        .encode()
        .unwrap();
    let path = std::env::temp_dir().join("head-extrusion-fixture.opd");
    owner
        .documents()
        .files()
        .saved
        .borrow_mut()
        .insert(path.clone(), bytes);
    Client::new(owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.to_str().unwrap().into(),
            ..Default::default()
        })
        .unwrap()
        .id
}
fn request() -> Extrude {
    Extrude {
        distance: "width".into(),
        operation: "new".into(),
        ..Default::default()
    }
}
#[test]
fn typed_extrude_creates_a_certified_scene_and_tracks_parameter_edits() {
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_resolution(resolution()).unwrap();
    let document = open_sketch(&mut owner);
    let mut input = request();
    input.operation = " New-Body ".into();
    input.extent = " DISTANCE ".into();
    input.direction = " Positive ".into();
    let reply = Client::new(owner.document_host())
        .features()
        .add_feature(&input)
        .unwrap()
        .0
        .unwrap();
    assert_eq!(reply["feature"], "Extrusion1");
    assert_eq!(reply["id"], 1);
    assert_eq!(reply["name"], "Extrusion1");
    assert_eq!(reply["suppressed"], false);
    assert_eq!(reply["health"], "");
    assert_eq!(reply["healthy"], true);
    assert_eq!(reply["bodies"], 1);
    let first = owner.take_scene().unwrap().unwrap();
    assert_eq!(first.status.visible_bodies, 1);
    assert!(!first.scene.triangles.is_empty());
    assert_cube(&owner, document, 1.);
    set_size(&mut owner, "20 mm");
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
    assert_cube(&owner, document, 2.);
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    owner.take_scene().unwrap().unwrap();
    assert_cube(&owner, document, 1.);
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    owner.take_scene().unwrap().unwrap();
    assert_cube(&owner, document, 2.);
    let face_key = cube_face_key(&owner, document);
    let keys = cube_keys(&owner, document);
    let path = "/memory/certified-cube.opd";
    Client::new(owner.document_host())
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document,
            new_full_document_name: path.into(),
        })
        .unwrap();
    Client::new(owner.document_host())
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
            force: true,
        })
        .unwrap();
    let reopened = Client::new(owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.into(),
            ..Default::default()
        })
        .unwrap()
        .id;
    owner.take_scene().unwrap().unwrap();
    assert_cube(&owner, reopened, 2.);
    assert_eq!(cube_face_key(&owner, reopened), face_key);
    assert_eq!(cube_keys(&owner, reopened), keys);
    assert_face(&owner, reopened, &face_key, 4.);
}
#[test]
fn rejected_typed_extrude_does_not_create_a_feature_or_hide_the_sketch() {
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_resolution(resolution()).unwrap();
    let document = open_sketch(&mut owner);
    let mut bad = request();
    bad.operation = "cut".into();
    assert!(
        Client::new(owner.document_host())
            .features()
            .add_feature(&bad)
            .is_err()
    );
    let snapshot = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot();
    assert!(snapshot.work_geometry.features().is_empty());
    assert!(
        snapshot
            .sketches
            .get(crate::scene_fixture::SKETCH)
            .unwrap()
            .visible()
    );
}

fn set_size(owner: &mut TestOwner, expression: &str) {
    Client::new(owner.document_host())
        .parameters()
        .set(&ParameterSetArgs {
            name: "width".into(),
            expression: expression.into(),
            ..Default::default()
        })
        .unwrap();
}
fn assert_cube(owner: &TestOwner, document: u64, side: f64) {
    let workspace = owner.documents().workspace();
    let session = workspace.session(document).unwrap();
    let guid = workspace.identity(document).unwrap().internal_name;
    let evaluated = owner.part_scenes.current(session).unwrap();
    let body = evaluated
        .body(oblikovati_model::feature::FeatureId(1))
        .unwrap();
    assert!(body.as_prism().unwrap().certification().is_some());
    let key = body.body_reference_key(session.history(), guid).unwrap();
    let report = body
        .as_prism()
        .unwrap()
        .body_report(session.history(), guid, key)
        .unwrap();
    assert!((report.properties.volume - side.powi(3)).abs() < 1e-8);
    assert_eq!(
        (
            report.topology.faces,
            report.topology.edges,
            report.topology.vertices
        ),
        (6, 12, 8)
    );
    let keys = body.reference_keys(session.history(), guid).unwrap();
    let face_keys: Vec<_> = keys
        .iter()
        .filter(|candidate| candidate.as_slice() != key)
        .collect();
    assert_eq!(face_keys.len(), 6);
    for face_key in face_keys {
        let face = body
            .as_prism()
            .unwrap()
            .face_report(session.history(), guid, face_key)
            .unwrap();
        assert_eq!(face.loops, 1);
        assert!((face.area - side.powi(2)).abs() < 1e-8);
    }
    assert_face(
        owner,
        document,
        &cube_face_key(owner, document),
        side.powi(2),
    );
}
fn cube_keys(owner: &TestOwner, document: u64) -> Vec<Vec<u8>> {
    let workspace = owner.documents().workspace();
    let session = workspace.session(document).unwrap();
    let guid = workspace.identity(document).unwrap().internal_name;
    owner
        .part_scenes
        .current(session)
        .unwrap()
        .body(oblikovati_model::feature::FeatureId(1))
        .unwrap()
        .reference_keys(session.history(), guid)
        .unwrap()
}
fn cube_face_key(owner: &TestOwner, document: u64) -> Vec<u8> {
    let guid = owner
        .documents()
        .workspace()
        .identity(document)
        .unwrap()
        .internal_name;
    oblikovati_model::identity::SemanticReferenceName::new(
        guid,
        1,
        1,
        "sketch-prism",
        "start",
        &[1],
    )
    .unwrap()
    .encode()
    .unwrap()
}
fn assert_face(owner: &TestOwner, document: u64, key: &[u8], area: f64) {
    let workspace = owner.documents().workspace();
    let session = workspace.session(document).unwrap();
    let guid = workspace.identity(document).unwrap().internal_name;
    let evaluated = owner.part_scenes.current(session).unwrap();
    let body = evaluated
        .body(oblikovati_model::feature::FeatureId(1))
        .unwrap();
    let face = body
        .as_prism()
        .unwrap()
        .face_report(session.history(), guid, key)
        .unwrap();
    assert!((face.area - area).abs() < 1e-8);
}

struct RejectExtrusionCandidate;
impl oblikovati_automation::ExtrudeRegionAccess for RejectExtrusionCandidate {
    fn replay_policy(&self) -> Option<oblikovati_martensite::PrismResolutionPolicy> {
        Some(oblikovati_martensite::PrismResolutionPolicy::Fixed(
            resolution(),
        ))
    }
    fn region(
        &self,
        _: &oblikovati_model::history::DocumentSnapshot,
        _: oblikovati_model::document_sketch::DocumentSketchId,
        _: usize,
    ) -> Result<oblikovati_model::feature::sketch_prism::SketchPrismSelection, String> {
        Ok(crate::scene_fixture::selected_sketch_boundary())
    }
    fn validate(
        &self,
        _: &oblikovati_model::history::DocumentSnapshot,
        _: oblikovati_model::identity::DocumentGuid,
        _: oblikovati_model::feature::FeatureId,
    ) -> Result<usize, String> {
        Err("candidate rejected by test geometry service".into())
    }
}
#[test]
fn rejected_candidate_preserves_the_document_and_sketch_visibility() {
    let (mut owner, _) = owner();
    let document = open_sketch(&mut owner);
    let policy = oblikovati_martensite::PrismResolutionPolicy::Fixed(resolution());
    let mut replay = oblikovati_automation::PartReplayService::with_policy(policy).unwrap();
    oblikovati_automation::PartUpdateAccess::ensure_part(&mut replay, &mut owner.files, document)
        .unwrap();
    let session = owner.documents().workspace().session(document).unwrap();
    let generation = session.generation();
    let history = session
        .undo_labels()
        .into_iter()
        .map(str::to_owned)
        .collect::<Vec<_>>();
    let backend = RejectExtrusionCandidate;
    let result = Client::new(owner.document_host().with_extrusion(&backend))
        .features()
        .add_feature(&request());
    assert!(
        result
            .unwrap_err()
            .to_string()
            .contains("candidate rejected")
    );
    let session = owner.documents().workspace().session(document).unwrap();
    assert_eq!(session.generation(), generation);
    assert_eq!(session.undo_labels(), history);
    let snapshot = session.snapshot();
    assert!(snapshot.work_geometry.features().is_empty());
    assert!(
        snapshot
            .sketches
            .get(crate::scene_fixture::SKETCH)
            .unwrap()
            .visible()
    );
}

#[test]
fn extrusion_preview_is_transient_and_commit_reuses_the_same_feature_identity() {
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_resolution(resolution()).unwrap();
    let document = open_sketch(&mut owner);
    let generation = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    let preview = oblikovati_gui::GuiExtrudePreview {
        generation: None,
        document,
        sketch: crate::scene_fixture::SKETCH,
        profile: 0,
        distance: "width".into(),
        second_distance: None,
        direction: Default::default(),
    };
    assert!(!owner.gui_command_state().unwrap().extrude_preview_active);
    assert!(owner.extrusion_preview_from_gui(Some(&preview)).unwrap());
    assert!(owner.gui_command_state().unwrap().extrude_preview_active);
    let shown = owner.take_scene().unwrap().unwrap();
    assert!(!shown.scene.triangles.is_empty());
    assert_eq!(shown.status.visible_bodies, 0);
    let session = owner.documents().workspace().session(document).unwrap();
    assert_eq!(session.generation(), generation);
    assert!(session.snapshot().work_geometry.features().is_empty());
    assert!(
        session
            .snapshot()
            .sketches
            .get(crate::scene_fixture::SKETCH)
            .unwrap()
            .visible()
    );
    assert!(owner.extrusion_preview_from_gui(None).unwrap());
    assert!(!owner.gui_command_state().unwrap().extrude_preview_active);
    assert!(
        owner
            .take_scene()
            .unwrap()
            .unwrap()
            .scene
            .triangles
            .is_empty()
    );
    assert!(
        owner
            .extrude_from_gui(
                document,
                crate::scene_fixture::SKETCH,
                0,
                "width",
                Positive,
                None
            )
            .unwrap()
    );
    owner.take_scene().unwrap().unwrap();
    assert_cube(&owner, document, 1.);
}

#[test]
fn invalid_or_stale_extrusion_preview_clears_the_old_scene() {
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_resolution(resolution()).unwrap();
    let document = open_sketch(&mut owner);
    let mut preview = oblikovati_gui::GuiExtrudePreview {
        generation: None,
        document,
        sketch: crate::scene_fixture::SKETCH,
        profile: 0,
        distance: "width".into(),
        second_distance: None,
        direction: Default::default(),
    };
    owner.extrusion_preview_from_gui(Some(&preview)).unwrap();
    owner.take_scene().unwrap().unwrap();
    preview.distance = "0 mm".into();
    assert!(owner.extrusion_preview_from_gui(Some(&preview)).is_err());
    assert!(
        owner
            .take_scene()
            .unwrap()
            .unwrap()
            .scene
            .triangles
            .is_empty()
    );
    preview.distance = "width".into();
    owner.extrusion_preview_from_gui(Some(&preview)).unwrap();
    owner.take_scene().unwrap().unwrap();
    set_size(&mut owner, "20 mm");
    assert!(
        owner
            .take_scene()
            .unwrap()
            .unwrap()
            .scene
            .triangles
            .is_empty()
    );
}

#[path = "extrusion_close.rs"]
mod close;

#[path = "extrusion_policy.rs"]
mod policy;

#[path = "extrusion_topology.rs"]
mod topology;

#[path = "extrusion_edit.rs"]
mod edit;

#[path = "extrusion_replay.rs"]
mod replay;
