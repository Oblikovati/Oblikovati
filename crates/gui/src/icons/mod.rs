// SPDX-License-Identifier: GPL-2.0-only
mod identity_paints;
mod identity_sources;
mod raster;
mod svg_roles;
pub(crate) use raster::{IconRoleMasks, rasterize_roles};

mod compose;

#[cfg(test)]
#[path = "../../tests/icons/roles.rs"]
mod tests;

mod catalog;

mod cache;
pub(crate) use cache::{GuiIconCache, GuiRibbonIcons, IconSize};
pub(crate) use catalog::{IconGlyph, document_glyph};

mod buttons;
pub(crate) use buttons::action_glyph;
