// SPDX-License-Identifier: GPL-2.0-only

//! Application-owned document rules, independent of storage and geometry engines.

pub mod application_options;
pub mod assets;
pub mod attachment;
pub mod attribute;
pub mod dependency;
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
pub mod named_view;

pub mod document_sketch;
pub mod sketch;

pub mod sketch_settings;
