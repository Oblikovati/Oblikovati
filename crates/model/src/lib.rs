// SPDX-License-Identifier: GPL-2.0-only

//! Application-owned document rules, independent of storage and geometry engines.

pub mod application_options;
pub mod application_recent;
pub mod assembly;
pub mod assets;
pub mod attachment;
pub mod attribute;
pub mod dependency;
pub mod drawing;
pub mod expression;
pub mod feature;
pub mod fits;
pub mod history;
pub mod identity;
mod metadata_value;
pub mod parameter;
pub mod property;
pub mod quantity;
pub mod resource;
pub mod units;

pub mod interest;
pub mod keymap;
pub mod marking_menu;
pub mod model_state;
pub mod named_view;

pub mod document_sketch;
pub mod sketch;

pub mod sketch_settings;

pub mod theme;

pub mod color_scheme;

pub mod application_prompts;
pub mod ui_scale;
