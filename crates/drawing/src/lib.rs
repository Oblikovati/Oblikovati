// SPDX-License-Identifier: GPL-2.0-only
//! Shared projection for drawing display and saved view presentation.
mod policy;
mod projection;
pub use policy::{DISPLAY_MESH, modelling_policy};
pub use projection::{
    DrawingProjectionService, evaluate_part, project_evaluated_part, sample_projection,
};
