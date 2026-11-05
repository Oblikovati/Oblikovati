// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSnapshot, HistoryError};

impl DocumentSnapshot {
    pub(super) fn validate_datum_metadata(&self) -> Result<(), HistoryError> {
        self.work_geometry
            .validate_plane_metadata()
            .map_err(HistoryError::DatumMetadata)?;
        self.validate_prism_reads()
            .map_err(HistoryError::PartProgram)
    }
    pub(super) fn validate_snapshot_transition(
        &mut self,
        before: &Self,
    ) -> Result<(), HistoryError> {
        self.refresh_prism_reads()
            .map_err(HistoryError::PartProgram)?;
        self.validate_datum_metadata()?;
        self.validate_scalar_transition(before)
    }
}
