// SPDX-License-Identifier: GPL-2.0-only
//! Sketch commands, one folder per scenario feature under `features/sketch/`.
pub(crate) mod command_alias;
pub(crate) mod constraint_glyphs;
#[path = "dimension-edit/mod.rs"]
pub(crate) mod dimension_edit;
#[path = "inspect.rs"]
pub(crate) mod inspect;
#[path = "line/mod.rs"]
pub(crate) mod line;
#[path = "precise-input/mod.rs"]
pub(crate) mod precise_input;
#[path = "region-properties/mod.rs"]
pub(crate) mod region_properties;
