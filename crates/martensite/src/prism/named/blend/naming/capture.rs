// SPDX-License-Identifier: GPL-2.0-only
//! Preserve exact blend names across an accepted edit.
use super::*;

pub(in crate::prism::named) struct CapturedBlendTopology {
    resolution: Option<PrismResolution>,
    achieved_tolerance: Option<PrismAchievedTolerance>,
    live: Vec<Vec<u8>>,
    deleted: Vec<Vec<u8>>,
    splits: SplitTopologyNames,
    entries: Vec<(Vec<u8>, Shape)>,
    active_body: Vec<u8>,
    primary_body: Vec<u8>,
    shape: Shape,
    thread_overlays: Vec<crate::DisplayTriangle>,
    mark_overlays: Vec<crate::DisplayTriangle>,
}

impl CapturedBlendTopology {
    pub(in crate::prism::named) fn prepare(
        mut self,
        candidate: &oblikovati_model::history::PreparedPrivateEdit<'_>,
        dependent: Option<&oblikovati_model::history::PreparedDependentRefresh<'_>>,
        document: DocumentGuid,
    ) -> Result<
        (
            oblikovati_model::history::PreparedValue<Self>,
            oblikovati_model::identity::PreparedExactReferenceBindings<Shape>,
        ),
        String,
    > {
        let entries = std::mem::take(&mut self.entries);
        let bindings =
            crate::prism::named::captured::prepare_index(candidate, dependent, document, entries)?;
        Ok((
            oblikovati_model::history::PreparedValue::new(candidate, self),
            bindings,
        ))
    }

    pub(in crate::prism::named) fn assemble(
        self,
        bindings: ExactReferenceBindings<Shape>,
    ) -> BlendNamedTopology {
        BlendNamedTopology {
            resolution: self.resolution,
            achieved_tolerance: self.achieved_tolerance,
            live: self.live,
            deleted: self.deleted,
            splits: self.splits,
            bindings,
            active_body: self.active_body,
            primary_body: self.primary_body,
            shape: self.shape,
            thread_overlays: self.thread_overlays,
            mark_overlays: self.mark_overlays,
        }
    }

    pub(in crate::prism::named) fn restore(
        self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<BlendNamedTopology, String> {
        let expected = self.entries.clone();
        let bindings = ExactReferenceBindings::from_entries(
            document,
            history.evaluation_scope(),
            self.entries,
        )?;
        for (key, shape) in expected {
            let kind = ReferenceEnvelope::decode(&key)
                .map_err(|error| error.to_string())?
                .kind;
            if !bindings
                .bind(history, document, &key, kind)?
                .is_same(&shape)
            {
                return Err(format!("blend name {key:?}: expected the captured entity"));
            }
        }
        Ok(BlendNamedTopology {
            resolution: self.resolution,
            achieved_tolerance: self.achieved_tolerance,
            live: self.live,
            deleted: self.deleted,
            splits: self.splits,
            bindings,
            active_body: self.active_body,
            primary_body: self.primary_body,
            shape: self.shape,
            thread_overlays: self.thread_overlays,
            mark_overlays: self.mark_overlays,
        })
    }
}

impl BlendNamedTopology {
    pub(in crate::prism::named) fn capture_detached(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<CapturedBlendTopology, String> {
        Ok(CapturedBlendTopology {
            resolution: self.resolution,
            achieved_tolerance: self.achieved_tolerance,
            live: self.live.clone(),
            deleted: self.deleted.clone(),
            splits: self.splits.clone(),
            entries: self.bindings.capture_entries(history, document)?,
            active_body: self.active_body.clone(),
            primary_body: self.primary_body.clone(),
            shape: self.shape.clone(),
            thread_overlays: self.thread_overlays.clone(),
            mark_overlays: self.mark_overlays.clone(),
        })
    }
}
