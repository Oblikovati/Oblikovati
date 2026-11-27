// SPDX-License-Identifier: GPL-2.0-only
//! Select the native font face for a text object with uniform font styling.
use martensite_tkservice::font::FontFontAspect;
use oblikovati_api::types::{SketchTextRun, SketchTextStyle};

/// Resolve the font aspect, for example `font_aspect(&style)` returns Bold for a fully bold label.
pub(super) fn font_aspect(style: &SketchTextStyle) -> Result<FontFontAspect, String> {
    let runs = style.formatting_runs.as_deref().unwrap_or_default();
    SketchTextRun::validate(Some(&style.content), runs).map_err(|cause| cause.to_string())?;
    if runs.iter().all(|run| !run.bold && !run.italic) {
        return Ok(FontFontAspect::Regular);
    }
    let first = &runs[0];
    if uniform_face(runs, style.content.chars().count()) {
        return Ok(native_face(first.bold, first.italic));
    }
    Err("sketch text font runs: expected one font face until mixed-font layout is available (Martensite #576)".into())
}

fn uniform_face(runs: &[SketchTextRun], length: usize) -> bool {
    let Some(first) = runs.first() else {
        return length == 0;
    };
    let mut next_start = 0;
    for run in runs {
        if run.start != next_start || (run.bold, run.italic) != (first.bold, first.italic) {
            return false;
        }
        next_start = run.end;
    }
    usize::try_from(next_start) == Ok(length)
}

fn native_face(bold: bool, italic: bool) -> FontFontAspect {
    match (bold, italic) {
        (false, false) => FontFontAspect::Regular,
        (true, false) => FontFontAspect::Bold,
        (false, true) => FontFontAspect::Italic,
        (true, true) => FontFontAspect::BoldItalic,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn uniform_face_uses_unicode_positions_and_ignores_other_attributes() {
        let regular = SketchTextStyle {
            content: "c界".into(),
            ..Default::default()
        };
        assert_eq!(font_aspect(&regular), Ok(FontFontAspect::Regular));
        for (bold, italic, aspect) in [
            (true, false, FontFontAspect::Bold),
            (false, true, FontFontAspect::Italic),
            (true, true, FontFontAspect::BoldItalic),
        ] {
            let first = SketchTextRun {
                start: 0,
                end: 1,
                bold,
                italic,
                ..Default::default()
            };
            let second = SketchTextRun {
                start: 1,
                end: 2,
                underline: true,
                ..first.clone()
            };
            let style = SketchTextStyle {
                formatting_runs: Some(vec![first, second]),
                ..regular.clone()
            };
            assert_eq!(font_aspect(&style), Ok(aspect));
        }
    }

    #[test]
    fn mixed_face_layout_declines_instead_of_rendering_as_one_face() {
        let second = SketchTextRun {
            start: 1,
            end: 2,
            bold: true,
            ..Default::default()
        };
        let style = SketchTextStyle {
            content: "c界".into(),
            formatting_runs: Some(vec![second]),
            ..Default::default()
        };
        assert!(font_aspect(&style).unwrap_err().contains("#576"));
    }

    #[cfg(target_os = "linux")]
    #[test]
    fn bold_face_increases_native_glyph_area_at_the_same_size() {
        let regular = SketchTextStyle {
            content: "c".into(),
            ..Default::default()
        };
        let bold = SketchTextStyle {
            formatting_runs: Some(vec![SketchTextRun {
                start: 0,
                end: 1,
                bold: true,
                ..Default::default()
            }]),
            ..regular.clone()
        };
        let plain_area = native_area(font_aspect(&regular).unwrap());
        assert!(plain_area > 0.);
        assert!(native_area(font_aspect(&bold).unwrap()) > plain_area);
    }

    #[cfg(target_os = "linux")]
    fn native_area(aspect: FontFontAspect) -> f64 {
        use martensite_tkernel::ncollection::String8;
        use martensite_tkgeombase::gprop::GProps;
        use martensite_tkservice::font::FontStrictLevel;
        use martensite_tktopalgo::brep_gprop::BRepGProp;
        use martensite_tkv3d::std_prs::StdPrsBRepFont;
        let family = String8::from_unicode_units(b"DejaVu Sans", 11);
        let font = StdPrsBRepFont::from_name(&family, aspect, 3., FontStrictLevel::Aliases);
        let shape = font.render_text(&String8::from_unicode_units(b"c", 1));
        let mut props = GProps::default();
        BRepGProp::surface_properties(&shape, &mut props, false, false).unwrap();
        props.mass()
    }
}
