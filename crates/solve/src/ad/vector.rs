// SPDX-License-Identifier: GPL-2.0-only
use super::{DerivativeError, DualNumber};

/// Fixed-size geometric residual vector with compatible, immutable partials.
#[derive(Clone, Debug, PartialEq)]
pub struct DualVector<const N: usize> {
    components: [DualNumber; N],
}

impl<const N: usize> DualVector<N> {
    /// Validate component derivative dimensions; constants may mix with seeded values.
    /// Example: `DualVector::new([x, y])?`.
    pub fn new(components: [DualNumber; N]) -> Result<Self, DerivativeError> {
        let dimensions: Vec<usize> = components
            .iter()
            .map(|value| value.gradient.len())
            .filter(|count| *count != 0)
            .collect();
        if dimensions.windows(2).any(|pair| pair[0] != pair[1]) {
            return Err(DerivativeError(format!(
                "vector gradient dimensions {dimensions:?}: expected one shared seeded dimension"
            )));
        }
        Ok(Self { components })
    }

    /// Inspect components without changing their values or partials; example: `vector.components()[0].value()`.
    pub fn components(&self) -> &[DualNumber; N] {
        &self.components
    }

    /// Add vectors componentwise; example: `position.plus(&offset)?`.
    pub fn plus(&self, other: &Self) -> Result<Self, DerivativeError> {
        self.zip_components(other, DualNumber::plus)
    }

    /// Build a displacement; example: `end.minus(&start)?`.
    pub fn minus(&self, other: &Self) -> Result<Self, DerivativeError> {
        self.zip_components(other, DualNumber::minus)
    }

    /// Scale every component by a constant; example: `direction.scaled(2.0)?`.
    pub fn scaled(&self, factor: f64) -> Result<Self, DerivativeError> {
        self.map_components(|value| value.scaled(factor))
    }

    /// Scale every component by a varying scalar; example: `direction.product(&distance)?`.
    pub fn product(&self, factor: &DualNumber) -> Result<Self, DerivativeError> {
        self.map_components(|value| value.product(factor))
    }

    /// Evaluate a dot-product residual and all its partials; example: `a.dot(&b)?`.
    pub fn dot(&self, other: &Self) -> Result<DualNumber, DerivativeError> {
        self.components
            .iter()
            .zip(&other.components)
            .try_fold(DualNumber::constant(0.0)?, |sum, (a, b)| {
                sum.plus(&a.product(b)?)
            })
    }

    /// Evaluate length without squaring components; zero length uses zero partials.
    /// Example: `displacement.length()?`.
    pub fn length(&self) -> Result<DualNumber, DerivativeError> {
        // Chained hypot keeps representable norms from overflowing or underflowing (#3699).
        self.components
            .iter()
            .try_fold(DualNumber::constant(0.0)?, |norm, value| norm.hypot(value))
    }

    fn map_components(
        &self,
        mut operation: impl FnMut(&DualNumber) -> Result<DualNumber, DerivativeError>,
    ) -> Result<Self, DerivativeError> {
        let mut components = self.components.clone();
        for component in &mut components {
            *component = operation(component)?;
        }
        Self::new(components)
    }

    fn zip_components(
        &self,
        other: &Self,
        operation: impl Fn(&DualNumber, &DualNumber) -> Result<DualNumber, DerivativeError>,
    ) -> Result<Self, DerivativeError> {
        let mut components = self.components.clone();
        for (component, right) in components.iter_mut().zip(&other.components) {
            *component = operation(component, right)?;
        }
        Self::new(components)
    }
}

impl DualVector<2> {
    /// Signed planar cross-product residual; this is not an exact topological predicate.
    /// Example: `direction.cross(&other)?`.
    pub fn cross(&self, other: &Self) -> Result<DualNumber, DerivativeError> {
        let [x, y] = &self.components;
        let [u, v] = &other.components;
        x.product(v)?.minus(&y.product(u)?)
    }
}

impl DualVector<3> {
    /// Spatial cross product and its partials; example: `normal.cross(&axis)?`.
    pub fn cross(&self, other: &Self) -> Result<Self, DerivativeError> {
        let [x, y, z] = &self.components;
        let [u, v, w] = &other.components;
        Self::new([
            y.product(w)?.minus(&z.product(v)?)?,
            z.product(u)?.minus(&x.product(w)?)?,
            x.product(v)?.minus(&y.product(u)?)?,
        ])
    }
}
