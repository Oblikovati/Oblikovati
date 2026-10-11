// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[derive(Clone, Copy, Debug)]
pub(in crate::datums) enum OwnedDatumScalar {
    Offset,
    Angle,
}
impl OwnedDatumScalar {
    pub(in crate::datums) fn role(self) -> &'static str {
        match self {
            Self::Offset => "offset",
            Self::Angle => "angle",
        }
    }
    pub(in crate::datums) fn dimension(self) -> Dimension {
        match self {
            Self::Offset => Dimension::Length,
            Self::Angle => Dimension::Angle,
        }
    }
    pub(in crate::datums) fn validate(
        self,
        snapshot: &DocumentSnapshot,
        source: &str,
    ) -> Result<(), Error> {
        snapshot
            .parameters
            .evaluate_scalar_expression(source, self.dimension())
            .map(|_| ())
            .map_err(api_error)
    }
    pub(in crate::datums) fn owned_parameter(
        self,
        snapshot: &DocumentSnapshot,
        owner: FeatureId,
    ) -> Result<Option<u64>, Error> {
        Ok(snapshot
            .work_geometry
            .get(owner)
            .map_err(api_error)?
            .owned_scalars()
            .iter()
            .find(|binding| binding.role == self.role())
            .map(|binding| binding.parameter))
    }
    pub(in crate::datums) fn source_unchanged(
        self,
        snapshot: &DocumentSnapshot,
        owner: FeatureId,
        expression: &str,
    ) -> Result<bool, Error> {
        let Some(id) = self.owned_parameter(snapshot, owner)? else {
            return Ok(false);
        };
        let Some(parameter) = snapshot.parameters.get(id) else {
            return Ok(false);
        };
        // #3759: identical text can acquire another physical value in a new display-unit context.
        let context = oblikovati_model::parameter::DimensionExpressionContext::new(
            snapshot.parameters.units(),
            self.dimension(),
        )
        .map_err(api_error)?;
        Ok(parameter.source == expression
            && parameter.expression_context.as_ref() == Some(&context))
    }
    pub(in crate::datums) fn insert(
        self,
        snapshot: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
        owner: FeatureId,
        source: &str,
    ) -> Result<u64, Error> {
        let id = ids.allocate().map_err(api_error)?;
        let name = format!("feature{}_{}", owner.0, self.role());
        snapshot
            .parameters
            .insert_scalar_expression(id, &name, source, self.dimension())
            .map_err(api_error)?;
        snapshot.parameters.model_value(id).map_err(api_error)?;
        Ok(id)
    }
    pub(in crate::datums) fn revise(
        self,
        snapshot: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
        owner: FeatureId,
        source: &str,
    ) -> Result<(u64, bool), Error> {
        if let Some(parameter) = self.owned_parameter(snapshot, owner)? {
            snapshot
                .parameters
                .set_scalar_expression(parameter, source, self.dimension())
                .map_err(api_error)?;
            snapshot
                .parameters
                .model_value(parameter)
                .map_err(api_error)?;
            return Ok((parameter, false));
        }
        // Legacy unowned drivers remain untouched when a feature first acquires an authored scalar (#3745).
        self.insert(snapshot, ids, owner, source)
            .map(|id| (id, true))
    }
}
