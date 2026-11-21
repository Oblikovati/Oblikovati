// SPDX-License-Identifier: GPL-2.0-only
//! Compose the same file and drawing services for headless commands and scripts.
use super::observations::{CliObservationFactory, CliObservations};
use oblikovati_automation::FileHost;
use oblikovati_drawing::DrawingProjectionService;
use oblikovati_model::identity::GuidSource;
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::files::FileWorkspace;

pub(super) type CliFiles<F> = FileWorkspace<CliObservations, CliObservations, F>;

pub(super) fn host<'a, F: DocumentFiles>(
    files: &'a mut CliFiles<F>,
    guids: &'a mut dyn GuidSource,
    version: &'a str,
    factory: &'a mut CliObservationFactory,
    projection: &'a DrawingProjectionService,
) -> FileHost<'a, CliObservations, CliObservations, F> {
    FileHost::new(files, guids, version)
        .with_observer_factory(factory)
        .with_drawing_views(projection)
}
