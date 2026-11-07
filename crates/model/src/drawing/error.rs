// SPDX-License-Identifier: GPL-2.0-only
use std::{error::Error, fmt};
/// A drawing recipe or edit was refused before the owner changed.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct DrawingError(pub(super) String);
impl fmt::Display for DrawingError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        output.write_str(&self.0)
    }
}
impl Error for DrawingError {}
