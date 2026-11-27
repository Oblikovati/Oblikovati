// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_gui::{GuiExtrudeDirection, GuiExtrudePreview};
use oblikovati_martensite::{PrismFaceRole, PrismPresentation};
use oblikovati_model::history::DocumentSnapshot;

pub(super) fn handle_axis(
    presentation: &PrismPresentation,
    snapshot: &DocumentSnapshot,
    draft: &GuiExtrudePreview,
) -> Option<[[f64; 3]; 2]> {
    let start = cap_center(presentation, PrismFaceRole::Start)?;
    let end = cap_center(presentation, PrismFaceRole::End)?;
    let offset = match draft.direction {
        GuiExtrudeDirection::Positive | GuiExtrudeDirection::Negative => 0.,
        GuiExtrudeDirection::Symmetric => 0.5,
        GuiExtrudeDirection::Asymmetric => {
            let evaluate =
                |value| oblikovati_automation::evaluated_extrude_distance(snapshot, value).ok();
            let a = evaluate(&draft.distance)?;
            let b = evaluate(draft.second_distance.as_ref()?)?;
            b / (a + b)
        }
    };
    let origin = std::array::from_fn(|i| start[i] + offset * (end[i] - start[i]));
    Some([origin, end])
}

fn cap_center(presentation: &PrismPresentation, role: PrismFaceRole) -> Option<[f64; 3]> {
    let face = presentation.faces.iter().find(|face| face.role == role)?;
    let (lo, hi) = face.triangles.iter().flatten().fold(
        ([f64::INFINITY; 3], [f64::NEG_INFINITY; 3]),
        |(lo, hi), point| {
            (
                std::array::from_fn(|i| lo[i].min(point[i])),
                std::array::from_fn(|i| hi[i].max(point[i])),
            )
        },
    );
    let center = std::array::from_fn(|i| (lo[i] + hi[i]) * 0.5);
    center
        .iter()
        .all(|value| value.is_finite())
        .then_some(center)
}
