// SPDX-License-Identifier: GPL-2.0-only

/// Internal parameter categories; these are not API IDs or a storage encoding.
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum Dimension {
    Unitless,
    Length,
    Angle,
    Area,
    Volume,
    Mass,
    Time,
    Boolean,
    Text,
}

const ARITHMETIC: [(Dimension, [i8; 4]); 7] = [
    (Dimension::Unitless, [0, 0, 0, 0]),
    (Dimension::Length, [1, 0, 0, 0]),
    (Dimension::Angle, [0, 1, 0, 0]),
    (Dimension::Area, [2, 0, 0, 0]),
    (Dimension::Volume, [3, 0, 0, 0]),
    (Dimension::Mass, [0, 0, 1, 0]),
    (Dimension::Time, [0, 0, 0, 1]),
];

impl Dimension {
    /// Identify numeric categories. Example: `assert!(Dimension::Length.is_arithmetic());`
    pub fn is_arithmetic(self) -> bool {
        self.exponents().is_some()
    }

    fn exponents(self) -> Option<[i8; 4]> {
        ARITHMETIC
            .iter()
            .find(|(unit, _)| *unit == self)
            .map(|(_, powers)| *powers)
    }

    pub(super) fn combine(self, other: Self, sign: i8) -> Option<Self> {
        let left = self.exponents()?;
        let right = other.exponents()?;
        let combined = std::array::from_fn(|index| left[index] + sign * right[index]);
        ARITHMETIC
            .iter()
            .find(|(_, powers)| *powers == combined)
            .map(|(unit, _)| *unit)
    }
}
