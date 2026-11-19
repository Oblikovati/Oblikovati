# Sketch constraint glyphs

In an active 2D sketch, press **F8** to show all geometric constraint glyphs.
Right-click one glyph and select **Hide** to hide that glyph. The same menu
provides **Show All Constraints** and **Hide All Constraints**. Press **F9**
to hide all glyphs. These display changes do not remove constraints or alter
geometry and dimensions.

Display changes use document undo and redo and are saved in native documents.
Show All restores individually hidden glyphs. When glyphs overlap, the
application does not select an ambiguous glyph; see issue #4169.

The typed constraint API can query the visibility state, change one glyph by
its stable relation ID, and show or hide all glyphs. IDs belong to their owning
sketch and are listed by the visibility query.
