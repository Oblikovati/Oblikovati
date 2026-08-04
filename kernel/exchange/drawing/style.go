// SPDX-License-Identifier: GPL-2.0-only

package drawing

// Entity formatting carried through a drawing exchange (#2015): the colour, line type and line
// weight the sketch Format panel's three lists expose, and the layer table those values are
// inherited from.
//
// Style lives beside the entities rather than on them, keyed by entity handle, for the same
// reason the sketch's own format table does: the overwhelming majority of entities carry no
// explicit override — they are BYLAYER — so an inline field would be empty on almost every
// entity of a large drawing.

// Colour index sentinels from the DXF/DWG colour model. Values 1–255 are AutoCAD Color Index
// entries; the two sentinels mean "inherit".
const (
	// ColorByBlock inherits from the containing block reference.
	ColorByBlock = 0
	// ColorByLayer inherits from the entity's layer — the default, and by far the common case.
	ColorByLayer = 256
)

// LineWeightByLayer is the line-weight sentinel meaning "inherit from the layer" (DXF group 370
// value -1).
const LineWeightByLayer = -1

// Style is one entity's explicit formatting, as read from the file. Each field may be an
// inherit sentinel, in which case the entity's layer supplies the value.
type Style struct {
	// Color is an AutoCAD Color Index, or ColorByLayer / ColorByBlock to inherit.
	Color int
	// LineType is the line-type record name; "" or "BYLAYER" inherits.
	LineType string
	// LineWeight is in hundredths of a millimetre, or LineWeightByLayer to inherit.
	LineWeight int
}

// Layer is one record of the drawing's layer table: the formatting entities on it inherit.
type Layer struct {
	Name       string
	Color      int
	LineType   string
	LineWeight int
}

// DefaultLayerName is the layer every drawing has and entities fall back to.
const DefaultLayerName = "0"

// ResolveStyle folds an entity's explicit style against its layer, returning the concrete colour
// index, line type and line weight to draw with. An entity that inherits everything resolves
// entirely to its layer's values; an unknown layer resolves to white continuous at the default
// weight, which is what a viewer shows for a missing layer.
//
//	color, lineType, weight := dr.ResolveStyle(handle)
func (d *Drawing) ResolveStyle(handle uint64) (color int, lineType string, weight int) {
	layer := d.layer(d.EntityLayer(handle))
	color, lineType, weight = layer.Color, layer.LineType, layer.LineWeight
	s, ok := d.Styles[handle]
	if !ok {
		return color, lineType, weight
	}
	if s.Color != ColorByLayer && s.Color != ColorByBlock {
		color = s.Color
	}
	if s.LineType != "" && s.LineType != "BYLAYER" {
		lineType = s.LineType
	}
	if s.LineWeight != LineWeightByLayer {
		weight = s.LineWeight
	}
	return color, lineType, weight
}

// layer returns the named layer's record, or the fallback used when the drawing names a layer it
// does not define.
func (d *Drawing) layer(name string) Layer {
	for _, l := range d.Layers {
		if l.Name == name {
			return l
		}
	}
	return Layer{Name: name, Color: defaultLayerColor, LineType: "CONTINUOUS", LineWeight: LineWeightByLayer}
}

// defaultLayerColor is ACI 7 — white on a dark background, black on a light one, which is what a
// layer with no colour of its own shows as.
const defaultLayerColor = 7

// SetStyle records one entity's explicit formatting.
func (d *Drawing) SetStyle(handle uint64, s Style) {
	if d.Styles == nil {
		d.Styles = map[uint64]Style{}
	}
	d.Styles[handle] = s
}

// SetEntityLayer records which layer an entity sits on, so its inherited formatting can be
// resolved. It is kept beside Styles rather than on the entity because only half the entity
// kinds carry a Layer field today, and the resolution needs it for all of them.
func (d *Drawing) SetEntityLayer(handle uint64, layer string) {
	if d.EntityLayers == nil {
		d.EntityLayers = map[uint64]string{}
	}
	d.EntityLayers[handle] = layer
}

// EntityLayer returns the layer an entity sits on, defaulting to the layer every drawing has.
func (d *Drawing) EntityLayer(handle uint64) string {
	if l, ok := d.EntityLayers[handle]; ok && l != "" {
		return l
	}
	return DefaultLayerName
}
