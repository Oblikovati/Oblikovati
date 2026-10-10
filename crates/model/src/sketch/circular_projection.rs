// SPDX-License-Identifier: GPL-2.0-only
use super::{circular_frame::CircularRadius, residual::point};
use oblikovati_solve::{ad::DualNumber, residual::ResidualError};

pub(super) enum FrameOutput {
    Centers,
    Radii,
    Frames,
}
pub(super) struct CircularProjection {
    pub prefix: usize,
    pub frames: Vec<CircularRadius>,
    pub output: FrameOutput,
    pub touch: bool,
}
impl CircularProjection {
    pub fn input_count(&self) -> usize {
        self.prefix
            + self
                .frames
                .iter()
                .map(|frame| match frame {
                    CircularRadius::Independent(_) => 3,
                    CircularRadius::Derived(_) => 4,
                })
                .sum::<usize>()
            + usize::from(self.touch) * 2
    }
    pub fn project(&self, variables: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
        let mut projected = variables[..self.prefix].to_vec();
        let frames = self.project_frames(variables)?;
        if !matches!(self.output, FrameOutput::Radii) {
            projected.extend(frames.centers);
        }
        if !matches!(self.output, FrameOutput::Centers) {
            projected.extend(frames.radii);
        }
        if self.touch {
            projected.extend_from_slice(&variables[frames.end..frames.end + 2]);
        }
        Ok(projected)
    }
    fn project_frames(&self, variables: &[DualNumber]) -> Result<ProjectedFrames, ResidualError> {
        let mut frames = ProjectedFrames {
            centers: Vec::new(),
            radii: Vec::new(),
            end: self.prefix,
        };
        for encoding in &self.frames {
            let (radius, width) = radius_at(variables, frames.end, *encoding)?;
            frames
                .centers
                .extend_from_slice(&variables[frames.end..frames.end + 2]);
            frames.radii.push(radius);
            frames.end += width;
        }
        Ok(frames)
    }
}
struct ProjectedFrames {
    centers: Vec<DualNumber>,
    radii: Vec<DualNumber>,
    end: usize,
}
fn radius_at(
    variables: &[DualNumber],
    offset: usize,
    encoding: CircularRadius,
) -> Result<(DualNumber, usize), ResidualError> {
    Ok(match encoding {
        CircularRadius::Independent(_) => (variables[offset + 2].clone(), 3),
        CircularRadius::Derived(_) => (
            point(variables, offset + 2)?
                .minus(&point(variables, offset)?)?
                .length()?,
            4,
        ),
    })
}

#[cfg(test)]
#[path = "../../tests/sketch/circular_projection.rs"]
mod tests;
