// SPDX-License-Identifier: GPL-2.0-only
//! A drawing view consumes the source document's current part evaluation.
use super::*;
use oblikovati_automation::{DrawingViewProjectionAccess, PartUpdateAccess};
use oblikovati_martensite::{EvaluationFailure, NamedSolid};
use oblikovati_model::feature::part::PartFeatureEvaluation;
use oblikovati_model::{
    drawing::DrawingBaseViewDefinition, history::DocumentEditHistory, identity::DocumentGuid,
};
use oblikovati_persistence::package::SavedDrawingLine;

struct CurrentOnlyProjection;
impl DrawingViewProjectionAccess for CurrentOnlyProjection {
    fn resolution_policy(&self) -> oblikovati_martensite::PrismResolutionPolicy {
        oblikovati_drawing::modelling_policy()
    }
    fn saved_lines_current(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        view: &DrawingBaseViewDefinition,
        part: &PartFeatureEvaluation<NamedSolid, EvaluationFailure>,
    ) -> Result<Vec<SavedDrawingLine>, String> {
        oblikovati_drawing::DrawingProjectionService
            .saved_lines_current(history, document, view, part)
    }
    fn accepts_policy(&self, policy: Option<oblikovati_martensite::PrismResolutionPolicy>) -> bool {
        oblikovati_drawing::DrawingProjectionService.accepts_policy(policy)
    }
    fn counts_current(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        view: &DrawingBaseViewDefinition,
        part: &PartFeatureEvaluation<NamedSolid, EvaluationFailure>,
    ) -> Result<[usize; 2], String> {
        oblikovati_drawing::DrawingProjectionService.counts_current(history, document, view, part)
    }
}

#[test]
fn drawing_base_uses_current_part_instead_of_independent_replay() {
    for headless in [false, true] {
        assert_current_projection(headless, true, true);
    }
    assert_current_projection(true, false, true);
    assert_current_projection(true, false, false);
}

fn assert_current_projection(headless: bool, matching_policy: bool, injected: bool) {
    use oblikovati_api::wire::{
        AddBaseViewArgs, BatchSaveArgs, BatchSaveItem, SaveDocumentAsArgs, SetModelReferenceArgs,
    };
    use oblikovati_gui::{GuiDocumentTemplate, GuiTemplateKind, GuiTemplateUnits};
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let part = super::super::part_scene::open_prism(&mut owner, false);
    owner
        .part_scenes
        .ensure_part(&mut owner.files, part)
        .unwrap();
    owner
        .create_document_template(GuiDocumentTemplate {
            kind: GuiTemplateKind::Drawing,
            units: GuiTemplateUnits::Metric,
        })
        .unwrap();
    let drawing = owner.files.workspace().active_document().unwrap();
    let reference = std::env::temp_dir()
        .join("head-scene.opd")
        .to_string_lossy()
        .into_owned();
    Client::new(owner.document_host())
        .drawing()
        .set_model_reference(&SetModelReferenceArgs {
            full_document_name: reference,
        })
        .unwrap();
    let mut replay = if matching_policy {
        oblikovati_automation::PartReplayService::with_policy(
            crate::part_scene::resolution_policy::desktop_policy(),
        )
        .unwrap()
    } else {
        oblikovati_automation::PartReplayService::default()
    };
    if headless {
        replay.ensure_part(&mut owner.files, part).unwrap();
    }
    let host =
        oblikovati_automation::FileHost::new(&mut owner.files, &mut owner.guids, &owner.version)
            .with_drawing_views(&CurrentOnlyProjection);
    let host = if !injected {
        host
    } else if headless {
        host.with_part_update(&mut replay)
    } else {
        host.with_part_topology(&mut owner.part_scenes)
    };
    let client = Client::new(host);
    let result = client.drawing_views().add_base(&AddBaseViewArgs {
        center_xmm: 100.,
        center_ymm: 80.,
        ..Default::default()
    });
    {
        assert!(result.unwrap().view.visible_count > 0);
        client
            .documents()
            .save_as(&SaveDocumentAsArgs {
                document: drawing,
                new_full_document_name: "/memory/current-replay.odd".into(),
            })
            .unwrap();
        let batch = client
            .documents()
            .batch_save(&BatchSaveArgs {
                operation: "saveCopyAs".into(),
                items: Some(vec![BatchSaveItem {
                    document: drawing,
                    target_file_name: "/memory/current-replay-copy.odd".into(),
                }]),
            })
            .unwrap();
        assert_eq!(batch.saved, 1);
    }
    drop(client);
    {
        let bytes = owner
            .files
            .files()
            .saved
            .borrow()
            .get(std::path::Path::new("/memory/current-replay.odd"))
            .cloned()
            .unwrap();
        let saved = oblikovati_persistence::package::DocumentPackage::decode(&bytes)
            .unwrap()
            .restore_saved_drawing_views()
            .unwrap()
            .unwrap();
        assert_eq!(saved.views.len(), 1);
        assert!(!saved.views[0].lines.is_empty());
        owner
            .files
            .commands(part)
            .unwrap()
            .run("Change depth", |snapshot, _| {
                snapshot.parameters.set_expression(3, "3 cm")
            })
            .unwrap();
        let host = oblikovati_automation::FileHost::new(
            &mut owner.files,
            &mut owner.guids,
            &owner.version,
        )
        .with_drawing_views(&CurrentOnlyProjection);
        let host = if !injected {
            host
        } else if headless {
            host.with_part_update(&mut replay)
        } else {
            host.with_part_topology(&mut owner.part_scenes)
        };
        Client::new(host)
            .documents()
            .save_as(&SaveDocumentAsArgs {
                document: drawing,
                new_full_document_name: "/memory/stale-replay.odd".into(),
            })
            .unwrap();
        assert!(
            owner
                .files
                .files()
                .saved
                .borrow()
                .contains_key(std::path::Path::new("/memory/stale-replay.odd"))
        );
        let session = owner.files.workspace().session(part).unwrap();
        let retained = session
            .owned_part_slot(Some(oblikovati_drawing::modelling_policy()))
            .unwrap();
        assert!(retained.current(session).is_ok());
        owner
            .files
            .commands(part)
            .unwrap()
            .run("Invalidate extrusion", |snapshot, _| {
                snapshot.parameters.set_expression(3, "0 cm")
            })
            .unwrap();
        let host = oblikovati_automation::FileHost::new(
            &mut owner.files,
            &mut owner.guids,
            &owner.version,
        )
        .with_drawing_views(&CurrentOnlyProjection);
        assert!(
            Client::new(host)
                .documents()
                .save_as(&SaveDocumentAsArgs {
                    document: drawing,
                    new_full_document_name: "/memory/invalid-replay.odd".into(),
                })
                .is_err()
        );
        assert!(
            !owner
                .files
                .files()
                .saved
                .borrow()
                .contains_key(std::path::Path::new("/memory/invalid-replay.odd"))
        );
    }
}
