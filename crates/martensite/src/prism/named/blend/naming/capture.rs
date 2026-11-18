// SPDX-License-Identifier: GPL-2.0-only
//! Preserve exact blend names across an accepted edit.
use super::*;

pub(in crate::prism::named) struct CapturedBlendTopology {
    resolution: Option<PrismResolution>,
    achieved_tolerance: Option<PrismAchievedTolerance>,
    live: Vec<Vec<u8>>,
    deleted: Vec<Vec<u8>>,
    entries: Vec<(Vec<u8>, Shape)>,
    active_body: Vec<u8>,
    primary_body: Vec<u8>,
    shape: Shape,
}

impl CapturedBlendTopology {
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
            bindings,
            active_body: self.active_body,
            primary_body: self.primary_body,
            shape: self.shape,
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
            entries: self.bindings.capture_entries(history, document)?,
            active_body: self.active_body.clone(),
            primary_body: self.primary_body.clone(),
            shape: self.shape.clone(),
        })
    }
}
