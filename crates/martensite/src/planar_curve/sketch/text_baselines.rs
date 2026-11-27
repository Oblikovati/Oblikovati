// SPDX-License-Identifier: GPL-2.0-only
//! Apply sketch line spacing to the native glyph plan before it builds the outlines.
use martensite_tkmath::gp::Xy;
use martensite_tkv3d::brep_font::{
    BRepFontBuilder, BRepFontBuilderTextPlan, BRepFontBuilderTextPlanItem,
};

/// Set baseline advance on a plan aligned to `Topfirstline`, for example from 3 to 6 units.
pub(super) fn set_baseline_advance(
    plan: BRepFontBuilderTextPlan,
    natural: f64,
    advance: f64,
) -> Result<BRepFontBuilderTextPlan, String> {
    require_valid_advance(natural, advance)?;
    if natural == advance {
        return Ok(plan);
    }
    let regions = (0..plan.nb_regions())
        .map(|index| plan.region_handle(index).clone())
        .collect();
    let items = plan
        .items()
        .iter()
        .map(|item| move_baseline(item, natural, advance));
    let top = *plan.upper_right();
    let bottom = Xy::new(
        plan.lower_left().x(),
        top.y() - (plan.height() / natural).round() * advance,
    );
    BRepFontBuilder::create_text_plan(
        plan.size(),
        plan.tolerance(),
        bottom,
        top,
        regions,
        items.collect(),
    )
    .ok_or_else(|| {
        format!(
            "sketch text baseline advance {advance:?}: expected finite glyph positions and bounds"
        )
    })
}

fn require_valid_advance(natural: f64, advance: f64) -> Result<(), String> {
    if [natural, advance]
        .into_iter()
        .all(|gap| gap > 0. && gap.is_finite())
    {
        return Ok(());
    }
    Err(format!(
        "sketch text baseline advances {natural:?} and {advance:?}: expected positive finite values"
    ))
}

fn move_baseline(
    item: &BRepFontBuilderTextPlanItem,
    natural: f64,
    advance: f64,
) -> BRepFontBuilderTextPlanItem {
    let old = item.position();
    // Martensite accumulates rows in f32. Recover the row index while retaining empty rows.
    let row = (old.y() / natural).round();
    BRepFontBuilderTextPlanItem::new(item.region(), Xy::new(old.x(), row * advance))
}

// The Linux test lane installs DejaVu Sans for the native glyph checks.
#[cfg(all(test, target_os = "linux"))]
mod tests {
    use super::*;
    use martensite_tkernel::ncollection::String8;
    use martensite_tkservice::{
        font::{FontFontAspect, FontStrictLevel},
        graphic3d::{
            Graphic3dHorizontalTextAlignment as AlignX, Graphic3dVerticalTextAlignment as AlignY,
        },
    };
    use martensite_tkv3d::std_prs::StdPrsBRepFont;

    #[test]
    fn multiline_spacing_preserves_empty_rows_and_shared_glyphs() {
        let font = StdPrsBRepFont::from_name(
            &String8::from_unicode_units(b"DejaVu Sans", 11),
            FontFontAspect::Regular,
            3.05,
            FontStrictLevel::Aliases,
        );
        let plan = font
            .plan_text(
                &String8::from_unicode_units(b"O\n\nO", 4),
                AlignX::Center,
                AlignY::Topfirstline,
            )
            .unwrap();
        let natural = font.line_spacing();
        let advance = natural * 2.;
        let spaced = set_baseline_advance(plan.clone(), natural, advance).unwrap();
        assert_eq!(spaced.items().len(), 2);
        assert_eq!(spaced.items()[0].position().y(), 0.);
        assert_eq!(spaced.items()[1].position().y(), -2. * advance);
        assert_eq!(
            spaced.items()[1].position().x(),
            plan.items()[1].position().x()
        );
        assert!(std::sync::Arc::ptr_eq(
            spaced.region_handle(0),
            plan.region_handle(0)
        ));
        assert!(!font.render_plan(&spaced).is_null());
        let same = set_baseline_advance(plan.clone(), natural, natural).unwrap();
        assert_eq!(same.items(), plan.items());
        assert!(set_baseline_advance(plan, natural, f64::INFINITY).is_err());
    }
}
