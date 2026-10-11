# Oblikovati

Oblikovati is a parametric mechanical design application. These terms describe
its documents and their relationships.

## Language

**Document**: A part, assembly, drawing, or presentation with its own identity
and saved design content.

**Design project**: The workspace and search locations used to resolve a set
of related documents. A project is not another document kind.

**Recipe**: The parameters, sketches, references, resources, and ordered
features that define a design. Evaluated geometry is a result of the recipe.

**Feature**: A named step in a document's design history. It consumes declared
inputs and produces geometry or other model content.

**Recompute**: Evaluation of the affected recipe after an input changes.

**Occurrence**: One placement of a component in an assembly. Several
occurrences can refer to the same component while keeping distinct identities.

**Reference key**: A persistent name for a model entity in its document and,
where required, occurrence context. It identifies the entity across recompute.

**Provenance**: The generating, modifying, and deleting relationships between
an operation's input and output entities.

**External geometric reference**: A description of a selection from a source
that cannot supply an Oblikovati reference key. It is not a persistent name.

**Working scale**: The physical length of one stored model length unit.
The display unit can change without changing this scale or the stored design.

**Health**: The reported condition of a feature or document, including errors
and warnings that affect evaluation or use of its result.

**Material**: The physical properties assigned to a model, including density
and directional properties where applicable.

**Appearance**: The visual properties assigned to a model. An appearance does
not determine its physical material properties.

**Datum**: A reference plane, axis, or point used to locate design geometry.
A construction datum is tied to its consumers and can be removed when its last
consumer is removed.

**Ground constraint**: A sketch relation that holds each defining point of an
entity at its current position. Grounding an ellipse holds its center, while
its radii and orientation remain free.

**Offset spline**: A derived curve at a signed distance from a named parent spline.
A positive distance is to the left of the parent's forward direction; its
geometric distance dimension measures the magnitude while retaining the side.
