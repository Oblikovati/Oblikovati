// SPDX-License-Identifier: GPL-2.0-only
use super::PlanarCurveFailure;
use martensite_tkgeomalgo::int_res2d::{Position, Situation, Transition, TypeTrans};
/// Native incidence position on an operand's domain.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum CurvePosition {
    Start,
    Interior,
    End,
}
/// Native touch-side classification; unknown is retained rather than guessed.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum TouchSide {
    Inside,
    Outside,
    Unknown,
}
/// Native transitions retain undecided results and touch-only metadata.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum TransitionKind {
    In {
        tangent: bool,
    },
    Out {
        tangent: bool,
    },
    Touch {
        tangent: bool,
        side: TouchSide,
        opposite: bool,
    },
    Undecided,
}
/// A transition belongs to the corresponding input parameter, not to an output index.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct CurveTransition {
    pub position: CurvePosition,
    pub kind: TransitionKind,
}
pub(super) fn convert(native: Transition) -> Result<CurveTransition, PlanarCurveFailure> {
    let position = match native.position_on_curve() {
        Position::Head => CurvePosition::Start,
        Position::Middle => CurvePosition::Interior,
        Position::End => CurvePosition::End,
    };
    Ok(CurveTransition {
        position,
        kind: convert_kind(native)?,
    })
}
fn convert_kind(native: Transition) -> Result<TransitionKind, PlanarCurveFailure> {
    Ok(match native.transition_type() {
        TypeTrans::In => TransitionKind::In {
            tangent: native.is_tangent()?,
        },
        TypeTrans::Out => TransitionKind::Out {
            tangent: native.is_tangent()?,
        },
        TypeTrans::Touch => TransitionKind::Touch {
            tangent: native.is_tangent()?,
            side: touch_side(native.situation()?),
            opposite: native.is_opposite()?,
        },
        TypeTrans::Undecided => TransitionKind::Undecided,
    })
}
fn touch_side(native: Situation) -> TouchSide {
    match native {
        Situation::Inside => TouchSide::Inside,
        Situation::Outside => TouchSide::Outside,
        Situation::Unknown => TouchSide::Unknown,
    }
}

#[cfg(test)]
#[path = "../../tests/unit/planar_curve_transition.rs"]
mod tests;
