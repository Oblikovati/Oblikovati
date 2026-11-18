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
    fn accepts_policy(&self, policy: Option<oblikovati_martensite::PrismResolutionPolicy>) -> bool {
        oblikovati_drawing::DrawingProjectionService.accepts_policy(policy)
    }
    fn counts(
        &self,
        _: &DocumentEditHistory,
        _: DocumentGuid,
        _: &DrawingBaseViewDefinition,
    ) -> Result<[usize; 2], String> {
        Err("unexpected separate drawing replay".into())
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
    fn saved_lines(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        view: &DrawingBaseViewDefinition,
    ) -> Result<Vec<SavedDrawingLine>, String> {
        oblikovati_drawing::DrawingProjectionService.saved_lines(history, document, view)
    }
}

#[test]
fn drawing_base_uses_current_part_instead_of_independent_replay() {
    for headless in [false, true] {
        assert_current_projection(headless, true);
    }
    assert_current_projection(true, false);
}

fn assert_current_projection(headless: bool, matching_policy: bool) {
    use oblikovati_api::wire::{AddBaseViewArgs, SetModelReferenceArgs};
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
    let host = if headless {
        host.with_part_update(&mut replay)
    } else {
        host.with_part_topology(&mut owner.part_scenes)
    };
    let result = Client::new(host)
        .drawing_views()
        .add_base(&AddBaseViewArgs {
            center_xmm: 100.,
            center_ymm: 80.,
            ..Default::default()
        });
    if matching_policy {
        assert!(result.unwrap().view.visible_count > 0);
    } else {
        assert!(
            result
                .unwrap_err()
                .to_string()
                .contains("unexpected separate drawing replay")
        );
    }
}
