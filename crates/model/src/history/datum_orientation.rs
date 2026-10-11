// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSnapshot, HistoryError};

impl DocumentSnapshot {
    pub(super) fn validate_datum_orientation(&self) -> Result<(), HistoryError> {
        self.work_geometry
            .validate_plane_orientations()
            .map_err(HistoryError::DatumOrientation)
    }
    pub(super) fn validate_snapshot_transition(&self, before: &Self) -> Result<(), HistoryError> {
        self.validate_datum_orientation()?;
        self.validate_scalar_transition(before)
    }
}
