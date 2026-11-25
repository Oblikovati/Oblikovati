// SPDX-License-Identifier: GPL-2.0-only
//! Vector artwork from the Oblikovati design handoff.
use super::IconGlyph;

pub(super) fn source(glyph: IconGlyph) -> Option<&'static str> {
    Some(match glyph {
        IconGlyph::NewPart => include_str!("../../assets/identity/doc-part-16.svg"),
        IconGlyph::NewAssembly => include_str!("../../assets/identity/doc-assembly-16.svg"),
        IconGlyph::NewDrawing => include_str!("../../assets/identity/doc-drawing-16.svg"),
        IconGlyph::Open => include_str!("../../assets/identity/cmd-app-document-open-32.svg"),
        IconGlyph::Save => include_str!("../../assets/identity/cmd-app-save-32.svg"),
        IconGlyph::Home => include_str!("../../assets/identity/cmd-app-home-recent-32.svg"),
        IconGlyph::Customize => {
            include_str!("../../assets/identity/cmd-app-ribbon-customize-32.svg")
        }
        IconGlyph::Projects => {
            include_str!("../../assets/identity/cmd-app-project-settings-32.svg")
        }
        IconGlyph::NewWindow => include_str!("../../assets/identity/cmd-app-new-view-32.svg"),
        IconGlyph::UserInterface => {
            include_str!("../../assets/identity/cmd-app-workspace-layout-32.svg")
        }
        IconGlyph::CreateSketch => include_str!("../../assets/identity/cmd-sketch-2d-32.svg"),
        IconGlyph::Extrude => include_str!("../../assets/identity/cmd-extrude-32.svg"),
        IconGlyph::Revolve => include_str!("../../assets/identity/cmd-revolve-32.svg"),
        IconGlyph::Fillet => include_str!("../../assets/identity/cmd-fillet-32.svg"),
        IconGlyph::Hole => include_str!("../../assets/identity/cmd-hole-32.svg"),
        IconGlyph::Shell => include_str!("../../assets/identity/cmd-shell-32.svg"),
        IconGlyph::WorkPlane => include_str!("../../assets/identity/cmd-work-plane-32.svg"),
        IconGlyph::RectangularPattern => {
            include_str!("../../assets/identity/cmd-pattern-rect-32.svg")
        }
        IconGlyph::BrandMark => include_str!("../../assets/identity/oblikovati-mark.svg"),
        IconGlyph::NavPan => include_str!("../../assets/identity/cmd-app-pan-32.svg"),
        IconGlyph::NavOrbit => include_str!("../../assets/identity/cmd-app-orbit-32.svg"),
        IconGlyph::NavZoom => include_str!("../../assets/identity/cmd-app-zoom-32.svg"),
        IconGlyph::NavWheel => include_str!("../../assets/identity/cmd-app-nav-wheel-32.svg"),
        _ => return None,
    })
}
