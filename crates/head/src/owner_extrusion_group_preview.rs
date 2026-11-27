// SPDX-License-Identifier: GPL-2.0-only
//! Render all the New Solid draft outputs against the current document prefix.
use super::*;

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn set_grouped_extrusion_preview(
        &mut self,
        draft: &oblikovati_gui::GuiExtrudePreview,
        request: &oblikovati_api::wire::featureargs::Extrude,
    ) -> Result<bool, String> {
        let prepared = self.prepare_grouped_extrusion(draft, request)?;
        let policy = self
            .extrusion_access
            .as_ref()
            .ok_or("expected certified Extrude access")?
            .0;
        let document = draft.document;
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let (scene, axis) = session
            .preview_projected(
                |candidate, ids| prepared.create(candidate, ids),
                |candidate, feature| {
                    use oblikovati_api::Error;
                    let evaluated = workspace
                        .preview_owned_part(document, Some(policy), candidate)
                        .map_err(|cause| Error(cause.to_string()))?
                        .evaluation;
                    let mut presentations = evaluated
                        .bodies_for_feature(feature)
                        .map(|body| extrusion_preview_presentation(body, feature));
                    let primary = presentations.next().ok_or_else(|| {
                        Error(format!(
                            "Extrude preview {feature:?}: expected named bodies, health {:?}",
                            evaluated.health()
                        ))
                    })??;
                    let scene = |presentation: &_| {
                        oblikovati_viewport::ViewportScene::from_prism(
                            presentation,
                            [0., 0., 1.],
                            crate::part_scene::NEW_SOLID_PREVIEW,
                        )
                        .map_err(Error)
                    };
                    let mut combined = scene(&primary)?;
                    for presentation in presentations {
                        combined.append_triangles(scene(&presentation?)?);
                    }
                    let axis = if matches!(
                        draft.reference_extent,
                        None | Some(
                            oblikovati_gui::GuiExtrudeReferenceExtent::DistanceFromPlane { .. }
                        )
                    ) {
                        handle_axis(&primary, session.snapshot(), draft)
                    } else {
                        None
                    };
                    Ok::<_, Error>((combined, axis))
                },
            )
            .map_err(|cause| cause.to_string())?;
        self.extrusion_preview =
            Some((document, session.generation(), scene, PreviewKind::Additive));
        self.extrusion_handle_axis = axis;
        self.extrusion_draft_preview = Some(draft.clone());
        self.part_scenes.invalidate_presentation();
        Ok(true)
    }
}
