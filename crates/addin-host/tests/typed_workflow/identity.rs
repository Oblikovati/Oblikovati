// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::identity::{DocumentGuid, GuidSource, IdentityTransitionError};

pub(super) struct WorkflowGuids(pub(super) u64);
impl GuidSource for WorkflowGuids {
    fn generate(&mut self) -> Result<DocumentGuid, IdentityTransitionError> {
        self.0 += 1;
        Ok(format!("00000000-0000-4000-8000-{:012x}", self.0)
            .parse()
            .unwrap())
    }
}
