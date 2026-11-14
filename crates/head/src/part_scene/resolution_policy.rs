// SPDX-License-Identifier: GPL-2.0-only
//! Use the modelling bounds shared by desktop and headless hosts.
/// Apply the shared Go ADR-0042 bounds; example: `owner.with_extrusion_policy(desktop_policy())?`.
pub(crate) fn desktop_policy() -> oblikovati_martensite::PrismResolutionPolicy {
    oblikovati_drawing::modelling_policy()
}
