// SPDX-License-Identifier: GPL-2.0-only

pub(crate) struct RowScale {
    maximum: f64,
    norm: f64,
}
impl RowScale {
    pub(crate) fn new(row: &[f64]) -> Self {
        let maximum = row
            .iter()
            .fold(0.0_f64, |largest, value| largest.max(value.abs()));
        if maximum == 0.0 {
            return Self {
                maximum: 1.0,
                norm: 1.0,
            };
        }
        // Normalize scaled coordinates, not an overflowing norm or reciprocal (#3702).
        let norm = row
            .iter()
            .fold(0.0_f64, |norm, value| norm.hypot(value / maximum));
        Self { maximum, norm }
    }
    pub(crate) fn normalize(&self, value: f64) -> f64 {
        (value / self.maximum) / self.norm
    }
    pub(crate) fn displacement(&self, value: f64, scale: f64) -> f64 {
        // Opposite units can cancel before division, avoiding an intermediate under/overflow.
        if (self.maximum < 1.0) != (scale < 1.0) {
            return (value / (self.maximum * scale)) / self.norm;
        }
        self.normalize(value) / scale
    }
    pub(crate) fn row(&self, row: &[f64]) -> Vec<f64> {
        row.iter().map(|&value| self.normalize(value)).collect()
    }
}

#[cfg(test)]
mod tests {
    use super::RowScale;
    #[test]
    fn opposite_units_do_not_erase_or_overflow_representable_displacements() {
        assert_eq!(RowScale::new(&[1e300]).displacement(1e-200, 1e-300), 1e-200);
        assert_eq!(RowScale::new(&[1e-300]).displacement(1e200, 1e300), 1e200);
        assert_eq!(RowScale::new(&[0.0]).displacement(2.0, 4.0), 0.5);
        assert_eq!(RowScale::new(&[3.0, 4.0]).row(&[3.0, 4.0]), [0.6, 0.8]);
    }
}
