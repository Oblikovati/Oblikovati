// SPDX-License-Identifier: GPL-2.0-only
use crate::encode;
use oblikovati_api::Error;
use oblikovati_model::sketch_settings::inference_options_view;
use oblikovati_session::{
    SketchInferenceProjectionInput, TransactionObserver,
    workspace::{WorkspaceObserver, WorkspaceSession},
};
pub(super) fn query_inference_preferences<O: TransactionObserver, W: WorkspaceObserver>(
    workspace: &WorkspaceSession<O, W>,
) -> Result<Vec<u8>, Error> {
    let settings = workspace
        .sketch_inference_settings()
        .map_err(|error| Error(error.to_string()))?;
    encode(&inference_options_view(&settings))
}
pub(super) fn inference_reply(
    candidate: SketchInferenceProjectionInput<'_>,
) -> Result<Vec<u8>, Error> {
    let settings = candidate.map_err(|error| Error(error.to_string()))?;
    encode(&inference_options_view(settings))
}
