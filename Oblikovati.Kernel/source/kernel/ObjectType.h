#pragma once

namespace Oblikovati::Kernel
{
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- START
	enum ObjectTypeEnum
	{
		/// <summary>Edge Collection Object.</summary>
		kEdgeCollectionObject = 28928, // 0x00007100
		/// <summary>Face Collection Object.</summary>
		kFaceCollectionObject = 40704, // 0x00009F00
		/// <summary>Oblikovati Application Object.</summary>
		kApplicationObject = 50331904, // 0x03000100
		/// <summary>Document Object.</summary>
		kDocumentObject = 50332160, // 0x03000200
		/// <summary>Documents collection Object.</summary>
		kDocumentsObject = 50332416, // 0x03000300
		/// <summary>View Object.</summary>
		kViewObject = 50332672, // 0x03000400
		/// <summary>Views collection Object.</summary>
		kViewsObject = 50332928, // 0x03000500
		/// <summary>The Application Events Object.</summary>
		kApplicationEventsObject = 50333184, // 0x03000600
		/// <summary>Document Events Object.</summary>
		kDocumentEventsObject = 50333440, // 0x03000700
		/// <summary>File-UI Events Object.</summary>
		kFileUIEventsObject = 50333952, // 0x03000900
		/// <summary>Object that holds a collection of IRxFileAndReferences interfaces.</summary>
		kFileAndReferencesCollectionObject = 50334976, // 0x03000D00
		/// <summary>File Access Events Object.</summary>
		kFileAccessEventsObject = 50335488, // 0x03000F00
		/// <summary>Object that holds a collection of IRxApplicationAddIn interfaces.</summary>
		kApplicationAddInsObject = 50335744, // 0x03001000
		/// <summary>Object that represents an Application AddIn inside Oblikovati.</summary>
		kApplicationAddInObject = 50336000, // 0x03001100
		/// <summary>Object passed to the AddIn on initialization.</summary>
		kApplicationAddInSiteObject = 50336768, // 0x03001400
		/// <summary>Object that represents a reference held within this Document, to a file.</summary>
		kReferencedFileDescriptorObject = 50337536, // 0x03001700
		/// <summary>Object that holds a collection of IRxFileVersion interfaces.</summary>
		kReferencedFileDescriptorsObject = 50337792, // 0x03001800
		/// <summary>Object that manages the collection of Property Set objects.</summary>
		kPropertySetsObject = 50338304, // 0x03001A00
		/// <summary>Object that represents a Property Set. This is a collection of related Properties.</summary>
		kPropertySetObject = 50338560, // 0x03001B00
		/// <summary>Object that represents Property.</summary>
		kPropertyObject = 50339072, // 0x03001D00
		/// <summary>Custom interface to access management information about a particular file version.</summary>
		kFileSaveAsObject = 50339328, // 0x03001E00
		/// <summary>Object that provides access to the current, active Project being used by Oblikovati. A Project is an organization of 'file locations'.</summary>
		kFileLocationsObject = 50339584, // 0x03001F00
		/// <summary>Graphics Preferences Object.</summary>
		kGraphicsPreferencesObject = 50340096, // 0x03002100
		/// <summary>User Input Events Object.</summary>
		kUserInputEventsObject = 50340352, // 0x03002200
		/// <summary>Document Object available inside the Apprentice Server.</summary>
		kApprenticeServerDocumentObject = 50340736, // 0x03002380
		/// <summary>Apprentice Server Documents collection object.</summary>
		kApprenticeServerDocumentsObject = 50340864, // 0x03002400
		/// <summary>Standalone mini component allowing limited access to an Oblikovati File.</summary>
		kApprenticeServerObject = 50341120, // 0x03002500
		/// <summary>Transaction Manager Object that encapsulates all of the transaction-based functionality.</summary>
		kTransactionManagerObject = 50341376, // 0x03002600
		/// <summary>Transaction Object that stands for a single transaction.</summary>
		kTransactionObject = 50341632, // 0x03002700
		/// <summary>Transactions enumerator object.</summary>
		kTransactionsEnumeratorObject = 50341888, // 0x03002800
		/// <summary>Transaction Events Object.</summary>
		kTransactionEventsObject = 50342144, // 0x03002900
		/// <summary>Object that represents an 'OLE' reference held within this Document, to a file.</summary>
		kReferencedOLEFileDescriptorObject = 50342400, // 0x03002A00
		/// <summary>Object that holds a collection of Referenced OLE File Descriptor objects.</summary>
		kReferencedOLEFileDescriptorsObject = 50342656, // 0x03002B00
		/// <summary>TranslatorAddIn Object.</summary>
		kTranslatorAddInObject = 50343936, // 0x03003000
		/// <summary>Command Object. Encapsulation of the data on a UI command element.</summary>
		kCommandObject = 50344192, // 0x03003100
		/// <summary>Command Manager Object that encapsulates all of the command-based functionality.</summary>
		kCommandManagerObject = 50344704, // 0x03003300
		/// <summary>Transaction Manager Object that encapsulates all of the transaction-based functionality.</summary>
		kHelpManagerObject = 50344960, // 0x03003400
		/// <summary>Graphics Preferences Object.</summary>
		kGeneralPreferencesObject = 50345216, // 0x03003500
		/// <summary>Camera Object that encapsulates all of the viewing parameters into a Scene for a particular View.</summary>
		kCameraObject = 50345472, // 0x03003600
		/// <summary>Object that encapsulates a given Software Version. Used in various contexts.</summary>
		kSoftwareVersionObject = 50345728, // 0x03003700
		/// <summary>Units of Measure Object for a Document.</summary>
		kUnitsOfMeasureObject = 50346240, // 0x03003900
		/// <summary>Parameters Collection Object (abstract base class).</summary>
		kParametersObject = 50347008, // 0x03003C00
		/// <summary>Generated Model Parameters Collection Object.</summary>
		kModelParametersObject = 50347264, // 0x03003D00
		/// <summary>Reference-only Parameters Collection Object.</summary>
		kReferenceParametersObject = 50347520, // 0x03003E00
		/// <summary>User-definable Parameters Collection Object.</summary>
		kUserParametersObject = 50347776, // 0x03003F00
		/// <summary>Table-driven Parameters Collection Object.</summary>
		kTableParametersObject = 50348032, // 0x03004000
		/// <summary>Parameter Tables Collection Object.</summary>
		kParameterTablesObject = 50348288, // 0x03004100
		/// <summary>Model Parameter Object.</summary>
		kModelParameterObject = 50348544, // 0x03004200
		/// <summary>Reference Parameter Object.</summary>
		kReferenceParameterObject = 50348800, // 0x03004300
		/// <summary>User Parameter Object.</summary>
		kUserParameterObject = 50349056, // 0x03004400
		/// <summary>Table Parameter Object.</summary>
		kTableParameterObject = 50349312, // 0x03004500
		/// <summary>Parameter Table Object.</summary>
		kParameterTableObject = 50349568, // 0x03004600
		/// <summary>DocumentsEnumerator Object.</summary>
		kDocumentsEnumeratorObject = 50349824, // 0x03004700
		/// <summary>Views enumerator object.</summary>
		kViewsEnumeratorObject = 50350080, // 0x03004800
		/// <summary>Object through which this document may be printed.</summary>
		kPrintManagerObject = 50351360, // 0x03004D00
		/// <summary>DrawingPrintManager object.</summary>
		kDrawingPrintManagerObject = 50351616, // 0x03004E00
		/// <summary>Attribute Manager for the Document.</summary>
		kAttributeManagerObject = 50351872, // 0x03004F00
		/// <summary>Collection of all AttributeSets associated with an Oblikovati Object.</summary>
		kAttributeSetsObject = 50352128, // 0x03005000
		/// <summary>Fundamental unit for attribution. Object can contain more than one AttributeSet.</summary>
		kAttributeSetObject = 50352384, // 0x03005100
		/// <summary>Attribute Object. Contained within the Attribute Set.</summary>
		kAttributeObject = 50352640, // 0x03005200
		/// <summary>Event object that provides ability to perform selections and receive mouse/keyboard inputs with exclusivity.</summary>
		kInteractionEventsObject = 50353152, // 0x03005400
		/// <summary>Event object that provides ability to perform selections with exclusivity.</summary>
		kSelectEventsObject = 50353408, // 0x03005500
		/// <summary>Event object that provides mouse events with exclusivity.</summary>
		kMouseEventsObject = 50353664, // 0x03005600
		/// <summary>Event object that provides keyboard events with exclusivity.</summary>
		kKeyboardEventsObject = 50354176, // 0x03005800
		/// <summary>Check Point Object that stands for a bookmark within a transaction.</summary>
		kCheckPointObject = 50354432, // 0x03005900
		/// <summary>Check Points Enumerator object.</summary>
		kCheckPointsEnumeratorObject = 50354688, // 0x03005A00
		/// <summary>ClientGraphicsCollection object.</summary>
		kClientGraphicsCollectionObject = 50356224, // 0x03006000
		/// <summary>ClientGraphics object.</summary>
		kClientGraphicsObject = 50356480, // 0x03006100
		/// <summary>GraphicsNode object.</summary>
		kGraphicsNodeObject = 50356736, // 0x03006200
		/// <summary>GraphicsNodeProxy Object.</summary>
		kGraphicsNodeProxyObject = 50356896, // 0x030062A0
		/// <summary>Graphics primitive object.</summary>
		kGraphicsPrimitiveObject = 50356992, // 0x03006300
		/// <summary>LineGraphics Object.</summary>
		kLineGraphicsObject = 50357248, // 0x03006400
		/// <summary>LineStripGraphics Object.</summary>
		kLineStripGraphicsObject = 50357504, // 0x03006500
		/// <summary>TriangleGraphics Object.</summary>
		kTriangleGraphicsObject = 50357760, // 0x03006600
		/// <summary>TriangleStripGraphics Object.</summary>
		kTriangleStripGraphicsObject = 50358016, // 0x03006700
		/// <summary>TriangeFanGraphics Object.</summary>
		kTriangleFanGraphicsObject = 50358272, // 0x03006800
		/// <summary>Point graphics object.</summary>
		kPointGraphicsObject = 50358528, // 0x03006900
		/// <summary>TextGraphics Object.</summary>
		kTextGraphicsObject = 50358784, // 0x03006A00
		/// <summary>RenderStyles object.</summary>
		kRenderStylesObject = 50359040, // 0x03006B00
		/// <summary>RenderStyle object.</summary>
		kRenderStyleObject = 50359296, // 0x03006C00
		/// <summary>GraphicsDataSetsCollection object.</summary>
		kGraphicsDataSetsCollectionObject = 50359552, // 0x03006D00
		/// <summary>GraphicsDataSets object.</summary>
		kGraphicsDataSetsObject = 50359808, // 0x03006E00
		/// <summary>GraphicsDataSet object.</summary>
		kGraphicsDataSetObject = 50360064, // 0x03006F00
		/// <summary>GraphicsCoordinateSet object.</summary>
		kGraphicsCoordinateSetObject = 50360320, // 0x03007000
		/// <summary>GraphicsNormalSet object.</summary>
		kGraphicsNormalSetObject = 50360576, // 0x03007100
		/// <summary>GraphicsColorSet object.</summary>
		kGraphicsColorSetObject = 50360832, // 0x03007200
		/// <summary>GraphicsIndexSet object.</summary>
		kGraphicsIndexSetObject = 50361088, // 0x03007300
		/// <summary>AttributeSets Enumerator Object.</summary>
		kAttributeSetsEnumeratorObject = 50361344, // 0x03007400
		/// <summary>Attributes Enumerator Object.</summary>
		kAttributesEnumeratorObject = 50361600, // 0x03007500
		/// <summary>Materials Collection Object.</summary>
		kMaterialsObject = 50362112, // 0x03007700
		/// <summary>The material object.</summary>
		kMaterialObject = 50362368, // 0x03007800
		/// <summary>Browser Panes object.</summary>
		kBrowserPanesObject = 50362880, // 0x03007A00
		/// <summary>BrowserPane object.</summary>
		kBrowserPaneObject = 50363136, // 0x03007B00
		/// <summary>Environments object.</summary>
		kEnvironmentsObject = 50363648, // 0x03007D00
		/// <summary>Environment object.</summary>
		kEnvironmentObject = 50363904, // 0x03007E00
		/// <summary>CommandBars object.</summary>
		kCommandBarsObject = 50364672, // 0x03008100
		/// <summary>CommandBar object.</summary>
		kCommandBarObject = 50364928, // 0x03008200
		/// <summary>CommandBarControls object.</summary>
		kCommandBarControlsObject = 50365184, // 0x03008300
		/// <summary>CommandBarControl object.</summary>
		kCommandBarControlObject = 50365440, // 0x03008400
		/// <summary>CommandBarPopUp object.</summary>
		kCommandBarPopUpObject = 50365696, // 0x03008500
		/// <summary>CommandBarButton object.</summary>
		kCommandBarButtonObject = 50365952, // 0x03008600
		/// <summary>The mass properties object.</summary>
		kMassPropertiesObject = 50366208, // 0x03008700
		/// <summary>The Selection set object.</summary>
		kSelectSetObject = 50366464, // 0x03008800
		/// <summary>The HighlightSets Object.</summary>
		kHighlightSetsObject = 50366720, // 0x03008900
		/// <summary>The Highlight set object.</summary>
		kHighlightSetObject = 50366976, // 0x03008A00
		/// <summary>Parameters Enumerator Object.</summary>
		kParametersEnumeratorObject = 50367488, // 0x03008C00
		/// <summary>Collection of client views for a document/drawing sheet.</summary>
		kClientViewsObject = 50368000, // 0x03008E00
		/// <summary>A view of the document/drawing sheet attached to a user specified window handle.</summary>
		kClientViewObject = 50368256, // 0x03008F00
		/// <summary>A Collection of Oblikovati's VBA Projects.</summary>
		kOblikovatiVBAProjectsObject = 50368512, // 0x03009000
		/// <summary>Represents Oblikovati's VBA project.</summary>
		kOblikovatiVBAProjectObject = 50368768, // 0x03009100
		/// <summary>A collection of Components in an OblikovatiVBAProject.</summary>
		kOblikovatiVBAComponentsObject = 50369024, // 0x03009200
		/// <summary>The OblikovatiVBAComponent object represents a code, class or form module.</summary>
		kOblikovatiVBAComponentObject = 50369280, // 0x03009300
		/// <summary>Represents a collection of members in a OblikovatiVBAComponent Object.</summary>
		kOblikovatiVBAMembersObject = 50369536, // 0x03009400
		/// <summary>Represents a member within a OblikovatiVBAComponent.</summary>
		kOblikovatiVBAMemberObject = 50369792, // 0x03009500
		/// <summary>Represents a collection of arguments in an OblikovatiVBAMember.</summary>
		kOblikovatiVBAArgumentsObject = 50370048, // 0x03009600
		/// <summary>Represents a VBAArgument object.</summary>
		kOblikovatiVBAArgumentObject = 50370304, // 0x03009700
		/// <summary>Represents a CommandCategory object.</summary>
		kCommandCategoryObject = 50370560, // 0x03009800
		/// <summary>Represents a CommandCategories object.</summary>
		kCommandCategoriesObject = 50370816, // 0x03009900
		/// <summary>Represents a ControlDefinitions object.</summary>
		kControlDefinitionsObject = 50371328, // 0x03009B00
		/// <summary>Represents a ButtonDefinition object.</summary>
		kButtonDefinitionObject = 50371584, // 0x03009C00
		/// <summary>Represents a ComboBoxDefinition object.</summary>
		kComboBoxDefinitionObject = 50372096, // 0x03009E00
		/// <summary>Represents a MacroControlDefinition object.</summary>
		kMacroControlDefinitionObject = 50372352, // 0x03009F00
		/// <summary>Represents a PanelBar object.</summary>
		kPanelBarObject = 50372864, // 0x0300A100
		/// <summary>Represents a ButtonDefinitionHandler Object.</summary>
		kButtonDefinitionHandlerObject = 50374144, // 0x0300A600
		/// <summary>Represents a EnvironmentBaseHandler Object.</summary>
		kEnvironmentBaseHandlerObject = 50374912, // 0x0300A900
		/// <summary>Represents a CommandBarBase collection Object.</summary>
		kCommandBarBaseCollectionObject = 50375424, // 0x0300AB00
		/// <summary>CommandBarBase object.</summary>
		kCommandBarBaseObject = 50375680, // 0x0300AC00
		/// <summary>Represents a EnvironmentsDef Object.</summary>
		kEnvironmentBaseCollectionObject = 50375936, // 0x0300AD00
		/// <summary>Represents a EnvironmentBase Object.</summary>
		kEnvironmentBaseObject = 50376192, // 0x0300AE00
		/// <summary>Represents a CommandBarList Object.</summary>
		kCommandBarListObject = 50376704, // 0x0300B000
		/// <summary>File Dialog Object.</summary>
		kFileDialogObject = 50377728, // 0x0300B400
		/// <summary>Tolerance Object.</summary>
		kToleranceObject = 50377984, // 0x0300B500
		/// <summary>File Manager Object.</summary>
		kFileManagerObject = 50378240, // 0x0300B600
		/// <summary>Document Sub Type Object.</summary>
		kDocumentSubTypeObject = 50378496, // 0x0300B700
		/// <summary>Design Project Object.</summary>
		kDesignProjectObject = 50379008, // 0x0300B900
		/// <summary>Color Object.</summary>
		kColorObject = 50383104, // 0x0300C900
		/// <summary>Provides access to various sketch related document settings.</summary>
		kSketchSettingsObject = 50383360, // 0x0300CA00
		/// <summary>Provides access to various 3D sketch related document settings.</summary>
		kSketch3DSettingsObject = 50383616, // 0x0300CB00
		/// <summary>Provides access to various modeling related document settings.</summary>
		kModelingSettingsObject = 50383872, // 0x0300CC00
		/// <summary>Provides access to nodes underneath a node.</summary>
		kBrowserNodesEnumeratorObject = 50384128, // 0x0300CD00
		/// <summary>Represents a node in the browser.</summary>
		kBrowserNodeObject = 50384384, // 0x0300CE00
		/// <summary>Contains the definition of a node in the browser.</summary>
		kBrowserNodeDefinitionObject = 50384896, // 0x0300D000
		/// <summary>Contains the definition for a client-created node in the browser.</summary>
		kClientBrowserNodeDefinitionObject = 50385152, // 0x0300D100
		/// <summary>Contains the icon used to define a new ClientBrowserNodeDefinition or override an existing BrowserNodeDefinition.</summary>
		kClientNodeResourceObject = 50385664, // 0x0300D300
		/// <summary>Represents a ClientNodeResources collection Object.</summary>
		kClientNodeResourcesObject = 50385920, // 0x0300D400
		/// <summary>Assembly Options Object.</summary>
		kAssemblyOptionsObject = 50386176, // 0x0300D500
		/// <summary>General Options Object.</summary>
		kGeneralOptionsObject = 50386688, // 0x0300D700
		/// <summary>Interaction Graphics Object.</summary>
		kInteractionGraphicsObject = 50387200, // 0x0300D900
		/// <summary>Contains the definition of a native-created node in the browser.</summary>
		kNativeBrowserNodeDefinitionObject = 50387456, // 0x0300DA00
		/// <summary>UserInterfaceManager Object.</summary>
		kUserInterfaceManagerObject = 50388224, // 0x0300DD00
		/// <summary>ChangeManager Object.</summary>
		kChangeManagerObject = 50388736, // 0x0300DF00
		/// <summary>ChangeDefinition Object.</summary>
		kChangeDefinitionsObject = 50388992, // 0x0300E000
		/// <summary>ChangeDefinition Object.</summary>
		kChangeDefinitionObject = 50389248, // 0x0300E100
		/// <summary>ChangeProcessor Object.</summary>
		kChangeProcessorObject = 50389504, // 0x0300E200
		/// <summary>Represents a ViewList object.</summary>
		kViewListObject = 50389760, // 0x0300E300
		/// <summary>Control definition events object.</summary>
		kControlDefinitionEventsObject = 50390016, // 0x0300E400
		/// <summary>Represents a PanelBarList object.</summary>
		kEnvironmentListObject = 50390528, // 0x0300E600
		/// <summary>Apprentice Print Manager Object.</summary>
		kApprenticePrintManagerObject = 50390784, // 0x0300E700
		/// <summary>Apprentice Drawing Print Manager Object.</summary>
		kApprenticeDrawingPrintManagerObject = 50391040, // 0x0300E800
		/// <summary>DisabledCommandList object.</summary>
		kDisabledCommandListObject = 50391296, // 0x0300E900
		/// <summary>EnvironmentManager object.</summary>
		kEnvironmentManagerObject = 50391552, // 0x0300EA00
		/// <summary>Event object that provides Triad events with exclusivity.</summary>
		kTriadEventsObject = 50392064, // 0x0300EC00
		/// <summary>Content Center Object.</summary>
		kContentCenterObject = 50392320, // 0x0300ED00
		/// <summary>ContentQuery Object.</summary>
		kContentQueryObject = 50393088, // 0x0300F000
		/// <summary>File Manager Events Object.</summary>
		kFileManagerEventsObject = 50398208, // 0x03010400
		/// <summary>CustomParameterGroups Object.</summary>
		kCustomParameterGroupsObject = 50398464, // 0x03010500
		/// <summary>ColorSchemes object.</summary>
		kColorSchemesObject = 50398976, // 0x03010700
		/// <summary>ColorScheme object.</summary>
		kColorSchemeObject = 50399232, // 0x03010800
		/// <summary>FileOptions object.</summary>
		kFileOptionsObject = 50399488, // 0x03010900
		/// <summary>SketchOptions object.</summary>
		kSketchOptionsObject = 50400000, // 0x03010B00
		/// <summary>Content Center configuration Object.</summary>
		kContentCenterConfigurationObject = 50400256, // 0x03010C00
		/// <summary>Design Project Manager Object.</summary>
		kDesignProjectManagerObject = 50401024, // 0x03010F00
		/// <summary>Design Projects Object.</summary>
		kDesignProjectsObject = 50401280, // 0x03011000
		/// <summary>CustomParameterGroup Object.</summary>
		kCustomParameterGroupObject = 50401536, // 0x03011100
		/// <summary>LightingStyles Object.</summary>
		kLightingStylesObject = 50401792, // 0x03011200
		/// <summary>LightingStyle Object.</summary>
		kLightingStyleObject = 50402048, // 0x03011300
		/// <summary>Lights Object.</summary>
		kLightsObject = 50402304, // 0x03011400
		/// <summary>Light Object.</summary>
		kLightObject = 50402560, // 0x03011500
		/// <summary>UserInterfaceEvents Object.</summary>
		kUserInterfaceEventsObject = 50402816, // 0x03011600
		/// <summary>TextureMaps Object.</summary>
		kTextureMapsObject = 50404608, // 0x03011D00
		/// <summary>TextureMaps Object.</summary>
		kTextureMapObject = 50404864, // 0x03011E00
		/// <summary>TextureCoordinateSet Object.</summary>
		kTextureCoordinateSetObject = 50405376, // 0x03012000
		/// <summary>PartOptions Object.</summary>
		kPartOptionsObject = 50405632, // 0x03012100
		/// <summary>Sketch3DOptions Object.</summary>
		kSketch3DOptionsObject = 50405888, // 0x03012200
		/// <summary>iFeature Options Object.</summary>
		kiFeatureOptionsObject = 50406144, // 0x03012300
		/// <summary>Notebook Options Object.</summary>
		kNotebookOptionsObject = 50406400, // 0x03012400
		/// <summary>HardwareOptions Object.</summary>
		kHardwareOptionsObject = 50406656, // 0x03012500
		/// <summary>FilesEnumerator Object.</summary>
		kFilesEnumeratorObject = 50406912, // 0x03012600
		/// <summary>File Object.</summary>
		kFileObject = 50407168, // 0x03012700
		/// <summary>FileDescriptorsEnumerator.</summary>
		kFileDescriptorsEnumeratorObject = 50407424, // 0x03012800
		/// <summary>FileDescriptor.</summary>
		kFileDescriptorObject = 50407680, // 0x03012900
		/// <summary>DocumentDescriptor.</summary>
		kDocumentDescriptorObject = 50407936, // 0x03012A00
		/// <summary>DocumentDescriptorsEnumerator.</summary>
		kDocumentDescriptorsEnumeratorObject = 50408192, // 0x03012B00
		/// <summary>SheetSettings Object.</summary>
		kSheetSettingsObject = 50408448, // 0x03012C00
		/// <summary>The Modeling Events Object.</summary>
		kModelingEventsObject = 50408704, // 0x03012D00
		/// <summary>Save Options Object.</summary>
		kSaveOptionsObject = 50408960, // 0x03012E00
		/// <summary>ContentCenterDialog Object.</summary>
		kContentCenterDialogObject = 50409216, // 0x03012F00
		/// <summary>ContentCenterDialogEvents Object.</summary>
		kContentCenterDialogEventsObject = 50409472, // 0x03013000
		/// <summary>LibraryManager Object.</summary>
		kLibraryManagerObject = 50409728, // 0x03013100
		/// <summary>CategoryManager Object.</summary>
		kCategoryManagerObject = 50409984, // 0x03013200
		/// <summary>FamilyManager Object.</summary>
		kFamilyManagerObject = 50410240, // 0x03013300
		/// <summary>MemberManager Object.</summary>
		kMemberManagerObject = 50410496, // 0x03013400
		/// <summary>QueryManager Object.</summary>
		kQueryManagerObject = 50410752, // 0x03013500
		/// <summary>The Sketch Events Object.</summary>
		kSketchEventsObject = 50411008, // 0x03013600
		/// <summary>The Representation Events Object.</summary>
		kRepresentationEventsObject = 50411264, // 0x03013700
		/// <summary>ThreadQuery object.</summary>
		kThreadTableQueryObject = 50411520, // 0x03013800
		/// <summary>DerivedParameterTables object.</summary>
		kDerivedParameterTablesObject = 50411776, // 0x03013900
		/// <summary>DerivedParameterTable Object.</summary>
		kDerivedParameterTableObject = 50412032, // 0x03013A00
		/// <summary>DerivedParameters Object.</summary>
		kDerivedParametersObject = 50412288, // 0x03013B00
		/// <summary>DerivedParameter Object.</summary>
		kDerivedParameterObject = 50412544, // 0x03013C00
		/// <summary>WireframeDisplayOptions Object.</summary>
		kWireframeDisplayOptionsObject = 50412800, // 0x03013D00
		/// <summary>ShadedDisplayOptions Object.</summary>
		kShadedDisplayOptionsObject = 50413056, // 0x03013E00
		/// <summary>DisplayOptions Object.</summary>
		kDisplayOptionsObject = 50413312, // 0x03013F00
		/// <summary>MeasureTools Object.</summary>
		kMeasureToolsObject = 50413568, // 0x03014000
		/// <summary>SweepGraphics object.</summary>
		kSweepGraphicsObject = 50414080, // 0x03014200
		/// <summary>The Style Events Object.</summary>
		kStyleEventsObject = 50414336, // 0x03014300
		/// <summary>File Dialog Events Object.</summary>
		kFileDialogEventsObject = 50414592, // 0x03014400
		/// <summary>CurveGraphics Object.</summary>
		kCurveGraphicsObject = 50414848, // 0x03014500
		/// <summary>Drawing Options Object.</summary>
		kDrawingOptionsObject = 50415104, // 0x03014600
		/// <summary>Document Interests Object.</summary>
		kDocumentInterestsObject = 50415360, // 0x03014700
		/// <summary>Document Interests Object.</summary>
		kDocumentInterestObject = 50415616, // 0x03014800
		/// <summary>Language Tools Object.</summary>
		kLanguageToolsObject = 50415872, // 0x03014900
		/// <summary>IntentConfiguration Object.</summary>
		kIntentConfigurationObject = 50416128, // 0x03014A00
		/// <summary>Event object that provides measure events with exclusivity.</summary>
		kMeasureEventsObject = 50416384, // 0x03014B00
		/// <summary>Draft analysis collection object.</summary>
		kDraftAnalysesObject = 50416640, // 0x03014C00
		/// <summary>Draft analysis object.</summary>
		kDraftAnalysisObject = 50416896, // 0x03014D00
		/// <summary>Analysis manager object.</summary>
		kAnalysisManagerObject = 50417152, // 0x03014E00
		/// <summary>Surface graphics object.</summary>
		kSurfaceGraphicsObject = 50417920, // 0x03015100
		/// <summary>File Metadata Object.</summary>
		kFileMetadataObject = 50418176, // 0x03015200
		/// <summary>surface graphics face list object.</summary>
		kSurfaceGraphicsFaceListObject = 50418432, // 0x03015300
		/// <summary>surface graphics face object.</summary>
		kSurfaceGraphicsFaceObject = 50418688, // 0x03015400
		/// <summary>surEdge graphics Edge list object.</summary>
		kSurfaceGraphicsEdgeListObject = 50418944, // 0x03015500
		/// <summary>surface graphics edge object.</summary>
		kSurfaceGraphicsEdgeObject = 50419200, // 0x03015600
		/// <summary>ProgressBar Object.</summary>
		kProgressBarObject = 50419968, // 0x03015900
		/// <summary>ContentCenterEvents Object.</summary>
		kContentCenterEventsObject = 50420224, // 0x03015A00
		/// <summary>ErrorManager Object.</summary>
		kErrorManagerObject = 50420736, // 0x03015C00
		/// <summary>MessageSection Object.</summary>
		kMessageSectionObject = 50420992, // 0x03015D00
		/// <summary>ContentCenterOptions Object.</summary>
		kContentCenterOptionsObject = 50421248, // 0x03015E00
		/// <summary>ContentTableCell Object.</summary>
		kContentTableCellObject = 50421504, // 0x03015F00
		/// <summary>ContentFamily Object.</summary>
		kContentFamilyObject = 50421760, // 0x03016000
		/// <summary>ContentTableRow Object.</summary>
		kContentTableRowObject = 50422016, // 0x03016100
		/// <summary>ContentTableRows Object.</summary>
		kContentTableRowsObject = 50422272, // 0x03016200
		/// <summary>ContentTableColumn Object.</summary>
		kContentTableColumnObject = 50422528, // 0x03016300
		/// <summary>ExpressionLimits Object.</summary>
		kExpressionLimitsObject = 50422784, // 0x03016400
		/// <summary>ContentTableColumns Object.</summary>
		kContentTableColumnsObject = 50423040, // 0x03016500
		/// <summary>ContentFamiliesEnumerator Object.</summary>
		kContentFamiliesEnumeratorObject = 50423296, // 0x03016600
		/// <summary>ContentTreeViewNodesEnumerator Object.</summary>
		kContentTreeViewNodesEnumeratorObject = 50423552, // 0x03016700
		/// <summary>ContentTreeViewNode Object.</summary>
		kContentTreeViewNodeObject = 50423808, // 0x03016800
		/// <summary>Ribbons Object.</summary>
		kRibbonsObject = 50424064, // 0x03016900
		/// <summary>Ribbon Object.</summary>
		kRibbonObject = 50424320, // 0x03016A00
		/// <summary>RibbonTabs Object.</summary>
		kRibbonTabsObject = 50424576, // 0x03016B00
		/// <summary>RibbonTab Object.</summary>
		kRibbonTabObject = 50424832, // 0x03016C00
		/// <summary>RibbonPanels Object.</summary>
		kRibbonPanelsObject = 50425088, // 0x03016D00
		/// <summary>RibbonPanel Object.</summary>
		kRibbonPanelObject = 50425344, // 0x03016E00
		/// <summary>CommandControls Object.</summary>
		kCommandControlsObject = 50425600, // 0x03016F00
		/// <summary>CommandControl Object.</summary>
		kCommandControlObject = 50425856, // 0x03017000
		/// <summary>StylesManager Object.</summary>
		kStylesManagerObject = 50426112, // 0x03017100
		/// <summary>Materials Enumerator Object.</summary>
		kMaterialsEnumeratorObject = 50426368, // 0x03017200
		/// <summary>CommandControlsEnumerator Object.</summary>
		kCommandControlsEnumeratorObject = 50426624, // 0x03017300
		/// <summary>Provides access to all browser folders directly under a browser node.</summary>
		kBrowserFoldersEnumeratorObject = 50426880, // 0x03017400
		/// <summary>Represents a folder in the browser.</summary>
		kBrowserFolderObject = 50427392, // 0x03017600
		/// <summary>FaceShellDefinitions Object.</summary>
		kFaceShellDefinitionsObject = 50427648, // 0x03017700
		/// <summary>FaceShellDefinition Object.</summary>
		kFaceShellDefinitionObject = 50427904, // 0x03017800
		/// <summary>FaceDefinitions Object.</summary>
		kFaceDefinitionsObject = 50428160, // 0x03017900
		/// <summary>FaceDefinition Object.</summary>
		kFaceDefinitionObject = 50428416, // 0x03017A00
		/// <summary>EdgeLoopDefinitions Object.</summary>
		kEdgeLoopDefinitionsObject = 50428672, // 0x03017B00
		/// <summary>EdgeLoopDefinition Object.</summary>
		kEdgeLoopDefinitionObject = 50428928, // 0x03017C00
		/// <summary>EdgeUseDefinitions Object.</summary>
		kEdgeUseDefinitionsObject = 50429184, // 0x03017D00
		/// <summary>EdgeUseDefinition Object.</summary>
		kEdgeUseDefinitionObject = 50429440, // 0x03017E00
		/// <summary>EdgeDefinitions Object.</summary>
		kEdgeDefinitionsObject = 50429696, // 0x03017F00
		/// <summary>EdgeDefinition Object.</summary>
		kEdgeDefinitionObject = 50429952, // 0x03018000
		/// <summary>VertexDefinitions Object.</summary>
		kVertexDefinitionsObject = 50430208, // 0x03018100
		/// <summary>VertexDefinition Object.</summary>
		kVertexDefinitionObject = 50430464, // 0x03018200
		/// <summary>CustomPropertyFormat Object.</summary>
		kCustomPropertyFormat = 50430720, // 0x03018300
		/// <summary>GripSnapOptions object.</summary>
		kGripSnapOptionsObject = 50430976, // 0x03018400
		/// <summary>UserCoordinateSystemSettings object.</summary>
		kUserCoordinateSystemSettingsObject = 50431232, // 0x03018500
		/// <summary>ObjectVisibility Object.</summary>
		kObjectVisibilityObject = 50431488, // 0x03018600
		/// <summary>GraphicsImageSet Object.</summary>
		kGraphicsImageSetObject = 50431744, // 0x03018700
		/// <summary>ProjectPaths Object.</summary>
		kProjectPathsObject = 50432000, // 0x03018800
		/// <summary>ProjectPath Object.</summary>
		kProjectPathObject = 50432256, // 0x03018900
		/// <summary>LumpDefinitions Object.</summary>
		kLumpDefinitionsObject = 50432512, // 0x03018A00
		/// <summary>LumpDefinition Object.</summary>
		kLumpDefinitionObject = 50432768, // 0x03018B00
		/// <summary>DockableWindows Object.</summary>
		kDockableWindowsObject = 50433024, // 0x03018C00
		/// <summary>DockableWindow Object.</summary>
		kDockableWindowObject = 50433280, // 0x03018D00
		/// <summary>ProgressiveToolTip Object.</summary>
		kProgressiveToolTipObject = 50433536, // 0x03018E00
		/// <summary>ExpressionList Object.</summary>
		kExpressionListObject = 50433792, // 0x03018F00
		/// <summary>PresentationOptions Object.</summary>
		kPresentationOptionsObject = 50434304, // 0x03019100
		/// <summary>SurfaceBodyDefinition Object.</summary>
		kSurfaceBodyDefinitionObject = 50435072, // 0x03019400
		/// <summary>Camera Events Object.</summary>
		kCameraEventsObject = 50435328, // 0x03019500
		/// <summary>HeadsUpDisplayOptions Object.</summary>
		kHeadsUpDisplayOptionsObject = 50435584, // 0x03019600
		/// <summary>DisplaySettings Object.</summary>
		kDisplaySettingsObject = 50435840, // 0x03019700
		/// <summary>DisplaySettings Object.</summary>
		kGroundPlaneSettingsObject = 50436096, // 0x03019800
		/// <summary>FileOpenOptions object.</summary>
		kFileOpenOptionsObject = 50436352, // 0x03019900
		/// <summary>ComponentGraphics Object.</summary>
		kComponentGraphicsObject = 50436608, // 0x03019A00
		/// <summary>BIMComponent object.</summary>
		kBIMComponentObject = 50436864, // 0x03019B00
		/// <summary>BIMExchangeServer object.</summary>
		kBIMExchangeServerObject = 50437120, // 0x03019C00
		/// <summary>BIMComponentDescription object.</summary>
		kBIMComponentDescriptionObject = 50437376, // 0x03019D00
		/// <summary>BIMConnectors object.</summary>
		kBIMConnectorsObject = 50437632, // 0x03019E00
		/// <summary>BIMConnectorLinks object.</summary>
		kBIMConnectorLinksObject = 50437888, // 0x03019F00
		/// <summary>BIMConnectorLink object.</summary>
		kBIMConnectorLinkObject = 50438032, // 0x03019F90
		/// <summary>BIMComponentPropertySets object.</summary>
		kBIMComponentPropertySetsObject = 50438144, // 0x0301A000
		/// <summary>BIMComponentPropertySet object.</summary>
		kBIMComponentPropertySetObject = 50438400, // 0x0301A100
		/// <summary>BIMComponentProperty object.</summary>
		kBIMComponentPropertyObject = 50438656, // 0x0301A200
		/// <summary>BIMConnector object.</summary>
		kBIMConnectorObject = 50438912, // 0x0301A300
		/// <summary>BIMConnectorDefinition object.</summary>
		kBIMConnectorDefinitionObject = 50439168, // 0x0301A400
		/// <summary>BIMPipeConnectorDefinition object.</summary>
		kBIMPipeConnectorDefinitionObject = 50439424, // 0x0301A500
		/// <summary>BIMElectricalConnectorDefinition object.</summary>
		kBIMElectricalConnectorDefinitionObject = 50439680, // 0x0301A600
		/// <summary>BIMDuctConnectorDefinition object.</summary>
		kBIMDuctConnectorDefinitionObject = 50439936, // 0x0301A700
		/// <summary>BIMConduitConnectorDefinition object.</summary>
		kBIMConduitConnectorDefinitionObject = 50440192, // 0x0301A800
		/// <summary>BIMCableTrayConnectorDefinition object.</summary>
		kBIMCableTrayConnectorDefinitionObject = 50440448, // 0x0301A900
		/// <summary>The DragContext object provides methods that allow a user to move an object similar to the standard Oblikovati Assembly Drag.</summary>
		kDragContextObject = 50440704, // 0x0301AA00
		/// <summary>The MiniToolbar object provides the ability to create and display in-canvas controls for user interaction.</summary>
		kMiniToolbarObject = 50441216, // 0x0301AC00
		/// <summary>BalloonTips Object.</summary>
		kBalloonTipsObject = 50441472, // 0x0301AD00
		/// <summary>BalloonTip Object.</summary>
		kBalloonTipObject = 50441728, // 0x0301AE00
		/// <summary>Represents a RadialMarkingMenu Object.</summary>
		kRadialMarkingMenuObject = 50441984, // 0x0301AF00
		/// <summary>Control definition events object.</summary>
		kDockableWindowsEventsObject = 50442240, // 0x0301B000
		/// <summary>Object that provides access to help related events.</summary>
		kHelpEventsObject = 50442496, // 0x0301B100
		/// <summary>Object that provides access to events associated with custom reference keys.</summary>
		kReferenceKeyEventsObject = 50443008, // 0x0301B300
		/// <summary>The MiniToolbarControl object is the base class for all controls within a MiniToolbar.</summary>
		kMiniToolbarControlObject = 50443264, // 0x0301B400
		/// <summary>The MiniToolbarButton object represents a button control within a MiniToolbar.</summary>
		kMiniToolbarButtonObject = 50443520, // 0x0301B500
		/// <summary>The MiniToolbarCheckBox object represents a check box control within a MiniToolbar.</summary>
		kMiniToolbarCheckBoxObject = 50443776, // 0x0301B600
		/// <summary>The MiniToolbarComboBox object represents a combobox control within a MiniToolbar.</summary>
		kMiniToolbarComboBoxObject = 50444032, // 0x0301B700
		/// <summary>The MiniToolbarDropdown object represents a dropdown control within a MiniToolbar.</summary>
		kMiniToolbarDropdownObject = 50444288, // 0x0301B800
		/// <summary>The MiniToolbarValueEditor object represents a value edit control within a MiniToolbar.</summary>
		kMiniToolbarValueEditorObject = 50444544, // 0x0301B900
		/// <summary>The MiniToolbarControls object contains the controls within a minitoolbar and methods to create new controls.</summary>
		kMiniToolbarControlsObject = 50444800, // 0x0301BA00
		/// <summary>ProjectOptionsButton object.</summary>
		kProjectOptionsButtonObject = 50445056, // 0x0301BB00
		/// <summary>The MiniToolbarListItem object represents an item in either a MiniToolBarComboBox or a MiniToolBarDropdown object.</summary>
		kMiniToolbarListItemObject = 50445312, // 0x0301BC00
		/// <summary>WireDefinitions object.</summary>
		kWireDefinitionsObject = 50445568, // 0x0301BD00
		/// <summary>WireDefinition object.</summary>
		kWireDefinitionObject = 50445824, // 0x0301BE00
		/// <summary>WireEdgeDefinitions object.</summary>
		kWireEdgeDefinitionsObject = 50446080, // 0x0301BF00
		/// <summary>WireEdgeDefinition object.</summary>
		kWireEdgeDefinitionObject = 50446336, // 0x0301C000
		/// <summary>RadialMarkingMenus object.</summary>
		kRadialMarkingMenusObject = 50446592, // 0x0301C100
		/// <summary>The MiniToolbarSlider object represents a slider control within a MiniToolbar.</summary>
		kMiniToolbarSliderObject = 50446848, // 0x0301C200
		/// <summary>surface graphics vertex list object.</summary>
		kSurfaceGraphicsVertexListObject = 50447104, // 0x0301C300
		/// <summary>surface graphics vertex object.</summary>
		kSurfaceGraphicsVertexObject = 50447360, // 0x0301C400
		/// <summary>Asset Libraries Object.</summary>
		kAssetLibrariesObject = 50447616, // 0x0301C500
		/// <summary>Asset library Object.</summary>
		kAssetLibraryObject = 50447872, // 0x0301C600
		/// <summary>Assets Object.</summary>
		kAssetsObject = 50448128, // 0x0301C700
		/// <summary>Assets Enumerator Object.</summary>
		kAssetsEnumeratorObject = 50448384, // 0x0301C800
		/// <summary>Asset Object.</summary>
		kAssetObject = 50448640, // 0x0301C900
		/// <summary>Material Asset Object.</summary>
		kMaterialAssetObject = 50448896, // 0x0301CA00
		/// <summary>Asset Categories Object.</summary>
		kAssetCategoriesObject = 50449152, // 0x0301CB00
		/// <summary>Asset Category Object.</summary>
		kAssetCategoryObject = 50449408, // 0x0301CC00
		/// <summary>Asset Value Object.</summary>
		kAssetValueObject = 50449664, // 0x0301CD00
		/// <summary>Choice Asset Value Object.</summary>
		kChoiceAssetValueObject = 50449920, // 0x0301CE00
		/// <summary>Float Asset Value Object.</summary>
		kFloatAssetValueObject = 50450176, // 0x0301CF00
		/// <summary>Integer Asset Value Object.</summary>
		kIntegerAssetValueObject = 50450432, // 0x0301D000
		/// <summary>Boolean Asset Value Object.</summary>
		kBooleanAssetValueObject = 50450944, // 0x0301D200
		/// <summary>Color Asset Value Object.</summary>
		kColorAssetValueObject = 50451200, // 0x0301D300
		/// <summary>Texture Asset Value Object.</summary>
		kTextureAssetValueObject = 50451456, // 0x0301D400
		/// <summary>String Asset Value Object.</summary>
		kStringAssetValueObject = 50451712, // 0x0301D500
		/// <summary>Filename Asset Value Object.</summary>
		kFilenameAssetValueObject = 50451968, // 0x0301D600
		/// <summary>Reference Asset Value Object.</summary>
		kReferenceAssetValueObject = 50452224, // 0x0301D700
		/// <summary>Asset Texture Object.</summary>
		kAssetTextureObject = 50452480, // 0x0301D800
		/// <summary>ProjectAssetLibraries Object.</summary>
		kProjectAssetLibrariesObject = 50452736, // 0x0301D900
		/// <summary>ProjectAssetLibrary Object.</summary>
		kProjectAssetLibraryObject = 50452992, // 0x0301DA00
		/// <summary>The MiniToolbarTextEditor object represents a text edit control within a MiniToolbar.</summary>
		kMiniToolbarTextEditorObject = 50453504, // 0x0301DC00
		/// <summary>Sketch Constraint Settings Object.</summary>
		kSketchConstraintSettingsObject = 50453760, // 0x0301DD00
		/// <summary>SelectionPreferences Object.</summary>
		kSelectionPreferencesObject = 50454016, // 0x0301DE00
		/// <summary>WebBrowserDockableWindow Object.</summary>
		kWebBrowserDockableWindowObject = 50454272, // 0x0301DF00
		/// <summary>TutorialsManager Object.</summary>
		kTutorialsManagerObject = 50454528, // 0x0301E000
		/// <summary>WebViews Object.</summary>
		kWebViewsObject = 50454784, // 0x0301E100
		/// <summary>WebView Object.</summary>
		kWebViewObject = 50455040, // 0x0301E200
		/// <summary>WebBrowserDialog Object.</summary>
		kWebBrowserDialogObject = 50455296, // 0x0301E300
		/// <summary>WebBrowserDialogs Object.</summary>
		kWebBrowserDialogsObject = 50455552, // 0x0301E400
		/// <summary>WebBrowserDialogEvents Object.</summary>
		kWebBrowserDialogEventsObject = 50455808, // 0x0301E500
		/// <summary>Provides access to flat pattern related document settings.</summary>
		kFlatPatternSettingsObject = 50456064, // 0x0301E600
		/// <summary>Object that represents an opaque reference held within this Document, to a file.</summary>
		kReferencedOpaqueFileDescriptorObject = 50456320, // 0x0301E700
		/// <summary>Object that holds a collection of Referenced Opaque File Descriptor objects.</summary>
		kReferencedOpaqueFileDescriptorsObject = 50456576, // 0x0301E800
		/// <summary>ManipulatorEvents Object.</summary>
		kManipulatorEventsObject = 50456832, // 0x0301E900
		/// <summary>PublicationStage Object.</summary>
		kPublicationStageObject = 50461184, // 0x0301FA00
		/// <summary>Factory Table Dialog Object.</summary>
		kFactoryTableDialog = 50461440, // 0x0301FB00
		/// <summary>The MiniToolbarTextBox object represents a text box control within a MiniToolbar.</summary>
		kMiniToolbarTextBoxObject = 50461696, // 0x0301FC00
		/// <summary>The SearchBox object represents a search box control within a BrowserPane object.</summary>
		kSearchBoxObject = 50461952, // 0x0301FD00
		/// <summary>SearchBox Events Object.</summary>
		kSearchBoxEventsObject = 50462208, // 0x0301FE00
		/// <summary>Publication Events Object.</summary>
		kPublicationEventsObject = 50462464, // 0x0301FF00
		/// <summary>CustomData Events Object.</summary>
		kCustomDataEventsObject = 50462720, // 0x03020000
		/// <summary>Plugin License Manager.</summary>
		kPluginLicenseManagerObject = 50462976, // 0x03020100
		/// <summary>ThemeManager Object.</summary>
		kThemeManagerObject = 50463232, // 0x03020200
		/// <summary>CustomDictionary Object.</summary>
		kCustomDictionaryObject = 50463488, // 0x03020300
		/// <summary>CustomDictionaries Object.</summary>
		kCustomDictionariesObject = 50463744, // 0x03020400
		/// <summary>SpellCheckOptions Object.</summary>
		kSpellCheckOptionsObject = 50464000, // 0x03020500
		/// <summary>PluginLicenseManager Events Object.</summary>
		kPluginLicenseManagerEventsObject = 50464256, // 0x03020600
		/// <summary>Theme Object.</summary>
		kThemeObject = 50464512, // 0x03020700
		/// <summary>ThemesEnumerator Object.</summary>
		kThemesEnumeratorObject = 50464768, // 0x03020800
		/// <summary>ViewFrame Object.</summary>
		kViewFrameObject = 50465024, // 0x03020900
		/// <summary>ViewFramesEnumerator Object.</summary>
		kViewFramesEnumeratorObject = 50466816, // 0x03021000
		/// <summary>The ModelStateEvents Object.</summary>
		kModelStateEventsObject = 50467072, // 0x03021100
		/// <summary>Represents a ClientResourceMaps collection Object.</summary>
		kClientResourceMapsObject = 50467328, // 0x03021200
		/// <summary>Object that stores client resource ids with theme related icons.</summary>
		kClientResourceMapObject = 50467584, // 0x03021300
		/// <summary>LicenseManager Object.</summary>
		kLicenseManagerObject = 50467840, // 0x03021400
		/// <summary>ViewTab object.</summary>
		kViewTabObject = 50468096, // 0x03021500
		/// <summary>ViewTabGroup object.</summary>
		kViewTabGroupObject = 50468352, // 0x03021600
		/// <summary>ViewTabsEnumerator object.</summary>
		kViewTabsEnumeratorObject = 50468608, // 0x03021700
		/// <summary>ViewTabGroupsEnumerator object.</summary>
		kViewTabGroupsEnumeratorObject = 50468864, // 0x03021800
		/// <summary>iLogicOptions Object.</summary>
		kiLogicOptionsObject = 50469120, // 0x03021900
		/// <summary>SurfaceBodyMassProperties Object.</summary>
		kSurfaceBodyMassPropertiesObject = 50469376, // 0x03021A00
		/// <summary>FinishParameters Object.</summary>
		kFinishParametersObject = 50469632, // 0x03021B00
		/// <summary>FinishParameter Object.</summary>
		kFinishParameterObject = 50469888, // 0x03021C00
		/// <summary>PromptMessage object.</summary>
		kPromptMessageObject = 50470144, // 0x03021D00
		/// <summary>PromptMessages object.</summary>
		kPromptMessagesObject = 50470400, // 0x03021E00
		/// <summary>PromptsOptions object.</summary>
		kPromptsOptionsObject = 50470656, // 0x03021F00
		/// <summary>ClientOperationEvents object.</summary>
		kClientOperationEventsObject = 50470912, // 0x03022000
		/// <summary>ComponentDefinition object.</summary>
		kComponentDefinitionObject = 67113264, // 0x04001130
		/// <summary>ComponentDefinitionReference object.</summary>
		kComponentDefinitionReferenceObject = 67113520, // 0x04001230
		/// <summary>ComponentOccurrence object.</summary>
		kComponentOccurrenceObject = 67113776, // 0x04001330
		/// <summary>ComponentOccurrenceProxy object.</summary>
		kComponentOccurrenceProxyObject = 67113888, // 0x040013A0
		/// <summary>ComponentDefinitions Object.</summary>
		kComponentDefinitionsObject = 67114288, // 0x04001530
		/// <summary>ComponentDefinitionsEnumerator Object.</summary>
		kComponentDefinitionsEnumeratorObject = 67114336, // 0x04001560
		/// <summary>ComponentOccurrences Object.</summary>
		kComponentOccurrencesObject = 67114544, // 0x04001630
		/// <summary>ComponentDefinitionReferences object.</summary>
		kComponentDefinitionReferencesObject = 67114800, // 0x04001730
		/// <summary>Faces object.</summary>
		kFacesObject = 67118592, // 0x04002600
		/// <summary>SurfaceBody object.</summary>
		kSurfaceBodyObject = 67118896, // 0x04002730
		/// <summary>SurfaceBodyProxy object.</summary>
		kSurfaceBodyProxyObject = 67119008, // 0x040027A0
		/// <summary>FaceShell object.</summary>
		kFaceShellObject = 67119152, // 0x04002830
		/// <summary>FaceShellProxy object.</summary>
		kFaceShellProxyObject = 67119264, // 0x040028A0
		/// <summary>Face Object.</summary>
		kFaceObject = 67119408, // 0x04002930
		/// <summary>FaceProxy object.</summary>
		kFaceProxyObject = 67119520, // 0x040029A0
		/// <summary>EdgeLoop object.</summary>
		kEdgeLoopObject = 67119664, // 0x04002A30
		/// <summary>EdgeLoopProxy object.</summary>
		kEdgeLoopProxyObject = 67119712, // 0x04002A60
		/// <summary>EdgeUse object.</summary>
		kEdgeUseObject = 67119920, // 0x04002B30
		/// <summary>EdgeUseProxy object.</summary>
		kEdgeUseProxyObject = 67120032, // 0x04002BA0
		/// <summary>Edge object.</summary>
		kEdgeObject = 67120176, // 0x04002C30
		/// <summary>EdgeProxy object.</summary>
		kEdgeProxyObject = 67120288, // 0x04002CA0
		/// <summary>Vertex Object.</summary>
		kVertexObject = 67120432, // 0x04002D30
		/// <summary>VertexProxy Object.</summary>
		kVertexProxyObject = 67120544, // 0x04002DA0
		/// <summary>SurfaceBodies object.</summary>
		kSurfaceBodiesObject = 67121456, // 0x04003130
		/// <summary>FaceShells object.</summary>
		kFaceShellsObject = 67121712, // 0x04003230
		/// <summary>EdgeLoops.</summary>
		kEdgeLoopsObject = 67122224, // 0x04003430
		/// <summary>EdgeUses object.</summary>
		kEdgeUsesObject = 67122480, // 0x04003530
		/// <summary>Vertices object.</summary>
		kVerticesObject = 67122992, // 0x04003730
		/// <summary>ComponentOccurrence enumerator object.</summary>
		kComponentOccurrencesEnumeratorObject = 67126528, // 0x04004500
		/// <summary>TransientGeometry object.</summary>
		kTransientGeometryObject = 67126912, // 0x04004680
		/// <summary>ReferenceKeyManager Object.</summary>
		kReferenceKeyManagerObject = 67128576, // 0x04004D00
		/// <summary>OccurrencePatternElement Object.</summary>
		kOccurrencePatternElementObject = 67128832, // 0x04004E00
		/// <summary>Occurrence Pattern Element Proxy Object.</summary>
		kOccurrencePatternElementProxyObject = 67128992, // 0x04004EA0
		/// <summary>Point Inference object.</summary>
		kPointInferenceObject = 67129088, // 0x04004F00
		/// <summary>PointInference Enumerator Object.</summary>
		kPointInferenceObjectEnumerator = 67129344, // 0x04005000
		/// <summary>BSplineCurveDefinition Object.</summary>
		kBSplineCurveDefinitionObject = 67130624, // 0x04005500
		/// <summary>BSplineCurve2dDefinition Object.</summary>
		kBSplineCurve2dDefinitionObject = 67130880, // 0x04005600
		/// <summary>BOMQuantity Object.</summary>
		kBOMQuantityObject = 67131136, // 0x04005700
		/// <summary>Wire Object.</summary>
		kWireObject = 67131392, // 0x04005800
		/// <summary>Wire Object.</summary>
		kWiresObject = 67132160, // 0x04005B00
		/// <summary>PartComponentDefinition Object.</summary>
		kPartComponentDefinitionObject = 83886592, // 0x05000200
		/// <summary>Generic Feature (constraint) Object.</summary>
		kPartFeatureObject = 83886848, // 0x05000300
		/// <summary>Part Feature Collection Object. Presents the current view of Part Features and allows their creation.</summary>
		kPartFeaturesObject = 83887104, // 0x05000400
		/// <summary>PartComponentDefinitions Collection Object.</summary>
		kPartComponentDefinitionsObject = 83887360, // 0x05000500
		/// <summary>WorkPlane Object.</summary>
		kWorkPlaneObject = 83887616, // 0x05000600
		/// <summary>WorkPlaneProxy Object.</summary>
		kWorkPlaneProxyObject = 83887776, // 0x050006A0
		/// <summary>WorkPlanes Collection Object.</summary>
		kWorkPlanesObject = 83887872, // 0x05000700
		/// <summary>Object that allows you to get and set the information that specifies a work plane defined by three points.</summary>
		kThreePointsWorkPlaneDefObject = 83888128, // 0x05000800
		/// <summary>Object that allows you to get and set the information that specifies a work plane defined by two lines.</summary>
		kTwoLinesWorkPlaneDefObject = 83888384, // 0x05000900
		/// <summary>Object that allows you to get and set the information that specifies a work plane defined normal to a line and through a point.</summary>
		kLineAndPointWorkPlaneDefObject = 83888640, // 0x05000A00
		/// <summary>Object that allows you to get and set the information that specifies a work plane that is parallel to an existing plane and passes through an existing point.</summary>
		kPlaneAndPointWorkPlaneDefObject = 83888896, // 0x05000B00
		/// <summary>Object that allows you to get and set the information that specifies a work plane that passes through a line and is a specified angle to a plane.</summary>
		kLinePlaneAndAngleWorkPlaneDefObject = 83889152, // 0x05000C00
		/// <summary>Object that allows you to get and set the information that specifies a work plane that is parallel to an existing plane and offset a specified distance.</summary>
		kPlaneAndOffsetWorkPlaneDefObject = 83889408, // 0x05000D00
		/// <summary>Object that allows you to get and set the information that specifies a work plane that passes through a line and is tangent to a surface.</summary>
		kLineAndTangentWorkPlaneDefObject = 83889664, // 0x05000E00
		/// <summary>Object that allows you to get and set the information that specifies a work plane that is parallel to a plane and tangent to a surface.</summary>
		kPlaneAndTangentWorkPlaneDefObject = 83889920, // 0x05000F00
		/// <summary>Object that allows you to get and set the information that specifies a fixed work plane.</summary>
		kFixedWorkPlaneDefObject = 83890432, // 0x05001100
		/// <summary>WorkAxes Collection Object.</summary>
		kWorkAxesObject = 83890944, // 0x05001300
		/// <summary>WorkAxis Object.</summary>
		kWorkAxisObject = 83891200, // 0x05001400
		/// <summary>WorkAxisProxy Object.</summary>
		kWorkAxisProxyObject = 83891360, // 0x050014A0
		/// <summary>Object that allows you to get and set the information that specifies a work axis along a line.</summary>
		kLineWorkAxisDefObject = 83891456, // 0x05001500
		/// <summary>Object that allows you to get and set the information that specifies a work axis along a line.</summary>
		kTwoPlanesWorkAxisDefObject = 83891712, // 0x05001600
		/// <summary>Object that allows you to get and set the information that specifies a work axis along a line.</summary>
		kTwoPointsWorkAxisDefObject = 83891968, // 0x05001700
		/// <summary>Object that allows you to get and set the information that specifies a work axis at the axis of a revolved face.</summary>
		kRevolvedFaceWorkAxisDefObject = 83892224, // 0x05001800
		/// <summary>Object that allows you to get and set the information that specifies a work axis that is normal to a plane and passes through a point.</summary>
		kPointAndPlaneWorkAxisDefObject = 83892480, // 0x05001900
		/// <summary>Object that allows you to get and set the information that specifies a work axis that is along a line defined by projecting a line onto a plane along the plane normal.</summary>
		kLineAndPlaneWorkAxisDefObject = 83892736, // 0x05001A00
		/// <summary>Object that allows you to get and set the information that specifies a fixed work axis.</summary>
		kFixedWorkAxisDefObject = 83892992, // 0x05001B00
		/// <summary>WorkPoints Collection Object.</summary>
		kWorkPointsObject = 83893248, // 0x05001C00
		/// <summary>WorkPoint Object.</summary>
		kWorkPointObject = 83893504, // 0x05001D00
		/// <summary>WorkPointProxy Object.</summary>
		kWorkPointProxyObject = 83893664, // 0x05001DA0
		/// <summary>Object that allows you to get and set the information that specifies a work point defined by the intersection of three planes.</summary>
		kThreePlanesWorkPointDefObject = 83893760, // 0x05001E00
		/// <summary>Object that allows you to get and set the information that specifies a work point defined by the intersection of two lines.</summary>
		kTwoLinesWorkPointDefObject = 83894016, // 0x05001F00
		/// <summary>Object that allows you to get and set the information that specifies a work point defined by the intersection of a line and face.</summary>
		kLineAndFaceWorkPointDefObject = 83894272, // 0x05002000
		/// <summary>Object that allows you to get and set the information that specifies a work point defined to be coincident to a point.</summary>
		kPointWorkPointDefObject = 83894528, // 0x05002100
		/// <summary>Object that allows you to get and set the information that specifies a work point defined to be at the midpoint of an edge.</summary>
		kMidPointWorkPointDefObject = 83894784, // 0x05002200
		/// <summary>Object that allows you to get and set the information that specifies a fixed work point.</summary>
		kFixedWorkPointDefObject = 83895040, // 0x05002300
		/// <summary>PlanarSketches Collection Object.</summary>
		kPlanarSketchesObject = 83895296, // 0x05002400
		/// <summary>PartEvents Object.</summary>
		kPartEventsObject = 83895552, // 0x05002500
		/// <summary>SketchLine Object.</summary>
		kSketchLineObject = 83896064, // 0x05002700
		/// <summary>SketchLineProxy Object.</summary>
		kSketchLineProxyObject = 83896176, // 0x05002770
		/// <summary>SketchLines Collection Object.</summary>
		kSketchLinesObject = 83896320, // 0x05002800
		/// <summary>SketchPoint Object.</summary>
		kSketchPointObject = 83896576, // 0x05002900
		/// <summary>SketchPointProxy Object.</summary>
		kSketchPointProxyObject = 83896688, // 0x05002970
		/// <summary>SketchPoints Collection Object.</summary>
		kSketchPointsObject = 83896832, // 0x05002A00
		/// <summary>SketchConstraintEnumerator Object.</summary>
		kSketchConstraintsEnumeratorObject = 83897088, // 0x05002B00
		/// <summary>SketchArcs Collection Object.</summary>
		kSketchArcsObject = 83897344, // 0x05002C00
		/// <summary>SketchSplines Collection Object.</summary>
		kSketchSplinesObject = 83897856, // 0x05002E00
		/// <summary>SketchEllipses Collection Object.</summary>
		kSketchEllipsesObject = 83898112, // 0x05002F00
		/// <summary>SketchCircles Collection Object.</summary>
		kSketchCirclesObject = 83898368, // 0x05003000
		/// <summary>Profile Object.</summary>
		kProfileObject = 83898624, // 0x05003100
		/// <summary>ProfileProxy Object.</summary>
		kProfileProxyObject = 83898736, // 0x05003170
		/// <summary>SketchArc Object.</summary>
		kSketchArcObject = 83898880, // 0x05003200
		/// <summary>SketchArcProxy Object.</summary>
		kSketchArcProxyObject = 83898992, // 0x05003270
		/// <summary>SketchSpline Object.</summary>
		kSketchSplineObject = 83899136, // 0x05003300
		/// <summary>SketchSplineProxy Object.</summary>
		kSketchSplineProxyObject = 83899248, // 0x05003370
		/// <summary>SketchEllipse Object.</summary>
		kSketchEllipseObject = 83899392, // 0x05003400
		/// <summary>SketchEllipseProxy Object.</summary>
		kSketchEllipseProxyObject = 83899504, // 0x05003470
		/// <summary>SketchCircle Object.</summary>
		kSketchCircleObject = 83899648, // 0x05003500
		/// <summary>SketchCircleProxy Object.</summary>
		kSketchCircleProxyObject = 83899760, // 0x05003570
		/// <summary>GeometricConstraints Collection Object.</summary>
		kGeometricConstraintsObject = 83899904, // 0x05003600
		/// <summary>CoincidentConstraint Object.</summary>
		kCoincidentConstraintObject = 83900160, // 0x05003700
		/// <summary>CoincidentConstraintProxy Object.</summary>
		kCoincidentConstraintProxyObject = 83900272, // 0x05003770
		/// <summary>CollinearConstraint Object.</summary>
		kCollinearConstraintObject = 83900416, // 0x05003800
		/// <summary>CollinearConstraintProxy Object.</summary>
		kCollinearConstraintProxyObject = 83900528, // 0x05003870
		/// <summary>ConcentricConstraint Object.</summary>
		kConcentricConstraintObject = 83900672, // 0x05003900
		/// <summary>ConcentricConstraintProxy Object.</summary>
		kConcentricConstraintProxyObject = 83900784, // 0x05003970
		/// <summary>SplineFitPointConstraint Object.</summary>
		kSplineFitPointConstraintObject = 83900928, // 0x05003A00
		/// <summary>SplineFitPointConstraintProxy Object.</summary>
		kSplineFitPointConstraintProxyObject = 83901040, // 0x05003A70
		/// <summary>OffsetConstraint Object.</summary>
		kOffsetConstraintObject = 83901184, // 0x05003B00
		/// <summary>OffsetConstraintProxy Object.</summary>
		kOffsetConstraintProxyObject = 83901296, // 0x05003B70
		/// <summary>EqualLengthConstraint Object.</summary>
		kEqualLengthConstraintObject = 83901440, // 0x05003C00
		/// <summary>EqualLengthConstraintProxy Object.</summary>
		kEqualLengthConstraintProxyObject = 83901552, // 0x05003C70
		/// <summary>EqualRadiusConstraint Object.</summary>
		kEqualRadiusConstraintObject = 83901696, // 0x05003D00
		/// <summary>EqualRadiusConstraintProxy Object.</summary>
		kEqualRadiusConstraintProxyObject = 83901808, // 0x05003D70
		/// <summary>GroundConstraint Object.</summary>
		kGroundConstraintObject = 83901952, // 0x05003E00
		/// <summary>GroundConstraintProxy Object.</summary>
		kGroundConstraintProxyObject = 83902064, // 0x05003E70
		/// <summary>HorizontalConstraint Object.</summary>
		kHorizontalConstraintObject = 83902208, // 0x05003F00
		/// <summary>HorizontalConstraintProxy Object.</summary>
		kHorizontalConstraintProxyObject = 83902320, // 0x05003F70
		/// <summary>HorizontalAlignConstraint Object.</summary>
		kHorizontalAlignConstraintObject = 83902464, // 0x05004000
		/// <summary>HorizontalAlignConstraintProxy Object.</summary>
		kHorizontalAlignConstraintProxyObject = 83902576, // 0x05004070
		/// <summary>MidpointConstraint Object.</summary>
		kMidpointConstraintObject = 83902720, // 0x05004100
		/// <summary>MidpointConstraintProxy Object.</summary>
		kMidpointConstraintProxyObject = 83902832, // 0x05004170
		/// <summary>ParallelConstraint Object.</summary>
		kParallelConstraintObject = 83902976, // 0x05004200
		/// <summary>ParallelConstraintProxy Object.</summary>
		kParallelConstraintProxyObject = 83903088, // 0x05004270
		/// <summary>PerpendicularConstraint Object.</summary>
		kPerpendicularConstraintObject = 83903232, // 0x05004300
		/// <summary>PerpendicularConstraintProxy Object.</summary>
		kPerpendicularConstraintProxyObject = 83903344, // 0x05004370
		/// <summary>SymmetryConstraint Object.</summary>
		kSymmetryConstraintObject = 83903488, // 0x05004400
		/// <summary>SymmetryConstraintProxy Object.</summary>
		kSymmetryConstraintProxyObject = 83903600, // 0x05004470
		/// <summary>TangentSketchConstraint Object.</summary>
		kTangentSketchConstraintObject = 83903744, // 0x05004500
		/// <summary>TangentSketchConstraintProxy Object.</summary>
		kTangentSketchConstraintProxyObject = 83903856, // 0x05004570
		/// <summary>VerticalConstraint Object.</summary>
		kVerticalConstraintObject = 83904000, // 0x05004600
		/// <summary>VerticalConstraintProxy Object.</summary>
		kVerticalConstraintProxyObject = 83904112, // 0x05004670
		/// <summary>VerticalAlignConstraint Object.</summary>
		kVerticalAlignConstraintObject = 83904256, // 0x05004700
		/// <summary>VerticalAlignConstraintProxy Object.</summary>
		kVerticalAlignConstraintProxyObject = 83904368, // 0x05004770
		/// <summary>SketchEllipticalArcs Collection Object.</summary>
		kSketchEllipticalArcsObject = 83904512, // 0x05004800
		/// <summary>SketchEllipticalArc Object.</summary>
		kSketchEllipticalArcObject = 83904768, // 0x05004900
		/// <summary>SketchEllipticalArcProxy Object.</summary>
		kSketchEllipticalArcProxyObject = 83904880, // 0x05004970
		/// <summary>PatternConstraint Object.</summary>
		kPatternConstraintObject = 83905024, // 0x05004A00
		/// <summary>PatternConstraintProxy Object.</summary>
		kPatternConstraintProxyObject = 83905136, // 0x05004A70
		/// <summary>DimensionConstraints Collection Object.</summary>
		kDimensionConstraintsObject = 83905280, // 0x05004B00
		/// <summary>OffsetDimConstraint Object.</summary>
		kOffsetDimConstraintObject = 83905536, // 0x05004C00
		/// <summary>OffsetDimConstraintProxy Object.</summary>
		kOffsetDimConstraintProxyObject = 83905648, // 0x05004C70
		/// <summary>TwoPointDistanceDimConstraint Object.</summary>
		kTwoPointDistanceDimConstraintObject = 83905792, // 0x05004D00
		/// <summary>TwoPointDistanceDimConstraintProxy Object.</summary>
		kTwoPointDistanceDimConstraintProxyObject = 83905904, // 0x05004D70
		/// <summary>TwoLineAngleDimConstraint Object.</summary>
		kTwoLineAngleDimConstraintObject = 83906048, // 0x05004E00
		/// <summary>TwoLineAngleDimConstraintProxy Object.</summary>
		kTwoLineAngleDimConstraintProxyObject = 83906160, // 0x05004E70
		/// <summary>ThreePointAngleDimConstraint Object.</summary>
		kThreePointAngleDimConstraintObject = 83906304, // 0x05004F00
		/// <summary>ThreePointAngleDimConstraintProxy Object.</summary>
		kThreePointAngleDimConstraintProxyObject = 83906416, // 0x05004F70
		/// <summary>DiameterDimConstraint Object.</summary>
		kDiameterDimConstraintObject = 83906560, // 0x05005000
		/// <summary>DiameterDimConstraintProxy Object.</summary>
		kDiameterDimConstraintProxyObject = 83906672, // 0x05005070
		/// <summary>RadiusDimConstraint Object.</summary>
		kRadiusDimConstraintObject = 83906816, // 0x05005100
		/// <summary>RadiusDimConstraintProxy Object.</summary>
		kRadiusDimConstraintProxyObject = 83906928, // 0x05005170
		/// <summary>TangentDistanceDimConstraint Object.</summary>
		kTangentDistanceDimConstraintObject = 83907072, // 0x05005200
		/// <summary>TangentDistanceDimConstraintProxy Object.</summary>
		kTangentDistanceDimConstraintProxyObject = 83907184, // 0x05005270
		/// <summary>SketchEntitiesEnumerator Object.</summary>
		kSketchEntitiesEnumeratorObject = 83908096, // 0x05005600
		/// <summary>ProfilePath Object.</summary>
		kProfilePathObject = 83908352, // 0x05005700
		/// <summary>ProfilePathProxy Object.</summary>
		kProfilePathProxyObject = 83908464, // 0x05005770
		/// <summary>ProfileEntity Object.</summary>
		kProfileEntityObject = 83908608, // 0x05005800
		/// <summary>ProfileEntityProxy Object.</summary>
		kProfileEntityProxyObject = 83908720, // 0x05005870
		/// <summary>Profiles Collection Object.</summary>
		kProfilesObject = 83908864, // 0x05005900
		/// <summary>Part Chamfer Feature Object.</summary>
		kChamferFeatureObject = 83909120, // 0x05005A00
		/// <summary>Part Chamfer Features Collection Object.</summary>
		kChamferFeaturesObject = 83909376, // 0x05005B00
		/// <summary>Part CircularPattern Feature Object.</summary>
		kCircularPatternFeatureObject = 83909632, // 0x05005C00
		/// <summary>Part CircularPattern Features Collection Object.</summary>
		kCircularPatternFeaturesObject = 83909888, // 0x05005D00
		/// <summary>Part Coil Feature Object.</summary>
		kCoilFeatureObject = 83910144, // 0x05005E00
		/// <summary>Part Coil Features Collection Object.</summary>
		kCoilFeaturesObject = 83910400, // 0x05005F00
		/// <summary>Part Extrude Feature Object.</summary>
		kExtrudeFeatureObject = 83910656, // 0x05006000
		/// <summary>Part Extrude Features Collection Object.</summary>
		kExtrudeFeaturesObject = 83910912, // 0x05006100
		/// <summary>Part FaceDraft Feature Object.</summary>
		kFaceDraftFeatureObject = 83911168, // 0x05006200
		/// <summary>Part FaceDraft Features Collection Object.</summary>
		kFaceDraftFeaturesObject = 83911424, // 0x05006300
		/// <summary>Part Fillet Feature Object.</summary>
		kFilletFeatureObject = 83911680, // 0x05006400
		/// <summary>Part Fillet Features Collection Object.</summary>
		kFilletFeaturesObject = 83911936, // 0x05006500
		/// <summary>Part Hole Feature Object.</summary>
		kHoleFeatureObject = 83912192, // 0x05006600
		/// <summary>Part Hole Features Collection Object.</summary>
		kHoleFeaturesObject = 83912448, // 0x05006700
		/// <summary>Part Loft Feature Object.</summary>
		kLoftFeatureObject = 83912704, // 0x05006800
		/// <summary>Part Loft Features Collection Object.</summary>
		kLoftFeaturesObject = 83912960, // 0x05006900
		/// <summary>Part Mirror Feature Object.</summary>
		kMirrorFeatureObject = 83913216, // 0x05006A00
		/// <summary>Part Mirror Features Collection Object.</summary>
		kMirrorFeaturesObject = 83913472, // 0x05006B00
		/// <summary>Part RectangularPattern Feature Object.</summary>
		kRectangularPatternFeatureObject = 83913728, // 0x05006C00
		/// <summary>Part RectangularPattern Features Collection Object.</summary>
		kRectangularPatternFeaturesObject = 83913984, // 0x05006D00
		/// <summary>Part Revolve Feature Object.</summary>
		kRevolveFeatureObject = 83914240, // 0x05006E00
		/// <summary>Part Revolve Features Collection Object.</summary>
		kRevolveFeaturesObject = 83914496, // 0x05006F00
		/// <summary>Part Rib Feature Object.</summary>
		kRibFeatureObject = 83914752, // 0x05007000
		/// <summary>Part Rib Features Collection Object.</summary>
		kRibFeaturesObject = 83915008, // 0x05007100
		/// <summary>Part Shell Feature Object.</summary>
		kShellFeatureObject = 83915264, // 0x05007200
		/// <summary>Part Shell Features Collection Object.</summary>
		kShellFeaturesObject = 83915520, // 0x05007300
		/// <summary>Part Split Feature Object.</summary>
		kSplitFeatureObject = 83915776, // 0x05007400
		/// <summary>Part Split Features Collection Object.</summary>
		kSplitFeaturesObject = 83916032, // 0x05007500
		/// <summary>Part Sweep Feature Object.</summary>
		kSweepFeatureObject = 83916288, // 0x05007600
		/// <summary>Part Sweep Features Collection Object.</summary>
		kSweepFeaturesObject = 83916544, // 0x05007700
		/// <summary>Part Thread Feature Object.</summary>
		kThreadFeatureObject = 83916800, // 0x05007800
		/// <summary>Part Thread Features Collection Object.</summary>
		kThreadFeaturesObject = 83917056, // 0x05007900
		/// <summary>Part Features Enumerator Object. Presents a snapshot of the set of Part Features.</summary>
		kPartFeaturesEnumeratorObject = 83917312, // 0x05007A00
		/// <summary>DistanceExtent object.</summary>
		kDistanceExtentObject = 83917824, // 0x05007C00
		/// <summary>AngleExtent object.</summary>
		kAngleExtentObject = 83918080, // 0x05007D00
		/// <summary>FullSweepExtent object.</summary>
		kFullSweepExtentObject = 83918336, // 0x05007E00
		/// <summary>ToExtent object.</summary>
		kToExtentObject = 83918848, // 0x05008000
		/// <summary>ToNextExtent object.</summary>
		kToNextExtentObject = 83919104, // 0x05008100
		/// <summary>FromToExtent object.</summary>
		kFromToExtentObject = 83919360, // 0x05008200
		/// <summary>ThroughAllExtent object.</summary>
		kThroughAllExtentObject = 83919616, // 0x05008300
		/// <summary>ThreadInfo object.</summary>
		kThreadInfoObject = 83919872, // 0x05008400
		/// <summary>StandardThreadInfo object.</summary>
		kStandardThreadInfoObject = 83920128, // 0x05008500
		/// <summary>HoleTapInfo object.</summary>
		kHoleTapInfoObject = 83920384, // 0x05008600
		/// <summary>TaperedThreadInfo object.</summary>
		kTaperedThreadInfoObject = 83920640, // 0x05008700
		/// <summary>NonParametricBaseFeature object.</summary>
		kNonParametricBaseFeatureObject = 83920896, // 0x05008800
		/// <summary>ReferenceFeature object.</summary>
		kReferenceFeatureObject = 83921152, // 0x05008900
		/// <summary>Reference features collection object.</summary>
		kReferenceFeaturesObject = 83923456, // 0x05009200
		/// <summary>FeaturePatternElements object.</summary>
		kFeaturePatternElementsObject = 83923712, // 0x05009300
		/// <summary>FeaturePatternElement object.</summary>
		kFeaturePatternElementObject = 83923968, // 0x05009400
		/// <summary>FeaturePatternElementProxy object.</summary>
		kFeaturePatternElementProxyObject = 83923974, // 0x05009406
		/// <summary>EllipseRadiusDimConstraint Object.</summary>
		kEllipseRadiusDimConstraintObject = 83924224, // 0x05009500
		/// <summary>EllipseRadiusDimConstraintProxy Object.</summary>
		kEllipseRadiusDimConstraintProxyObject = 83924336, // 0x05009570
		/// <summary>PlanarSketch Object, situated in 3D space.</summary>
		kPlanarSketchObject = 83924736, // 0x05009700
		/// <summary>PlanarSketchProxy Object.</summary>
		kPlanarSketchProxyObject = 83924848, // 0x05009770
		/// <summary>Part Chamfer Feature Definition Distance object.</summary>
		kDistanceChamferDefObject = 83925504, // 0x05009A00
		/// <summary>Part Chamfer Feature Definition DistanceAndAngle object.</summary>
		kDistanceAndAngleChamferDefObject = 83925760, // 0x05009B00
		/// <summary>Part Chamfer Feature Definition TwoDistances object.</summary>
		kTwoDistancesChamferDefObject = 83926016, // 0x05009C00
		/// <summary>Part Fillet Definition Object.</summary>
		kFilletDefinitionObject = 83926272, // 0x05009D00
		/// <summary>Part Fillet Radius EdgeSet Object.</summary>
		kFilletRadiusEdgeSetObject = 83926784, // 0x05009F00
		/// <summary>Part Fillet Constant Radius EdgeSet Object.</summary>
		kFilletConstantRadiusEdgeSetObject = 83927040, // 0x0500A000
		/// <summary>Part Fillet Variable Radius EdgeSet Object.</summary>
		kFilletVariableRadiusEdgeSetObject = 83927296, // 0x0500A100
		/// <summary>Part Fillet Intermediate Radius Specifier Object.</summary>
		kFilletIntermediateRadiusObject = 83927552, // 0x0500A200
		/// <summary>Part Fillet Setback Vertex Object.</summary>
		kFilletSetbackVertexObject = 83928064, // 0x0500A400
		/// <summary>Part Fillet Setback Object.</summary>
		kFilletSetbackObject = 83928320, // 0x0500A500
		/// <summary>Derived Part Components Collection.</summary>
		kDerivedPartComponentsObject = 83928576, // 0x0500A600
		/// <summary>Derived Part Components Collection.</summary>
		kDerivedPartComponentObject = 83928832, // 0x0500A700
		/// <summary>DerivedPartComponentProxy object.</summary>
		kDerivedPartComponentProxyObject = 83928944, // 0x0500A770
		/// <summary>Derived Part Entities Collection.</summary>
		kDerivedPartEntitiesObject = 83929088, // 0x0500A800
		/// <summary>Derived Part Entity Object.</summary>
		kDerivedPartEntityObject = 83929344, // 0x0500A900
		/// <summary>Derived Part Definition Object.</summary>
		kDerivedPartDefinitionObject = 83929600, // 0x0500AA00
		/// <summary>Reference Component Object.</summary>
		kReferenceComponentObject = 83929856, // 0x0500AB00
		/// <summary>Derived Assembly Components Collection Object.</summary>
		kDerivedAssemblyComponentsObject = 83930112, // 0x0500AC00
		/// <summary>Derived Assembly Component Object.</summary>
		kDerivedAssemblyComponentObject = 83930368, // 0x0500AD00
		/// <summary>DerivedAssemblyComponentProxy object.</summary>
		kDerivedAssemblyComponentProxyObject = 83930480, // 0x0500AD70
		/// <summary>Derived Assembly Definition Object.</summary>
		kDerivedAssemblyDefinitionObject = 83930624, // 0x0500AE00
		/// <summary>Derived Assembly Occurrence Object.</summary>
		kDerivedAssemblyOccurrenceObject = 83931136, // 0x0500B000
		/// <summary>Pitch and revolution coil extent object.</summary>
		kPitchAndRevolutionCoilExtentObject = 83931392, // 0x0500B100
		/// <summary>Revolution and height coil extent object.</summary>
		kRevolutionAndHeightCoilExtentObject = 83931648, // 0x0500B200
		/// <summary>Pitch and height coil extent object.</summary>
		kPitchAndHeightCoilExtentObject = 83931904, // 0x0500B300
		/// <summary>Spiral coil extent object.</summary>
		kSpiralCoilExtentObject = 83932160, // 0x0500B400
		/// <summary>Sketches Enumerator Object.</summary>
		kSketchesEnumeratorObject = 83932416, // 0x0500B500
		/// <summary>Reference Components Object.</summary>
		kReferenceComponentsObject = 83932672, // 0x0500B600
		/// <summary>iFeature Component Object.</summary>
		kiFeatureComponentObject = 83932928, // 0x0500B700
		/// <summary>iFeature Components collection Object.</summary>
		kiFeatureComponentsObject = 83933184, // 0x0500B800
		/// <summary>iPart Factory Object.</summary>
		kiPartFactoryObject = 83933440, // 0x0500B900
		/// <summary>iPart Member Object.</summary>
		kiPartMemberObject = 83933696, // 0x0500BA00
		/// <summary>iFeature Template Descriptor Object.</summary>
		kiFeatureTemplateDescriptorObject = 83933952, // 0x0500BB00
		/// <summary>iFeature Template Descriptors collection Object.</summary>
		kiFeatureTemplateDescriptorsObject = 83934208, // 0x0500BC00
		/// <summary>Derived Assembly Occurrence Object.</summary>
		kDerivedAssemblyOccurrencesObject = 83934464, // 0x0500BD00
		/// <summary>Reference features Enumerator Object.</summary>
		kReferenceFeaturesEnumeratorObject = 83934720, // 0x0500BE00
		/// <summary>iPart Table Columns Object.</summary>
		kiPartTableColumnsObject = 83934976, // 0x0500BF00
		/// <summary>iPart Table Column Object.</summary>
		kiPartTableColumnObject = 83935232, // 0x0500C000
		/// <summary>iPart Table Rows Object.</summary>
		kiPartTableRowsObject = 83935488, // 0x0500C100
		/// <summary>iPart Table Row Object.</summary>
		kiPartTableRowObject = 83935744, // 0x0500C200
		/// <summary>iPart Table Cell Object.</summary>
		kiPartTableCellObject = 83936000, // 0x0500C300
		/// <summary>3D Sketch Object.</summary>
		kSketch3DObject = 83936256, // 0x0500C400
		/// <summary>Sketch3DProxy Object.</summary>
		kSketch3DProxyObject = 83936368, // 0x0500C470
		/// <summary>3D Sketches Collection Object.</summary>
		kSketches3DObject = 83936512, // 0x0500C500
		/// <summary>SketchEntities3DEnumerator Object.</summary>
		kSketchEntities3DEnumeratorObject = 83937024, // 0x0500C700
		/// <summary>SketchLine3D Object.</summary>
		kSketchLine3DObject = 83937280, // 0x0500C800
		/// <summary>SketchLine3DProxy Object.</summary>
		kSketchLine3DProxyObject = 83937392, // 0x0500C870
		/// <summary>SketchLines3D Collection Object.</summary>
		kSketchLines3DObject = 83937536, // 0x0500C900
		/// <summary>SketchPoint3D Object.</summary>
		kSketchPoint3DObject = 83937792, // 0x0500CA00
		/// <summary>SketchPoint3DProxy Object.</summary>
		kSketchPoint3DProxyObject = 83937904, // 0x0500CA70
		/// <summary>SketchPoints3D Collection Object.</summary>
		kSketchPoints3DObject = 83938048, // 0x0500CB00
		/// <summary>SketchArc3D Object.</summary>
		kSketchArc3DObject = 83938304, // 0x0500CC00
		/// <summary>SketchArc3DProxy Object.</summary>
		kSketchArc3DProxyObject = 83938416, // 0x0500CC70
		/// <summary>SketchArcs3D Collection Object.</summary>
		kSketchArcs3DObject = 83938560, // 0x0500CD00
		/// <summary>SketchSpline3D Object.</summary>
		kSketchSpline3DObject = 83939072, // 0x0500CF00
		/// <summary>SketchSpline3DProxy Object.</summary>
		kSketchSpline3DProxyObject = 83939184, // 0x0500CF70
		/// <summary>SketchEllipse3D Object.</summary>
		kSketchEllipse3DObject = 83939328, // 0x0500D000
		/// <summary>SketchEllipse3DProxy Object.</summary>
		kSketchEllipse3DProxyObject = 83939440, // 0x0500D070
		/// <summary>SketchEllipticalArc3D Object.</summary>
		kSketchEllipticalArc3DObject = 83939584, // 0x0500D100
		/// <summary>SketchEllipticalArc3DProxy Object.</summary>
		kSketchEllipticalArc3DProxyObject = 83939696, // 0x0500D170
		/// <summary>SketchCircle3D Object.</summary>
		kSketchCircle3DObject = 83939840, // 0x0500D200
		/// <summary>SketchCircle3DProxy Object.</summary>
		kSketchCircle3DProxyObject = 83939952, // 0x0500D270
		/// <summary>SketchSplines3D Collection Object.</summary>
		kSketchSplines3DObject = 83940096, // 0x0500D300
		/// <summary>SketchEllipses3D Collection Object.</summary>
		kSketchEllipses3DObject = 83940352, // 0x0500D400
		/// <summary>SketchEllipticalArcs3D Collection Object.</summary>
		kSketchEllipticalArcs3DObject = 83940608, // 0x0500D500
		/// <summary>SketchCircles3D Collection Object.</summary>
		kSketchCircles3DObject = 83940864, // 0x0500D600
		/// <summary>SketchConstraints3DEnumerator Object.</summary>
		kSketchConstraints3dEnumeratorObject = 83941120, // 0x0500D700
		/// <summary>GeometricConstraints3D Collection Object.</summary>
		kGeometricConstraints3DObject = 83941376, // 0x0500D800
		/// <summary>BendConstraint Object.</summary>
		kBendConstraintObject = 83941888, // 0x0500DA00
		/// <summary>BendConstraintProxy Object.</summary>
		kBendConstraintProxyObject = 83942000, // 0x0500DA70
		/// <summary>CoincidentConstraint3D Object.</summary>
		kCoincidentConstraint3DObject = 83942144, // 0x0500DB00
		/// <summary>CoincidentConstraint3DProxy Object.</summary>
		kCoincidentConstraint3DProxyObject = 83942256, // 0x0500DB70
		/// <summary>Path Object.</summary>
		kPathObject = 83942400, // 0x0500DC00
		/// <summary>Path Proxy Object.</summary>
		kPathProxyObject = 83942402, // 0x0500DC02
		/// <summary>PathEntity Object.</summary>
		kPathEntityObject = 83942656, // 0x0500DD00
		/// <summary>PathEntityProxy Object.</summary>
		kPathEntityProxyObject = 83942661, // 0x0500DD05
		/// <summary>iFeature Definition object.</summary>
		kiFeatureDefinitionObject = 83942912, // 0x0500DE00
		/// <summary>iFeature Inputs object.</summary>
		kiFeatureInputsObject = 83943424, // 0x0500E000
		/// <summary>iFeature Input object.</summary>
		kiFeatureInputObject = 83943680, // 0x0500E100
		/// <summary>iFeature Parameter Input object.</summary>
		kiFeatureParameterInputObject = 83943936, // 0x0500E200
		/// <summary>iFeature Geometry Input object.</summary>
		kiFeatureEntityInputObject = 83944192, // 0x0500E300
		/// <summary>iFeature WorkPlane Input object.</summary>
		kiFeatureWorkPlaneInputObject = 83944448, // 0x0500E400
		/// <summary>iFeature SketchPlane Input object.</summary>
		kiFeatureSketchPlaneInputObject = 83944704, // 0x0500E500
		/// <summary>iFeature Vector Input object.</summary>
		kiFeatureVectorInputObject = 83944960, // 0x0500E600
		/// <summary>WorkSurface Object.</summary>
		kWorkSurfaceObject = 83945216, // 0x0500E700
		/// <summary>WorkSurfaceProxy Object.</summary>
		kWorkSurfaceProxyObject = 83945328, // 0x0500E770
		/// <summary>WorkSurfaces Collection Object.</summary>
		kWorkSurfacesObject = 83945472, // 0x0500E800
		/// <summary>Part Base Features Collection Object.</summary>
		kNonParametricBaseFeaturesObject = 83945728, // 0x0500E900
		/// <summary>MapPointCurve Object.</summary>
		kMapPointCurveObject = 83945984, // 0x0500EA00
		/// <summary>MapPointCurves Collection Object.</summary>
		kMapPointCurvesObject = 83946240, // 0x0500EB00
		/// <summary>A iMateDefinition collection object.</summary>
		kiMateDefinitionsObject = 83946496, // 0x0500EC00
		/// <summary>iMateDefinition object.</summary>
		kiMateDefinitionObject = 83946752, // 0x0500ED00
		/// <summary>AngleiMateDefinition object.</summary>
		kAngleiMateDefinitionObject = 83947008, // 0x0500EE00
		/// <summary>AngleiMateDefinitionProxy Object.</summary>
		kAngleiMateDefinitionProxyObject = 83947136, // 0x0500EE80
		/// <summary>FlushiMateDefinition object.</summary>
		kFlushiMateDefinitionObject = 83947264, // 0x0500EF00
		/// <summary>FlushiMateDefinition Proxy object.</summary>
		kFlushiMateDefinitionProxyObject = 83947392, // 0x0500EF80
		/// <summary>InsertiMateDefinition object.</summary>
		kInsertiMateDefinitionObject = 83947520, // 0x0500F000
		/// <summary>InsertiMateDefinition Proxy object.</summary>
		kInsertiMateDefinitionProxyObject = 83947648, // 0x0500F080
		/// <summary>MateiMateDefinition object.</summary>
		kMateiMateDefinitionObject = 83947776, // 0x0500F100
		/// <summary>MateiMateDefinition Proxy object.</summary>
		kMateiMateDefinitionProxyObject = 83947904, // 0x0500F180
		/// <summary>RotateRotateiMateDefinition object.</summary>
		kRotateRotateiMateDefinitionObject = 83948032, // 0x0500F200
		/// <summary>RotateRotateiMateDefinition Proxy object.</summary>
		kRotateRotateiMateDefinitionProxyObject = 83948160, // 0x0500F280
		/// <summary>RotateTranslateiMateDefinition object.</summary>
		kRotateTranslateiMateDefinitionObject = 83948288, // 0x0500F300
		/// <summary>RotateTranslateiMateDefinition Proxy object.</summary>
		kRotateTranslateiMateDefinitionProxyObject = 83948416, // 0x0500F380
		/// <summary>TangentiMateDefinition object.</summary>
		kTangentiMateDefinitionObject = 83948544, // 0x0500F400
		/// <summary>TangentiMateDefinition Proxy object.</summary>
		kTangentiMateDefinitionProxyObject = 83948672, // 0x0500F480
		/// <summary>CompositeiMateDefinition object.</summary>
		kCompositeiMateDefinitionObject = 83948800, // 0x0500F500
		/// <summary>CompositeiMateDefinition Proxy object.</summary>
		kCompositeiMateDefinitionProxyObject = 83948864, // 0x0500F540
		/// <summary>iMateResults object.</summary>
		kiMateResultsObject = 83949056, // 0x0500F600
		/// <summary>iMateResult object.</summary>
		kiMateResultObject = 83949312, // 0x0500F700
		/// <summary>iMateResultProxy object.</summary>
		kiMateResultProxyObject = 83949321, // 0x0500F709
		/// <summary>iMateResultsEnumerator object.</summary>
		kiMateResultsEnumeratorObject = 83949568, // 0x0500F800
		/// <summary>Profiles3D Collection Object.</summary>
		kProfiles3DObject = 83949824, // 0x0500F900
		/// <summary>Profile3D Object.</summary>
		kProfile3DObject = 83950080, // 0x0500FA00
		/// <summary>Profile3DProxy Object.</summary>
		kProfile3DProxyObject = 83950192, // 0x0500FA70
		/// <summary>ProfilePath3D Object.</summary>
		kProfilePath3DObject = 83950336, // 0x0500FB00
		/// <summary>ProfilePath3DProxy Object.</summary>
		kProfilePath3DProxyObject = 83950448, // 0x0500FB70
		/// <summary>ProfileEntity3D Object.</summary>
		kProfileEntity3DObject = 83950592, // 0x0500FC00
		/// <summary>ProfileEntity3DProxy Object.</summary>
		kProfileEntity3DProxyObject = 83950704, // 0x0500FC70
		/// <summary>Derived Part Uniform Scale Definition Object.</summary>
		kDerivedPartUniformScaleDefObject = 83950848, // 0x0500FD00
		/// <summary>Derived Part Transform Definition Object.</summary>
		kDerivedPartTransformDefObject = 83951104, // 0x0500FE00
		/// <summary>Derived Part Coordinate System Definition Object.</summary>
		kDerivedPartCoordinateSystemDefObject = 83951360, // 0x0500FF00
		/// <summary>Object that allows you to get and set the information that specifies a work plane defined normal to a curve and through a point.</summary>
		kNormalToCurveWorkPlaneDefObject = 83951616, // 0x05010000
		/// <summary>Object that allows you to get and set the information that specifies a work plane defined by two planes.</summary>
		kTwoPlanesWorkPlaneDefObject = 83951872, // 0x05010100
		/// <summary>Object that allows you to get and set the information that specifies a work point defined by a curve and another entity.</summary>
		kCurveAndEntityWorkPointDefObject = 83952128, // 0x05010200
		/// <summary>iMateDefinitionssEnumerator object.</summary>
		kiMateDefinitionsEnumeratorObject = 83952384, // 0x05010300
		/// <summary>Part Decal Feature Object.</summary>
		kDecalFeatureObject = 83952640, // 0x05010400
		/// <summary>Part Decal Features Collection Object.</summary>
		kDecalFeaturesObject = 83952896, // 0x05010500
		/// <summary>Part DeleteFace Feature Object.</summary>
		kDeleteFaceFeatureObject = 83953152, // 0x05010600
		/// <summary>Part DeleteFace Features Collection Object.</summary>
		kDeleteFaceFeaturesObject = 83953408, // 0x05010700
		/// <summary>Part Emboss Feature Object.</summary>
		kEmbossFeatureObject = 83953664, // 0x05010800
		/// <summary>Part Emboss Features Collection Object.</summary>
		kEmbossFeaturesObject = 83953920, // 0x05010900
		/// <summary>Part Knit Feature Object.</summary>
		kKnitFeatureObject = 83954176, // 0x05010A00
		/// <summary>Part Knit Features Collection Object.</summary>
		kKnitFeaturesObject = 83954432, // 0x05010B00
		/// <summary>Part ReplaceFace Feature Object.</summary>
		kReplaceFaceFeatureObject = 83954688, // 0x05010C00
		/// <summary>Part ReplaceFace Features Collection Object.</summary>
		kReplaceFaceFeaturesObject = 83954944, // 0x05010D00
		/// <summary>Part Thicken Feature Object.</summary>
		kThickenFeatureObject = 83955200, // 0x05010E00
		/// <summary>Part Thicken Features Collection Object.</summary>
		kThickenFeaturesObject = 83955456, // 0x05010F00
		/// <summary>SplineFitPointsConstraint3D Object.</summary>
		kSplineFitPointsConstraint3DObject = 83955712, // 0x05011000
		/// <summary>SplineFitPointConstraintProxy Object.</summary>
		kSplineFitPointsConstraint3DProxyObject = 83955824, // 0x05011070
		/// <summary>TangentConstraint3D Object.</summary>
		kTangentConstraint3DObject = 83955968, // 0x05011100
		/// <summary>TangentConstraint3DProxy Object.</summary>
		kTangentConstraint3DProxyObject = 83956080, // 0x05011170
		/// <summary>3d Annotation Object.</summary>
		k3dAViewObject = 83956224, // 0x05011200
		/// <summary>Part Chamfer Feature Proxy Object.</summary>
		kChamferFeatureProxyObject = 83956480, // 0x05011300
		/// <summary>Part Circular Feature Proxy Object.</summary>
		kCircularPatternFeatureProxyObject = 83956736, // 0x05011400
		/// <summary>Part Coil Feature Proxy Object.</summary>
		kCoilFeatureProxyObject = 83956992, // 0x05011500
		/// <summary>Part Decal Feature Proxy Object.</summary>
		kDecalFeatureProxyObject = 83957248, // 0x05011600
		/// <summary>Part Delete Feature Proxy Object.</summary>
		kDeleteFaceFeatureProxyObject = 83957504, // 0x05011700
		/// <summary>Part Emboss Feature Proxy Object.</summary>
		kEmbossFeatureProxyObject = 83957760, // 0x05011800
		/// <summary>Part Extrude Feature Proxy Object.</summary>
		kExtrudeFeatureProxyObject = 83958016, // 0x05011900
		/// <summary>Part Face Draft Feature Proxy Object.</summary>
		kFaceDraftFeatureProxyObject = 83958272, // 0x05011A00
		/// <summary>Part Fillet Feature Proxy Object.</summary>
		kFilletFeatureProxyObject = 83958528, // 0x05011B00
		/// <summary>Part Hole Feature Proxy Object.</summary>
		kHoleFeatureProxyObject = 83958784, // 0x05011C00
		/// <summary>Part Knit Feature Proxy Object.</summary>
		kKnitFeatureProxyObject = 83959040, // 0x05011D00
		/// <summary>Part Loft Feature Proxy Object.</summary>
		kLoftFeatureProxyObject = 83959296, // 0x05011E00
		/// <summary>Part Mirror Feature Proxy Object.</summary>
		kMirrorFeatureProxyObject = 83959552, // 0x05011F00
		/// <summary>Part Rectangular Pattern Feature Proxy Object.</summary>
		kRectangularPatternFeatureProxyObject = 83959808, // 0x05012000
		/// <summary>Part Revolve Feature Proxy Object.</summary>
		kRevolveFeatureProxyObject = 83960064, // 0x05012100
		/// <summary>Part Rib Feature Proxy Object.</summary>
		kRibFeatureProxyObject = 83960320, // 0x05012200
		/// <summary>Part Replace Face Feature Proxy Object.</summary>
		kReplaceFaceFeatureProxyObject = 83960576, // 0x05012300
		/// <summary>Part Shell Feature Proxy Object.</summary>
		kShellFeatureProxyObject = 83960832, // 0x05012400
		/// <summary>Part Split Feature Proxy Object.</summary>
		kSplitFeatureProxyObject = 83961088, // 0x05012500
		/// <summary>Part Sweep Feature Proxy Object.</summary>
		kSweepFeatureProxyObject = 83961344, // 0x05012600
		/// <summary>Part Thicken Feature Proxy Object.</summary>
		kThickenFeatureProxyObject = 83961600, // 0x05012700
		/// <summary>Part Thread Feature Proxy Object.</summary>
		kThreadFeatureProxyObject = 83961856, // 0x05012800
		/// <summary>Part Non Parametric Base Feature Proxy Object.</summary>
		kNonParametricBaseFeatureProxyObject = 83962112, // 0x05012900
		/// <summary>Part Reference Feature Proxy Object.</summary>
		kReferenceFeatureProxyObject = 83962368, // 0x05012A00
		/// <summary>SketchOffsetSpline Object.</summary>
		kSketchOffsetSplineObject = 83962624, // 0x05012B00
		/// <summary>SketchOffsetSplineProxy Object.</summary>
		kSketchOffsetSplineProxyObject = 83962736, // 0x05012B70
		/// <summary>SketchOffsetSplines Collection Object.</summary>
		kSketchOffsetSplinesObject = 83962880, // 0x05012C00
		/// <summary>Shell modeling feature Object.</summary>
		kShellDefinitionObject = 83963136, // 0x05012D00
		/// <summary>Shell Thickness Face Set Object.</summary>
		kShellThicknessFaceSetObject = 83963392, // 0x05012E00
		/// <summary>SketchFixedSplines Collection Object.</summary>
		kSketchFixedSplinesObject = 83963648, // 0x05012F00
		/// <summary>SketchFixedSpline Object.</summary>
		kSketchFixedSplineObject = 83963904, // 0x05013000
		/// <summary>SketchFixedSplineProxy Object.</summary>
		kSketchFixedSplineProxyObject = 83964016, // 0x05013070
		/// <summary>SketchFixedSplines3D Collection Object.</summary>
		kSketchFixedSplines3DObject = 83964160, // 0x05013100
		/// <summary>SketchFixedSpline3D Object.</summary>
		kSketchFixedSpline3DObject = 83964416, // 0x05013200
		/// <summary>SketchFixedSpline3DProxy Object.</summary>
		kSketchFixedSpline3DProxyObject = 83964528, // 0x05013270
		/// <summary>Part Boundary Patch Feature Object.</summary>
		kBoundaryPatchFeatureObject = 83964672, // 0x05013300
		/// <summary>Part Boundary Patch Features Collection Object.</summary>
		kBoundaryPatchFeaturesObject = 83964928, // 0x05013400
		/// <summary>Part Boundary Patch Feature Proxy Object.</summary>
		kBoundaryPatchFeatureProxyObject = 83965184, // 0x05013500
		/// <summary>PerpendicularConstraint3D Object.</summary>
		kPerpendicularConstraint3DObject = 83965440, // 0x05013600
		/// <summary>PerpendicularConstraint3DProxy Object.</summary>
		kPerpendicularConstraint3DProxyObject = 83965552, // 0x05013670
		/// <summary>ParallelConstraint3D Object.</summary>
		kParallelConstraint3DObject = 83965696, // 0x05013700
		/// <summary>ParallelConstraint3DProxy Object.</summary>
		kParallelConstraint3DProxyObject = 83965808, // 0x05013770
		/// <summary>GroundConstraint3D Object.</summary>
		kGroundConstraint3DObject = 83965952, // 0x05013800
		/// <summary>GroundConstraint3DProxy Object.</summary>
		kGroundConstraint3DProxyObject = 83966064, // 0x05013870
		/// <summary>CollinearConstraint3D Object.</summary>
		kCollinearConstraint3DObject = 83966208, // 0x05013900
		/// <summary>CollinearConstraint3DProxy Object.</summary>
		kCollinearConstraint3DProxyObject = 83966320, // 0x05013970
		/// <summary>HolePlacementDefinition Object.</summary>
		kHolePlacementDefinitionObject = 83966464, // 0x05013A00
		/// <summary>SketchHolePlacementDefinition Object.</summary>
		kSketchHolePlacementDefinitionObject = 83966720, // 0x05013B00
		/// <summary>LinearHolePlacementDefinition Object.</summary>
		kLinearHolePlacementDefinitionObject = 83966976, // 0x05013C00
		/// <summary>ConcentricHolePlacementDefinition Object.</summary>
		kConcentricHolePlacementDefinitionObject = 83967232, // 0x05013D00
		/// <summary>PointHolePlacementDefinition Object.</summary>
		kPointHolePlacementDefinitionObject = 83967488, // 0x05013E00
		/// <summary>MoveFaceFeature Object.</summary>
		kMoveFaceFeatureObject = 83967744, // 0x05013F00
		/// <summary>MoveFaceFeatureProxy Object.</summary>
		kMoveFaceFeatureProxyObject = 83968000, // 0x05014000
		/// <summary>MoveFaceFeatures Collection Object.</summary>
		kMoveFaceFeaturesObject = 83968256, // 0x05014100
		/// <summary>Object that allows you to get and set the information that specifies a work plane that passes through a point and is tangent to a surface.</summary>
		kPointAndTangentWorkPlaneDefObject = 83968512, // 0x05014200
		/// <summary>Object that allows you to get and set the information that specifies a work axis that is normal to a surface and passes through a point.</summary>
		kNormalToSurfaceWorkAxisDefObject = 83968768, // 0x05014300
		/// <summary>SketchFillRegions Object.</summary>
		kSketchFillRegions = 83969024, // 0x05014400
		/// <summary>SketchFillRegion Object.</summary>
		kSketchFillRegion = 83969280, // 0x05014500
		/// <summary>Object representing the end of features node (stop node in browser).</summary>
		kEndOfFeaturesObject = 83969536, // 0x05014600
		/// <summary>Sculpt Feature Object.</summary>
		kSculptFeatureObject = 83969792, // 0x05014700
		/// <summary>Sculpt Features Collection Object.</summary>
		kSculptFeaturesObject = 83970048, // 0x05014800
		/// <summary>SculptFeatureProxy Object.</summary>
		kSculptFeatureProxyObject = 83970304, // 0x05014900
		/// <summary>DimensionConstraints3D Object.</summary>
		kDimensionConstraints3DObject = 83970560, // 0x05014A00
		/// <summary>TwoLineAngleDimConstraint3D Object.</summary>
		kTwoLineAngleDimConstraint3DObject = 83971072, // 0x05014C00
		/// <summary>TwoLineAngleDimConstraint3DProxy Object.</summary>
		kTwoLineAngleDimConstraint3DProxyObject = 83971184, // 0x05014C70
		/// <summary>TwoPointDistanceDimConstraint3D__Object.</summary>
		kTwoPointDistanceDimConstraint3DObject = 83971328, // 0x05014D00
		/// <summary>TwoPointDistanceDimConstraint3DProxy Object.</summary>
		kTwoPointDistanceDimConstraint3DProxyObject = 83971440, // 0x05014D70
		/// <summary>PointAndPlaneDistanceDimConstraint3D Object.</summary>
		kPointAndPlaneDistanceDimConstraint3DObject = 83971584, // 0x05014E00
		/// <summary>PointAndPlaneDistanceDimConstraint3DProxy Object.</summary>
		kPointAndPlaneDistanceDimConstraint3DProxyObject = 83971696, // 0x05014E70
		/// <summary>LineLengthDimConstraint3D Object.</summary>
		kLineLengthDimConstraint3DObject = 83971840, // 0x05014F00
		/// <summary>LineLengthDimConstraint3DProxy Object.</summary>
		kLineLengthDimConstraint3DProxyObject = 83971952, // 0x05014F70
		/// <summary>SketchImages collection object.</summary>
		kSketchImagesObject = 83972096, // 0x05015000
		/// <summary>SkecthImage object.</summary>
		kSketchImageObject = 83972352, // 0x05015100
		/// <summary>SketchImageProxy object.</summary>
		kSketchImageProxyObject = 83972368, // 0x05015110
		/// <summary>OffsetSplineDimConstraint Object.</summary>
		kOffsetSplineDimConstraintObject = 83972608, // 0x05015200
		/// <summary>OffsetSplineDimConstraintProxy Object.</summary>
		kOffsetSplineDimConstraintProxyObject = 83972720, // 0x05015270
		/// <summary>TextBoxConstraint Object.</summary>
		kTextBoxConstraintObject = 83972864, // 0x05015300
		/// <summary>TextBoxConstraintProxy Object.</summary>
		kTextBoxConstraintProxyObject = 83972976, // 0x05015370
		/// <summary>CustomConstraint3D Object.</summary>
		kCustomConstraint3DObject = 83973120, // 0x05015400
		/// <summary>CustomConstraint3DProxy Object.</summary>
		kCustomConstraint3DProxyObject = 83973232, // 0x05015470
		/// <summary>SmoothConstraint Object.</summary>
		kSmoothConstraintObject = 83973376, // 0x05015500
		/// <summary>SmoothConstraintProxy Object.</summary>
		kSmoothConstraintProxyObject = 83973488, // 0x05015570
		/// <summary>SweepDefinition Object.</summary>
		kSweepDefinitionObject = 83973632, // 0x05015600
		/// <summary>PathSweepDef Object.</summary>
		kPathSweepDefObject = 83973888, // 0x05015700
		/// <summary>PathAndGuideRailSweepDef Object.</summary>
		kPathAndGuideRailSweepDefObject = 83974144, // 0x05015800
		/// <summary>Trim Feature Object.</summary>
		kTrimFeatureObject = 83974400, // 0x05015900
		/// <summary>Trim Features Collection Object.</summary>
		kTrimFeaturesObject = 83976192, // 0x05016000
		/// <summary>TrimFeatureProxy Object.</summary>
		kTrimFeatureProxyObject = 83976448, // 0x05016100
		/// <summary>PathAndGuideSurfaceSweepDef Object.</summary>
		kPathAndGuideSurfaceSweepDefObject = 83976704, // 0x05016200
		/// <summary>SculptSurface object.</summary>
		kSculptSurfaceObject = 83976960, // 0x05016300
		/// <summary>LoftRails Object.</summary>
		kLoftRailsObject = 83977216, // 0x05016400
		/// <summary>LoftRail Object.</summary>
		kLoftRailObject = 83977472, // 0x05016500
		/// <summary>SmoothConstraint3D Object.</summary>
		kSmoothConstraint3DObject = 83977728, // 0x05016600
		/// <summary>SmoothConstraint3DProxy Object.</summary>
		kSmoothConstraint3DProxyObject = 83977840, // 0x05016670
		/// <summary>Extend Feature Object.</summary>
		kExtendFeatureObject = 83977984, // 0x05016700
		/// <summary>Extend Features Collection Object.</summary>
		kExtendFeaturesObject = 83978240, // 0x05016800
		/// <summary>ExtendFeatureProxy Object.</summary>
		kExtendFeatureProxyObject = 83978496, // 0x05016900
		/// <summary>FilletFaceSet Object.</summary>
		kFilletConstantRadiusFaceSetObject = 83978752, // 0x05016A00
		/// <summary>FilletFaceSet Object.</summary>
		kFilletFaceSetObject = 83978752, // 0x05016A00
		/// <summary>FilletFullRoundSet Object.</summary>
		kFilletFullRoundSetObject = 83979008, // 0x05016B00
		/// <summary>Object that allows you to get and set the information that specifies a work axis that is along a line and passes through a point.</summary>
		kLineAndPointWorkAxisDefObject = 83979264, // 0x05016C00
		/// <summary>Object that allows you to get and set the information that specifies a work axis using an analytic edge.</summary>
		kAnalyticEdgeWorkAxisDefObject = 83979520, // 0x05016D00
		/// <summary>Object that allows you to get and set the information that specifies a work point using a non-linear edge.</summary>
		kNonLinearEdgeWorkPointDefObject = 83979776, // 0x05016E00
		/// <summary>SketchSplineHandle3D Object.</summary>
		kSketchSplineHandle3DObject = 83980032, // 0x05016F00
		/// <summary>SketchSplineHandle3DProxy Object.</summary>
		kSketchSplineHandle3DProxyObject = 83980144, // 0x05016F70
		/// <summary>Part Face Offset Feature Object.</summary>
		kFaceOffsetFeatureObject = 83982080, // 0x05017700
		/// <summary>Part Face Offset Feature Proxy Object.</summary>
		kFaceOffsetFeatureProxyObject = 83982336, // 0x05017800
		/// <summary>Part Face Offset Features Collection Object.</summary>
		kFaceOffsetFeaturesObject = 83982592, // 0x05017900
		/// <summary>SketchSplineHandle Object.</summary>
		kSketchSplineHandleObject = 83984128, // 0x05017F00
		/// <summary>SketchSplineHandleProxy Object.</summary>
		kSketchSplineHandleProxyObject = 83984240, // 0x05017F70
		/// <summary>The ClientFeatures collection object provides access to client features and provides methods to create additional client features.</summary>
		kClientFeaturesObject = 83988480, // 0x05019000
		/// <summary>The ClientFeature object represents a client feature in a part or an assembly document.</summary>
		kClientFeatureObject = 83988736, // 0x05019100
		/// <summary>ClientFeatureProxy object.</summary>
		kClientFeatureProxyObject = 83988848, // 0x05019170
		/// <summary>The ClientFeatureDefinition object is used to define and query all the inputs of a ClientFeature.</summary>
		kClientFeatureDefinitionObject = 83988992, // 0x05019200
		/// <summary>ClientFeatureElements object.</summary>
		kClientFeatureElementsObject = 83989248, // 0x05019300
		/// <summary>ClientFeatureElement object.</summary>
		kClientFeatureElementObject = 83989504, // 0x05019400
		/// <summary>ClientFeatureElementProxy object.</summary>
		kClientFeatureElementProxyObject = 83989616, // 0x05019470
		/// <summary>BoundaryPatch definition object.</summary>
		kBoundaryPatchDefinitionObject = 83989760, // 0x05019500
		/// <summary>BoundaryPatchLoop object.</summary>
		kBoundaryPatchLoopObject = 83990016, // 0x05019600
		/// <summary>PathAndSectionTwistsSweepDef Object.</summary>
		kPathAndSectionTwistsSweepDefObject = 83990272, // 0x05019700
		/// <summary>TorusMidPlaneWorkPlaneDef Object.</summary>
		kTorusMidPlaneWorkPlaneDefObject = 83990528, // 0x05019800
		/// <summary>TorusCenterPointWorkPointDef Object.</summary>
		kTorusCenterPointWorkPointDefObject = 83990784, // 0x05019900
		/// <summary>FeatureDimensions object.</summary>
		kFeatureDimensionsObject = 83991040, // 0x05019A00
		/// <summary>FeatureDimension object.</summary>
		kFeatureDimensionObject = 83991296, // 0x05019B00
		/// <summary>FeatureDimensionProxy object.</summary>
		kFeatureDimensionProxyObject = 83991408, // 0x05019B70
		/// <summary>BendPart Feature Object.</summary>
		kBendPartFeatureObject = 83991552, // 0x05019C00
		/// <summary>BendPart Feature Proxy Object.</summary>
		kBendPartFeatureProxyObject = 83991680, // 0x05019C80
		/// <summary>BendPart Features Object.</summary>
		kBendPartFeaturesObject = 83991808, // 0x05019D00
		/// <summary>RegionProperties Object.</summary>
		kRegionPropertiesObject = 83992064, // 0x05019E00
		/// <summary>LoftSectionDimensions Object.</summary>
		kLoftSectionDimensionsObject = 83992320, // 0x05019F00
		/// <summary>LoftSectionDimension Object.</summary>
		kLoftSectionDimensionObject = 83992576, // 0x0501A000
		/// <summary>BoundaryPatchLoops object.</summary>
		kBoundaryPatchLoopsObject = 83992832, // 0x0501A100
		/// <summary>CentroidWorkPointDef Object.</summary>
		kCentroidWorkPointDefObject = 83993088, // 0x0501A200
		/// <summary>CutAcrossBendsExtent object.</summary>
		kCutAcrossBendsExtentObject = 83993344, // 0x0501A300
		/// <summary>DistanceHeightExtent object.</summary>
		kDistanceHeightExtentObject = 83995648, // 0x0501AC00
		/// <summary>LegacyDistanceHeightExtent object.</summary>
		kLegacyDistanceHeightExtentObject = 83995904, // 0x0501AD00
		/// <summary>ToHeightExtent object.</summary>
		kToHeightExtentObject = 83996160, // 0x0501AE00
		/// <summary>EdgeWidthExtent object.</summary>
		kEdgeWidthExtentObject = 83996416, // 0x0501AF00
		/// <summary>CenteredWidthExtent object.</summary>
		kCenteredWidthExtentObject = 83996672, // 0x0501B000
		/// <summary>WidthOffsetWidthExtent object.</summary>
		kWidthOffsetWidthExtentObject = 83996928, // 0x0501B100
		/// <summary>OffsetWidthExtent object.</summary>
		kOffsetWidthExtentObject = 83997184, // 0x0501B200
		/// <summary>FromToWidthExtent object.</summary>
		kFromToWidthExtentObject = 83997440, // 0x0501B300
		/// <summary>Sketches3D Enumerator Object.</summary>
		kSketches3DEnumeratorObject = 83997696, // 0x0501B400
		/// <summary>iFeatures object.</summary>
		kiFeaturesObject = 83997952, // 0x0501B500
		/// <summary>iFeature object.</summary>
		kiFeatureObject = 83998208, // 0x0501B600
		/// <summary>iFeature proxy object.</summary>
		kiFeatureProxyObject = 83998320, // 0x0501B670
		/// <summary>iFeature table cell object.</summary>
		kiFeatureTableCellObject = 83998464, // 0x0501B700
		/// <summary>iFeature table column object.</summary>
		kiFeatureTableColumnObject = 83998720, // 0x0501B800
		/// <summary>iFeature table columns object.</summary>
		kiFeatureTableColumnsObject = 83998976, // 0x0501B900
		/// <summary>iFeature table row object.</summary>
		kiFeatureTableRowObject = 83999232, // 0x0501BA00
		/// <summary>iFeature table rows object.</summary>
		kiFeatureTableRowsObject = 83999488, // 0x0501BB00
		/// <summary>iFeature table object.</summary>
		kiFeatureTableObject = 83999744, // 0x0501BC00
		/// <summary>MidpointConstraint3D Object.</summary>
		kMidpointConstraint3DObject = 84000000, // 0x0501BD00
		/// <summary>MidpointConstraint3DProxy Object.</summary>
		kMidpointConstraint3DProxyObject = 84000112, // 0x0501BD70
		/// <summary>GraphicsTextureCoordinateSet Object.</summary>
		kGraphicsTextureCoordinateSetObject = 84000256, // 0x0501BE00
		/// <summary>GraphicsColorMapper Object.</summary>
		kGraphicsColorMapperObject = 84000512, // 0x0501BF00
		/// <summary>FlatPatternFeatures collection object provides access to the features that exist within the flat pattern.</summary>
		kFlatPatternFeaturesObject = 84000768, // 0x0501C000
		/// <summary>FactoryOptions Object.</summary>
		kFactoryOptionsObject = 84001024, // 0x0501C100
		/// <summary>CombineFeatures Object.</summary>
		kCombineFeaturesObject = 84001280, // 0x0501C200
		/// <summary>CombineFeature Object.</summary>
		kCombineFeatureObject = 84001536, // 0x0501C300
		/// <summary>CombineFeatureProxy Object.</summary>
		kCombineFeatureProxyObject = 84001696, // 0x0501C3A0
		/// <summary>MoveFeatures Object.</summary>
		kMoveFeaturesObject = 84001792, // 0x0501C400
		/// <summary>MoveFeature Object.</summary>
		kMoveFeatureObject = 84002048, // 0x0501C500
		/// <summary>MoveFeatureProxy Object.</summary>
		kMoveFeatureProxyObject = 84002208, // 0x0501C5A0
		/// <summary>Part Boss Feature Object.</summary>
		kBossFeatureObject = 84002304, // 0x0501C600
		/// <summary>Part Boss Feature Proxy Object.</summary>
		kBossFeatureProxyObject = 84002416, // 0x0501C670
		/// <summary>Part Boss Features Collection Object.</summary>
		kBossFeaturesObject = 84002560, // 0x0501C700
		/// <summary>Part Grill Feature Object.</summary>
		kGrillFeatureObject = 84002816, // 0x0501C800
		/// <summary>Part Grill Feature Proxy Object.</summary>
		kGrillFeatureProxyObject = 84002928, // 0x0501C870
		/// <summary>Part Grill Features Collection Object.</summary>
		kGrillFeaturesObject = 84003072, // 0x0501C900
		/// <summary>Part Lip Feature Object.</summary>
		kLipFeatureObject = 84003328, // 0x0501CA00
		/// <summary>Part Lip Feature Proxy Object.</summary>
		kLipFeatureProxyObject = 84003440, // 0x0501CA70
		/// <summary>Part Lip Features Collection Object.</summary>
		kLipFeaturesObject = 84003584, // 0x0501CB00
		/// <summary>Part Rest Feature Object.</summary>
		kRestFeatureObject = 84003840, // 0x0501CC00
		/// <summary>Part Rest Feature Proxy Object.</summary>
		kRestFeatureProxyObject = 84003952, // 0x0501CC70
		/// <summary>Part Rest Features Collection Object.</summary>
		kRestFeaturesObject = 84004096, // 0x0501CD00
		/// <summary>Part RuleFillet Feature Object.</summary>
		kRuleFilletFeatureObject = 84004352, // 0x0501CE00
		/// <summary>Part RuleFillet Feature Proxy Object.</summary>
		kRuleFilletFeatureProxyObject = 84004464, // 0x0501CE70
		/// <summary>Part RuleFillet Features Collection Object.</summary>
		kRuleFilletFeaturesObject = 84004608, // 0x0501CF00
		/// <summary>Part SnapFit Feature Object.</summary>
		kSnapFitFeatureObject = 84004864, // 0x0501D000
		/// <summary>Part SnapFit Feature Proxy Object.</summary>
		kSnapFitFeatureProxyObject = 84004976, // 0x0501D070
		/// <summary>Part SnapFit Features Collection Object.</summary>
		kSnapFitFeaturesObject = 84005120, // 0x0501D100
		/// <summary>ProjectedCuts Object.</summary>
		kProjectedCutsObject = 84005376, // 0x0501D200
		/// <summary>ProjectedCut Object.</summary>
		kProjectedCutObject = 84005632, // 0x0501D300
		/// <summary>ProjectedCutProxy Object.</summary>
		kProjectedCutProxyObject = 84005888, // 0x0501D400
		/// <summary>SketchBlockDefinitions Collection Object.</summary>
		kSketchBlockDefinitionsObject = 84006144, // 0x0501D500
		/// <summary>Sketch Block Definition Object.</summary>
		kSketchBlockDefinitionObject = 84006400, // 0x0501D600
		/// <summary>SketchBlockDefinitionProxy Object.</summary>
		kSketchBlockDefinitionProxyObject = 84006512, // 0x0501D670
		/// <summary>SketchBlocks Collection Object.</summary>
		kSketchBlocksObject = 84006656, // 0x0501D700
		/// <summary>Sketch Block Object.</summary>
		kSketchBlockObject = 84006912, // 0x0501D800
		/// <summary>SketchBlockProxy Object.</summary>
		kSketchBlockProxyObject = 84007024, // 0x0501D870
		/// <summary>Sketches Enumerator Object.</summary>
		kSketchBlocksEnumeratorObject = 84007168, // 0x0501D900
		/// <summary>SketchBlockDefinitions Enumerator Object.</summary>
		kSketchBlockDefinitionsEnumeratorObject = 84007424, // 0x0501DA00
		/// <summary>Derived Alias Studio Collection.</summary>
		kDerivedAliasComponentsObject = 84007680, // 0x0501DB00
		/// <summary>Derived Alias Studio Component.</summary>
		kDerivedAliasComponentObject = 84007936, // 0x0501DC00
		/// <summary>Derived Alias Studio Component Proxy.</summary>
		kDerivedAliasComponentProxyObject = 84007938, // 0x0501DC02
		/// <summary>UserCoordinateSystem Object.</summary>
		kUserCoordinateSystemObject = 84008192, // 0x0501DD00
		/// <summary>UserCoordinateSystemProxy Object.</summary>
		kUserCoordinateSystemProxyObject = 84008352, // 0x0501DDA0
		/// <summary>UserCoordinateSystems Collection Object.</summary>
		kUserCoordinateSystemsObject = 84008448, // 0x0501DE00
		/// <summary>RadiusDimConstraint3D Object.</summary>
		kRadiusDimConstraint3DObject = 84008704, // 0x0501DF00
		/// <summary>RadiusDimConstraint3DProxy Object.</summary>
		kRadiusDimConstraint3DProxyObject = 84008816, // 0x0501DF70
		/// <summary>ConcentricConstraint3D Object.</summary>
		kConcentricConstraint3DObject = 84008960, // 0x0501E000
		/// <summary>ConcentricConstraint3DProxy Object.</summary>
		kConcentricConstraint3DProxyObject = 84009072, // 0x0501E070
		/// <summary>HelicalConstraint3D Object.</summary>
		kHelicalConstraint3DObject = 84009216, // 0x0501E100
		/// <summary>HelicalConstraint3DProxy Object.</summary>
		kHelicalConstraint3DProxyObject = 84009328, // 0x0501E170
		/// <summary>Part CoreCavity Feature Definition Object.</summary>
		kCoreCavityDefinitionObject = 84009728, // 0x0501E300
		/// <summary>Part CoreCavity Feature Proxy Object.</summary>
		kCoreCavityFeatureProxyObject = 84009984, // 0x0501E400
		/// <summary>Part CoreCavity Feature Object.</summary>
		kCoreCavityFeatureObject = 84010240, // 0x0501E500
		/// <summary>Part CoreCavity Features Collection Object.</summary>
		kCoreCavityFeaturesObject = 84010496, // 0x0501E600
		/// <summary>UserCoordinateSystemDefinition Object.</summary>
		kUserCoordinateSystemDefinitionObject = 84010752, // 0x0501E700
		/// <summary>Part AliasFreeform Feature Proxy Object.</summary>
		kAliasFreeformFeatureProxyObject = 84011008, // 0x0501E800
		/// <summary>Part AliasFreeform Feature Object.</summary>
		kAliasFreeformFeatureObject = 84011264, // 0x0501E900
		/// <summary>Part AliasFreeform Features Collection Object.</summary>
		kAliasFreeformFeaturesObject = 84011520, // 0x0501EA00
		/// <summary>MoveFace Definition Object.</summary>
		kMoveFaceDefinitionObject = 84011776, // 0x0501EB00
		/// <summary>Direction and distance move definition object.</summary>
		kDirectionAndDistanceMoveDefinitionObject = 84012032, // 0x0501EC00
		/// <summary>Planar move definition object.</summary>
		kPlanarMoveDefinitionObject = 84012288, // 0x0501ED00
		/// <summary>Free move definition object.</summary>
		kFreeMoveDefinitionObject = 84012544, // 0x0501EE00
		/// <summary>SilhouetteCurve Object.</summary>
		kSilhouetteCurveObject = 84012800, // 0x0501EF00
		/// <summary>SilhouetteCurveProxy Object.</summary>
		kSilhouetteCurveProxyObject = 84012929, // 0x0501EF81
		/// <summary>SilhouetteCurves Collection Object.</summary>
		kSilhouetteCurvesObject = 84013056, // 0x0501F000
		/// <summary>RibDefinition Object.</summary>
		kRibDefinitionObject = 84013312, // 0x0501F100
		/// <summary>Boss Sets Collection Object.</summary>
		kBossSetsObject = 84013568, // 0x0501F200
		/// <summary>Boss Set Object.</summary>
		kBossSetObject = 84013824, // 0x0501F300
		/// <summary>ExtrudeDefinition Object.</summary>
		kExtrudeDefinitionObject = 84014080, // 0x0501F400
		/// <summary>ArcLengthDimConstraint Object.</summary>
		kArcLengthDimConstraintObject = 84014336, // 0x0501F500
		/// <summary>ArcLengthDimConstraintProxy Object.</summary>
		kArcLengthDimConstraintProxyObject = 84014448, // 0x0501F570
		/// <summary>MoveDefinition Object.</summary>
		kMoveDefinitionObject = 84014592, // 0x0501F600
		/// <summary>MoveOperation Object.</summary>
		kMoveOperationObject = 84014848, // 0x0501F700
		/// <summary>FreeDragMoveOperation Object.</summary>
		kFreeDragMoveOperationObject = 84015104, // 0x0501F800
		/// <summary>MoveAlongRayMoveOperation Object.</summary>
		kMoveAlongRayMoveOperationObject = 84015360, // 0x0501F900
		/// <summary>RotateAboutLineMoveOperation Object.</summary>
		kRotateAboutLineMoveOperationObject = 84015616, // 0x0501FA00
		/// <summary>Part MidSurface Feature Proxy Object.</summary>
		kMidSurfaceFeatureProxyObject = 84015872, // 0x0501FB00
		/// <summary>Part MidSurface Feature Object.</summary>
		kMidSurfaceFeatureObject = 84016128, // 0x0501FC00
		/// <summary>Part MidSurface Features Collection Object.</summary>
		kMidSurfaceFeaturesObject = 84016384, // 0x0501FD00
		/// <summary>Face Offset modeling feature Object.</summary>
		kFaceOffsetDefinitionObject = 84016640, // 0x0501FE00
		/// <summary>MidSurface Thickness Collection Object.</summary>
		kMidSurfaceThicknessesObject = 84016896, // 0x0501FF00
		/// <summary>MidSurface Thickness Object.</summary>
		kMidSurfaceThicknessObject = 84017152, // 0x05020000
		/// <summary>The PointCloud object represents a single point cloud within Oblikovati.</summary>
		kPointCloudObject = 84017408, // 0x05020100
		/// <summary>PointCloudProxy Object.</summary>
		kPointCloudProxyObject = 84017520, // 0x05020170
		/// <summary>The PointCloudPoint object represents a point within a point cloud.</summary>
		kPointCloudPointObject = 84017664, // 0x05020200
		/// <summary>PointCloudPointProxy Object.</summary>
		kPointCloudPointProxyObject = 84017776, // 0x05020270
		/// <summary>SketchControlPointSpline Object.</summary>
		kSketchControlPointSplineObject = 84017920, // 0x05020300
		/// <summary>SketchControlPointSplineProxy Object.</summary>
		kSketchControlPointSplineProxyObject = 84018032, // 0x05020370
		/// <summary>SketchControlPointSplines Collection Object.</summary>
		kSketchControlPointSplinesObject = 84018176, // 0x05020400
		/// <summary>SketchControlPointSpline3D Object.</summary>
		kSketchControlPointSpline3DObject = 84018432, // 0x05020500
		/// <summary>SketchControlPointSpline3DProxy Object.</summary>
		kSketchControlPointSpline3DProxyObject = 84018544, // 0x05020570
		/// <summary>SketchControlPointSplines3D Collection Object.</summary>
		kSketchControlPointSplines3DObject = 84018688, // 0x05020600
		/// <summary>SketchEquationCurve Object.</summary>
		kSketchEquationCurveObject = 84018944, // 0x05020700
		/// <summary>SketchEquationCurveProxy Object.</summary>
		kSketchEquationCurveProxyObject = 84019056, // 0x05020770
		/// <summary>SketchEquationCurves Collection Object.</summary>
		kSketchEquationCurvesObject = 84019200, // 0x05020800
		/// <summary>SketchEquationCurve3D Object.</summary>
		kSketchEquationCurve3DObject = 84019456, // 0x05020900
		/// <summary>SketchEquationCurve3DProxy Object.</summary>
		kSketchEquationCurve3DProxyObject = 84019568, // 0x05020970
		/// <summary>SketchEquationCurves3D Collection Object.</summary>
		kSketchEquationCurves3DObject = 84019712, // 0x05020A00
		/// <summary>ModelAnnotations Object.</summary>
		kModelAnnotationsObject = 84019968, // 0x05020B00
		/// <summary>ModelAnnotation Object.</summary>
		kModelAnnotationObject = 84020224, // 0x05020C00
		/// <summary>ModelDimensions Object.</summary>
		kModelDimensionsObject = 84020480, // 0x05020D00
		/// <summary>ModelDimension Object.</summary>
		kModelDimensionObject = 84020736, // 0x05020E00
		/// <summary>LinearModelDimensions Object.</summary>
		kLinearModelDimensionsObject = 84020992, // 0x05020F00
		/// <summary>ModelDimensionDefinition Object.</summary>
		kModelDimensionDefinitionObject = 84021248, // 0x05021000
		/// <summary>LinearModelDimensionDefinition Object.</summary>
		kLinearModelDimensionDefinitionObject = 84021504, // 0x05021100
		/// <summary>LinearModelDimension Object.</summary>
		kLinearModelDimensionObject = 84021760, // 0x05021200
		/// <summary>LinearModelDimension Object.</summary>
		kLinearModelDimensionProxyObject = 84021872, // 0x05021270
		/// <summary>AngularModelDimensions Object.</summary>
		kAngularModelDimensionsObject = 84022016, // 0x05021300
		/// <summary>RadiusModelDimensions Object.</summary>
		kRadiusModelDimensionsObject = 84022272, // 0x05021400
		/// <summary>DiameterModelDimensions Object.</summary>
		kDiameterModelDimensionsObject = 84022528, // 0x05021500
		/// <summary>AngularModelDimension Object.</summary>
		kAngularModelDimensionObject = 84022784, // 0x05021600
		/// <summary>AngularModelDimensionProxy Object.</summary>
		kAngularModelDimensionProxyObject = 84022896, // 0x05021670
		/// <summary>RadiusModelDimension Object.</summary>
		kRadiusModelDimensionObject = 84023040, // 0x05021700
		/// <summary>RadiusModelDimensionProxy Object.</summary>
		kRadiusModelDimensionProxyObject = 84023152, // 0x05021770
		/// <summary>DiameterModelDimension Object.</summary>
		kDiameterModelDimensionObject = 84023296, // 0x05021800
		/// <summary>DiameterModelDimensionProxy Object.</summary>
		kDiameterModelDimensionProxyObject = 84023408, // 0x05021870
		/// <summary>ModelFeatureControlFrames Object.</summary>
		kModelFeatureControlFramesObject = 84023808, // 0x05021A00
		/// <summary>ModelFeatureControlFrame Object.</summary>
		kModelFeatureControlFrameObject = 84024064, // 0x05021B00
		/// <summary>ModelFeatureControlFrameProxy Object.</summary>
		kModelFeatureControlFrameProxyObject = 84024176, // 0x05021B70
		/// <summary>ModelLeader Object.</summary>
		kModelLeaderObject = 84024320, // 0x05021C00
		/// <summary>ModelLeaderNode Object.</summary>
		kModelLeaderNodeObject = 84024576, // 0x05021D00
		/// <summary>ModelLeaderNodesEnumerator Object.</summary>
		kModelLeaderNodesEnumeratorObject = 84024832, // 0x05021E00
		/// <summary>ModelLeaderNote Object.</summary>
		kModelLeaderNoteObject = 84025088, // 0x05021F00
		/// <summary>ModelLeaderNoteProxy Object.</summary>
		kModelLeaderNoteProxyObject = 84025200, // 0x05021F70
		/// <summary>ModelLeaderNotes Object.</summary>
		kModelLeaderNotesObject = 84025344, // 0x05022000
		/// <summary>AngularModelDimensionDefinition Object.</summary>
		kAngularModelDimensionDefinitionObject = 84025600, // 0x05022100
		/// <summary>RadiusModelDimensionDefinition Object.</summary>
		kRadiusModelDimensionDefinitionObject = 84025856, // 0x05022200
		/// <summary>DiameterModelDimensionDefinition Object.</summary>
		kDiameterModelDimensionDefinitionObject = 84026112, // 0x05022300
		/// <summary>IntersectionCurves Object.</summary>
		kIntersectionCurvesObject = 84026368, // 0x05022400
		/// <summary>IntersectionCurve Object.</summary>
		kIntersectionCurveObject = 84026624, // 0x05022500
		/// <summary>IntersectionCurveProxy Object.</summary>
		kIntersectionCurveProxyObject = 84026736, // 0x05022570
		/// <summary>ModelLeaderNoteDefinition Object.</summary>
		kModelLeaderNoteDefinitionObject = 84026880, // 0x05022600
		/// <summary>ModelAnnotationText Object.</summary>
		kModelAnnotationTextObject = 84027136, // 0x05022700
		/// <summary>SphereCenterPointWorkPointDef Object.</summary>
		kSphereCenterPointWorkPointDefObject = 84027392, // 0x05022800
		/// <summary>AnnotationPlanes Object.</summary>
		kAnnotationPlanesObject = 84027648, // 0x05022900
		/// <summary>AnnotationPlane Object.</summary>
		kAnnotationPlaneObject = 84027904, // 0x05022A00
		/// <summary>AnnotationPlaneProxy Object.</summary>
		kAnnotationPlaneProxyObject = 84028016, // 0x05022A70
		/// <summary>ModelSurfaceTextureSymbol Object.</summary>
		kModelSurfaceTextureSymbolObject = 84028160, // 0x05022B00
		/// <summary>ModelSurfaceTextureSymbolProxy Object.</summary>
		kModelSurfaceTextureSymbolProxyObject = 84028272, // 0x05022B70
		/// <summary>ModelSurfaceTextureSymbolDefinition Object.</summary>
		kModelSurfaceTextureSymbolDefinitionObject = 84028416, // 0x05022C00
		/// <summary>ModelSurfaceTextureSymbols Object.</summary>
		kModelSurfaceTextureSymbolsObject = 84028672, // 0x05022D00
		/// <summary>ModelHoleThreadNote Object.</summary>
		kModelHoleThreadNoteObject = 84028928, // 0x05022E00
		/// <summary>ModelHoleThreadNoteProxy Object.</summary>
		kModelHoleThreadNoteProxyObject = 84029040, // 0x05022E70
		/// <summary>ModelHoleThreadNotes Object.</summary>
		kModelHoleThreadNotesObject = 84029184, // 0x05022F00
		/// <summary>ModelHoleThreadNoteDefinition Object.</summary>
		kModelHoleThreadNoteDefinitionObject = 84029440, // 0x05023000
		/// <summary>AnnotationPlaneDefinition Object.</summary>
		kAnnotationPlaneDefinitionObject = 84029696, // 0x05023100
		/// <summary>AnnotationPlaneDefinitionsEnumerator Object.</summary>
		kAnnotationPlaneDefinitionsEnumeratorObject = 84029952, // 0x05023200
		/// <summary>ModelAnnotationsEnumerator Object.</summary>
		kModelAnnotationsEnumeratorObject = 84030208, // 0x05023300
		/// <summary>ModelFeatureControlFrameRows Object.</summary>
		kModelFeatureControlFrameRowsObject = 84030464, // 0x05023400
		/// <summary>ModelFeatureControlFrameRow Object.</summary>
		kModelFeatureControlFrameRowObject = 84030720, // 0x05023500
		/// <summary>ModelFeatureControlFrameDefinition Object.</summary>
		kModelFeatureControlFrameDefinitionObject = 84030976, // 0x05023600
		/// <summary>ModelDatumIdentifiers Object.</summary>
		kModelDatumIdentifiersObject = 84031232, // 0x05023700
		/// <summary>ModelDatumIdentifier Object.</summary>
		kModelDatumIdentifierObject = 84031488, // 0x05023800
		/// <summary>ModelDatumIdentifierProxy Object.</summary>
		kModelDatumIdentifierProxyObject = 84031600, // 0x05023870
		/// <summary>ModelDatumIdentifierDefinition Object.</summary>
		kModelDatumIdentifierDefinitionObject = 84031744, // 0x05023900
		/// <summary>The PointCloudPlane object represents a plane within a point cloud.</summary>
		kPointCloudPlaneObject = 84032000, // 0x05023A00
		/// <summary>PointCloudPlaneProxy Object.</summary>
		kPointCloudPlaneProxyObject = 84032112, // 0x05023A70
		/// <summary>Part Freeform Feature Object.</summary>
		kFreeformFeatureObject = 84032256, // 0x05023B00
		/// <summary>Part Freeform Features Collection Object.</summary>
		kFreeformFeaturesObject = 84032512, // 0x05023C00
		/// <summary>Part Freeform Feature Proxy Object.</summary>
		kFreeformFeatureProxyObject = 84032768, // 0x05023D00
		/// <summary>Part DirectEdit Feature Object.</summary>
		kDirectEditFeatureObject = 84033024, // 0x05023E00
		/// <summary>Part DirectEdit Features Collection Object.</summary>
		kDirectEditFeaturesObject = 84033280, // 0x05023F00
		/// <summary>Part DirectEdit Feature Proxy Object.</summary>
		kDirectEditFeatureProxyObject = 84033536, // 0x05024000
		/// <summary>Part DirectEdit Operation Object.</summary>
		kDirectEditOperationObject = 84033792, // 0x05024100
		/// <summary>Part DirectEdit Operations Collection Object.</summary>
		kDirectEditOperationsObject = 84034048, // 0x05024200
		/// <summary>Part DirectEdit Operation Proxy Object.</summary>
		kDirectEditOperationProxyObject = 84034304, // 0x05024300
		/// <summary>Part DirectEdit Move Operation Object.</summary>
		kDirectEditMoveOperationObject = 84034560, // 0x05024400
		/// <summary>Part DirectEdit Move Operation Proxy Object.</summary>
		kDirectEditMoveOperationProxyObject = 84034816, // 0x05024500
		/// <summary>Part DirectEdit Size Operation Object.</summary>
		kDirectEditSizeOperationObject = 84035072, // 0x05024600
		/// <summary>Part DirectEdit Size Operation Proxy Object.</summary>
		kDirectEditSizeOperationProxyObject = 84035328, // 0x05024700
		/// <summary>Part DirectEdit Rotate Operation Object.</summary>
		kDirectEditRotateOperationObject = 84035584, // 0x05024800
		/// <summary>Part DirectEdit Rotate Operation Proxy Object.</summary>
		kDirectEditRotateOperationProxyObject = 84035840, // 0x05024900
		/// <summary>Part DirectEdit Delete Operation Object.</summary>
		kDirectEditDeleteOperationObject = 84036096, // 0x05024A00
		/// <summary>Part DirectEdit Delete Operation Proxy Object.</summary>
		kDirectEditDeleteOperationProxyObject = 84036352, // 0x05024B00
		/// <summary>PointClouds Collection Object.</summary>
		kPointCloudsObject = 84036608, // 0x05024C00
		/// <summary>PointCloudRegions Collection Object.</summary>
		kPointCloudRegionsObject = 84036864, // 0x05024D00
		/// <summary>PointCloudRegion Object.</summary>
		kPointCloudRegionObject = 84037120, // 0x05024E00
		/// <summary>PointCloudScans Collection Object.</summary>
		kPointCloudScansObject = 84037376, // 0x05024F00
		/// <summary>PointCloudScan Object.</summary>
		kPointCloudScanObject = 84037632, // 0x05025000
		/// <summary>PointCloudCrops Collection Object.</summary>
		kPointCloudCropsObject = 84037888, // 0x05025100
		/// <summary>PointCloudCrop Object.</summary>
		kPointCloudCropObject = 84038144, // 0x05025200
		/// <summary>ImportedDWGComponent Object.</summary>
		kImportedDWGComponentObject = 84038400, // 0x05025300
		/// <summary>ImportedDWGComponentProxy Object.</summary>
		kImportedDWGComponentProxyObject = 84038656, // 0x05025400
		/// <summary>ImportedDWGComponentDefinition Object.</summary>
		kImportedDWGComponentDefinitionObject = 84038912, // 0x05025500
		/// <summary>DWGEntitiesEnumerator Collection Object.</summary>
		kDWGEntitiesEnumeratorObject = 84039168, // 0x05025600
		/// <summary>DWGEntity Object.</summary>
		kDWGEntityObject = 84039424, // 0x05025700
		/// <summary>DWGEntityProxy Object.</summary>
		kDWGEntityProxyObject = 84039680, // 0x05025800
		/// <summary>DWGBlockReferencesEnumerator Collection Object.</summary>
		kDWGBlockReferencesEnumeratorObject = 84039936, // 0x05025900
		/// <summary>DWGBlockReference Object.</summary>
		kDWGBlockReferenceObject = 84040192, // 0x05025A00
		/// <summary>DWGBlockReferenceProxy Object.</summary>
		kDWGBlockReferenceProxyObject = 84040448, // 0x05025B00
		/// <summary>DWGBlockDefinition Object.</summary>
		kDWGBlockDefinitionObject = 84040704, // 0x05025C00
		/// <summary>DWGBlockDefinitionProxy Object.</summary>
		kDWGBlockDefinitionProxyObject = 84040960, // 0x05025D00
		/// <summary>DWGArcsEnumerator Collection Object.</summary>
		kDWGArcsEnumeratorObject = 84041216, // 0x05025E00
		/// <summary>DWGArc Object.</summary>
		kDWGArcObject = 84041472, // 0x05025F00
		/// <summary>DWGArcProxy Object.</summary>
		kDWGArcProxyObject = 84041728, // 0x05026000
		/// <summary>DWGEllipticalArcsEnumerator Collection Object.</summary>
		kDWGEllipticalArcsEnumeratorObject = 84041984, // 0x05026100
		/// <summary>DWGEllipticalArc Object.</summary>
		kDWGEllipticalArcObject = 84042240, // 0x05026200
		/// <summary>DWGEllipticalArcProxy Object.</summary>
		kDWGEllipticalArcProxyObject = 84042496, // 0x05026300
		/// <summary>DWGLinesEnumerator Collection Object.</summary>
		kDWGLinesEnumeratorObject = 84042752, // 0x05026400
		/// <summary>DWGLine Object.</summary>
		kDWGLineObject = 84043008, // 0x05026500
		/// <summary>DWGLineProxy Object.</summary>
		kDWGLineProxyObject = 84043264, // 0x05026600
		/// <summary>DWGPointsEnumerator Collection Object.</summary>
		kDWGPointsEnumeratorObject = 84043520, // 0x05026700
		/// <summary>DWGPoint Object.</summary>
		kDWGPointObject = 84043776, // 0x05026800
		/// <summary>DWGPointProxy Object.</summary>
		kDWGPointProxyObject = 84044032, // 0x05026900
		/// <summary>DWGPolylinesEnumerator Collection Object.</summary>
		kDWGPolylinesEnumeratorObject = 84044288, // 0x05026A00
		/// <summary>DWGPolyline Object.</summary>
		kDWGPolylineObject = 84044544, // 0x05026B00
		/// <summary>DWGPolylineProxy Object.</summary>
		kDWGPolylineProxyObject = 84044800, // 0x05026C00
		/// <summary>DWGSplinesEnumerator Collection Object.</summary>
		kDWGSplinesEnumeratorObject = 84045056, // 0x05026D00
		/// <summary>DWGSpline Object.</summary>
		kDWGSplineObject = 84045312, // 0x05026E00
		/// <summary>DWGSplineProxy Object.</summary>
		kDWGSplineProxyObject = 84045568, // 0x05026F00
		/// <summary>DWGPolylines2DEnumerator Collection Object.</summary>
		kDWGPolylines2DEnumeratorObject = 84045824, // 0x05027000
		/// <summary>DWGPolyline2D Object.</summary>
		kDWGPolyline2DObject = 84046080, // 0x05027100
		/// <summary>The FaceDraftDefinition object represents the definition for a FaceDraftFeature.</summary>
		kFaceDraftDefinitionObject = 84046336, // 0x05027200
		/// <summary>ImportedComponent Object.</summary>
		kImportedComponentObject = 84046592, // 0x05027300
		/// <summary>ImportedComponents Object.</summary>
		kImportedComponentsObject = 84046848, // 0x05027400
		/// <summary>ImportedComponentDefinition Object.</summary>
		kImportedComponentDefinitionObject = 84047104, // 0x05027500
		/// <summary>ImportedModelEntities Object.</summary>
		kImportedModelEntitiesObject = 84047360, // 0x05027600
		/// <summary>ImportedModelEntity Object.</summary>
		kImportedModelEntityObject = 84047616, // 0x05027700
		/// <summary>ImportedGenericComponentDefinition Object.</summary>
		kImportedGenericComponentDefinitionObject = 84047872, // 0x05027800
		/// <summary>DWGPolyline2DProxy Object.</summary>
		kDWGPolyline2DProxyObject = 84048128, // 0x05027900
		/// <summary>DWGPolylines3DEnumerator Collection Object.</summary>
		kDWGPolylines3DEnumeratorObject = 84048384, // 0x05027A00
		/// <summary>DWGPolyline3D Object.</summary>
		kDWGPolyline3DObject = 84048640, // 0x05027B00
		/// <summary>DWGPolyline3DProxy Object.</summary>
		kDWGPolyline3DProxyObject = 84048896, // 0x05027C00
		/// <summary>Part DirectEdit Scale Operation Object.</summary>
		kDirectEditScaleOperationObject = 84049152, // 0x05027D00
		/// <summary>Part DirectEdit Scale Operation Proxy Object.</summary>
		kDirectEditScaleOperationProxyObject = 84049408, // 0x05027E00
		/// <summary>ImportedGenericComponent Object.</summary>
		kImportedGenericComponentObject = 84049664, // 0x05027F00
		/// <summary>ImportedGenericComponentProxy Object.</summary>
		kImportedGenericComponentProxyObject = 84049920, // 0x05028000
		/// <summary>Part RuledSurfaceDefinition Object.</summary>
		kRuledSurfaceDefinitionObject = 84050176, // 0x05028100
		/// <summary>Part RuledSurfaceFeature Object.</summary>
		kRuledSurfaceFeatureObject = 84050432, // 0x05028200
		/// <summary>Part RuledSurfaceFeatures Collection Object.</summary>
		kRuledSurfaceFeaturesObject = 84050688, // 0x05028300
		/// <summary>Part RuledSurfaceFeatureProxy Object.</summary>
		kRuledSurfaceFeatureProxyObject = 84050944, // 0x05028400
		/// <summary>OnFaceCurves Object.</summary>
		kOnFaceCurvesObject = 84051200, // 0x05028500
		/// <summary>OnFaceCurve Object.</summary>
		kOnFaceCurveObject = 84051456, // 0x05028600
		/// <summary>OnFaceCurveProxy Object.</summary>
		kOnFaceCurveProxyObject = 84051568, // 0x05028670
		/// <summary>Part SketchDrivenPattern Feature Object.</summary>
		kSketchDrivenPatternFeatureObject = 84051712, // 0x05028700
		/// <summary>Part Sketch Driven Pattern Features Collection Object.</summary>
		kSketchDrivenPatternFeaturesObject = 84051968, // 0x05028800
		/// <summary>Part Sketch Driven Pattern Feature Proxy Object.</summary>
		kSketchDrivenPatternFeatureProxyObject = 84052224, // 0x05028900
		/// <summary>MeshFeature Object.</summary>
		kMeshFeatureObject = 84052480, // 0x05028A00
		/// <summary>MeshFeatureProxy Object.</summary>
		kMeshFeatureProxyObject = 84052592, // 0x05028A70
		/// <summary>MeshFeatureSet Object.</summary>
		kMeshFeatureSetObject = 84052736, // 0x05028B00
		/// <summary>MeshFeatureSetProxy Object.</summary>
		kMeshFeatureSetProxyObject = 84052848, // 0x05028B70
		/// <summary>MeshFeatureSets Object.</summary>
		kMeshFeatureSetsObject = 84052992, // 0x05028C00
		/// <summary>MeshFeatureEntity Object.</summary>
		kMeshFeatureEntityObject = 84053248, // 0x05028D00
		/// <summary>MeshFeatureEntityProxy Object.</summary>
		kMeshFeatureEntityProxyObject = 84053360, // 0x05028D70
		/// <summary>MeshFeatureEntitiesEnumerator Object.</summary>
		kMeshFeatureEntitiesEnumeratorObject = 84053504, // 0x05028E00
		/// <summary>ModelCompositeAnnotations Object.</summary>
		kModelCompositeAnnotationsObject = 84053760, // 0x05028F00
		/// <summary>ModelCompositeAnnotation Object.</summary>
		kModelCompositeAnnotationObject = 84054016, // 0x05029000
		/// <summary>ModelCompositeAnnotationProxy Object.</summary>
		kModelCompositeAnnotationProxyObject = 84054128, // 0x05029070
		/// <summary>ModelCompositeAnnotationDefinition Object.</summary>
		kModelCompositeAnnotationDefinitionObject = 84054272, // 0x05029100
		/// <summary>MeshFace Object.</summary>
		kMeshFaceObject = 84055040, // 0x05029400
		/// <summary>MeshFaceProxy Object.</summary>
		kMeshFaceProxyObject = 84055152, // 0x05029470
		/// <summary>MeshEdge Object.</summary>
		kMeshEdgeObject = 84055296, // 0x05029500
		/// <summary>MeshEdgeProxy Object.</summary>
		kMeshEdgeProxyObject = 84055408, // 0x05029570
		/// <summary>MeshVertex Object.</summary>
		kMeshVertexObject = 84055552, // 0x05029600
		/// <summary>MeshVertexProxy Object.</summary>
		kMeshVertexProxyObject = 84055664, // 0x05029670
		/// <summary>EqualConstraint3D Object.</summary>
		kEqualConstraint3DObject = 84055808, // 0x05029700
		/// <summary>EqualConstraint3DProxy Object.</summary>
		kEqualConstraint3DProxyObject = 84055920, // 0x05029770
		/// <summary>ModelToleranceFeatureDefinition Object.</summary>
		kModelToleranceFeatureDefinitionObject = 84056064, // 0x05029800
		/// <summary>ModelToleranceFeature Object.</summary>
		kModelToleranceFeatureObject = 84056320, // 0x05029900
		/// <summary>ModelToleranceFeatureProxy Object.</summary>
		kModelToleranceFeatureProxyObject = 84056432, // 0x05029970
		/// <summary>ModelToleranceFeatures Object.</summary>
		kModelToleranceFeaturesObject = 84056576, // 0x05029A00
		/// <summary>ModelToleranceFeaturesEnumerator Object.</summary>
		kModelToleranceFeaturesEnumeratorObject = 84056832, // 0x05029B00
		/// <summary>The ImportedDWGLayer object.</summary>
		kImportedDWGLayerObject = 84057088, // 0x05029C00
		/// <summary>The ImportedDWGLayersEnumerator collection object.</summary>
		kImportedDWGLayersEnumeratorObject = 84057344, // 0x05029D00
		/// <summary>The DWGACMStandardPart object represents a DWG ACM standard part within a DWGModelSpaceDefinition.</summary>
		kDWGACMStandardPartObject = 84057600, // 0x05029E00
		/// <summary>The DWGACMStandardPartProxy object.</summary>
		kDWGACMStandardPartProxyObject = 84057856, // 0x05029F00
		/// <summary>The DWGACMStandardPartsEnumerator object provides access to all the DWG ACMStandardPart objects in a DWGModelSpaceDefinition.</summary>
		kDWGACMStandardPartsEnumeratorObject = 84058112, // 0x0502A000
		/// <summary>Part Sketch Driven Pattern Feature Definition object.</summary>
		kSketchDrivenPatternDefinitionObject = 84058368, // 0x0502A100
		/// <summary>Part RectangularPatternFeature Definition Object.</summary>
		kRectangularPatternFeatureDefinitionObject = 84058624, // 0x0502A200
		/// <summary>Part CircularPatternFeature Definition Object.</summary>
		kCircularPatternFeatureDefinitionObject = 84058880, // 0x0502A300
		/// <summary>Part MirrorFeature Definition Object.</summary>
		kMirrorFeatureDefinitionObject = 84059136, // 0x0502A400
		/// <summary>RuledSurfaceEdgeFacePairs object.</summary>
		kRuledSurfaceEdgeFacePairsObject = 84059392, // 0x0502A500
		/// <summary>RuledSurfaceEdgeFacePair object.</summary>
		kRuledSurfaceEdgeFacePairObject = 84059648, // 0x0502A600
		/// <summary>ParallelToXAxisConstraint3D Object.</summary>
		kParallelToXAxisConstraint3DObject = 84059904, // 0x0502A700
		/// <summary>ParallelToXAxisConstraint3DProxy Object.</summary>
		kParallelToXAxisConstraint3DProxyObject = 84060016, // 0x0502A770
		/// <summary>ParallelToYAxisConstraint3D Object.</summary>
		kParallelToYAxisConstraint3DObject = 84060160, // 0x0502A800
		/// <summary>ParallelToYAxisConstraint3DProxy Object.</summary>
		kParallelToYAxisConstraint3DProxyObject = 84060272, // 0x0502A870
		/// <summary>ParallelToZAxisConstraint3D Object.</summary>
		kParallelToZAxisConstraint3DObject = 84060416, // 0x0502A900
		/// <summary>ParallelToZAxisConstraint3DProxy Object.</summary>
		kParallelToZAxisConstraint3DProxyObject = 84060528, // 0x0502A970
		/// <summary>ParallelToXYPlaneConstraint3D Object.</summary>
		kParallelToXYPlaneConstraint3DObject = 84060672, // 0x0502AA00
		/// <summary>ParallelToXYPlaneConstraint3DProxy Object.</summary>
		kParallelToXYPlaneConstraint3DProxyObject = 84060784, // 0x0502AA70
		/// <summary>ParallelToYZPlaneConstraint3D Object.</summary>
		kParallelToYZPlaneConstraint3DObject = 84060928, // 0x0502AB00
		/// <summary>ParallelToYZPlaneConstraint3DProxy Object.</summary>
		kParallelToYZPlaneConstraint3DProxyObject = 84061040, // 0x0502AB70
		/// <summary>ParallelToXZPlaneConstraint3D Object.</summary>
		kParallelToXZPlaneConstraint3DObject = 84061184, // 0x0502AC00
		/// <summary>ParallelToXZPlaneConstraint3DProxy Object.</summary>
		kParallelToXZPlaneConstraint3DProxyObject = 84061296, // 0x0502AC70
		/// <summary>OnFaceConstraint3D Object.</summary>
		kOnFaceConstraint3DObject = 84061440, // 0x0502AD00
		/// <summary>OnFaceConstraint3DProxy Object.</summary>
		kOnFaceConstraint3DProxyObject = 84061552, // 0x0502AD70
		/// <summary>ModelDatumReferenceFrameDefinition Object.</summary>
		kModelDatumReferenceFrameDefinitionObject = 84061696, // 0x0502AE00
		/// <summary>ModelDatumReferenceFrame Object.</summary>
		kModelDatumReferenceFrameObject = 84061952, // 0x0502AF00
		/// <summary>ModelDatumReferenceFrameProxy Object.</summary>
		kModelDatumReferenceFrameProxyObject = 84062064, // 0x0502AF70
		/// <summary>ModelDatumReferenceFrames Object.</summary>
		kModelDatumReferenceFramesObject = 84062208, // 0x0502B000
		/// <summary>ModelGeneralNote Object.</summary>
		kModelGeneralNoteObject = 84062464, // 0x0502B100
		/// <summary>ModelGeneralNoteProxy Object.</summary>
		kModelGeneralNoteProxyObject = 84062576, // 0x0502B170
		/// <summary>ModelGeneralNoteDefinition Object.</summary>
		kModelGeneralNoteDefinitionObject = 84062720, // 0x0502B200
		/// <summary>ModelGeneralNotes Object.</summary>
		kModelGeneralNotesObject = 84062720, // 0x0502B200
		/// <summary>GeneralNoteSymbolDefinition Object.</summary>
		kGeneralNoteSymbolDefinitionObject = 84063232, // 0x0502B400
		/// <summary>GeneralNoteSymbolDefinitions Object.</summary>
		kGeneralNoteSymbolDefinitionsObject = 84063488, // 0x0502B500
		/// <summary>Derived Future Part Components Collection.</summary>
		kDerivedFuturePartComponentObject = 84063744, // 0x0502B600
		/// <summary>DerivedFuturePartComponentProxy object.</summary>
		kDerivedFuturePartComponentProxyObject = 84063856, // 0x0502B670
		/// <summary>Derived Future Part Components Collection.</summary>
		kDerivedFuturePartComponentsObject = 84064000, // 0x0502B700
		/// <summary>Derived Future Part Definition Object.</summary>
		kDerivedFuturePartDefinitionObject = 84064256, // 0x0502B800
		/// <summary>Derived Future Assembly Components Collection.</summary>
		kDerivedFutureAssemblyComponentObject = 84064512, // 0x0502B900
		/// <summary>DerivedFutureAssemblyComponentProxy object.</summary>
		kDerivedFutureAssemblyComponentProxyObject = 84064624, // 0x0502B970
		/// <summary>Derived Future Assembly Components Collection.</summary>
		kDerivedFutureAssemblyComponentsObject = 84064768, // 0x0502BA00
		/// <summary>Derived Future Assembly Definition Object.</summary>
		kDerivedFutureAssemblyDefinitionObject = 84065024, // 0x0502BB00
		/// <summary>Derived Future Assembly Occurrences Object.</summary>
		kDerivedFutureAssemblyOccurrencesObject = 84065280, // 0x0502BC00
		/// <summary>Derived Future Assembly Occurrence Object.</summary>
		kDerivedFutureAssemblyOccurrenceObject = 84065536, // 0x0502BD00
		/// <summary>The DWGAcadSupportedProxy object represents a DWG ACAD support proxy within a DWGModelSpaceDefinition.</summary>
		kDWGAcadSupportedProxyObject = 84065792, // 0x0502BE00
		/// <summary>The DWGAcadSupportedProxyProxy object.</summary>
		kDWGAcadSupportedProxyProxyObject = 84066048, // 0x0502BF00
		/// <summary>The DWGAcadSupportedProxiesEnumerator object provides access to all the DWG DWGAcadSupportedProxy objects in a DWGModelSpaceDefinition.</summary>
		kDWGAcadSupportedProxiesEnumeratorObject = 84068864, // 0x0502CA00
		/// <summary>DWGEntitySegmentsEnumerator Collection object.</summary>
		kDWGEntitySegmentsEnumeratorObject = 84069120, // 0x0502CB00
		/// <summary>DWGEntitySegment object.</summary>
		kDWGEntitySegmentObject = 84069376, // 0x0502CC00
		/// <summary>DWGEntitySegmentProxy object.</summary>
		kDWGEntitySegmentProxyObject = 84069632, // 0x0502CD00
		/// <summary>DWGEntityLineSegment object.</summary>
		kDWGEntityLineSegmentObject = 84069888, // 0x0502CE00
		/// <summary>DWGEntityLineSegmentProxy object.</summary>
		kDWGEntityLineSegmentProxyObject = 84070144, // 0x0502CF00
		/// <summary>DWGEntityArcSegment object.</summary>
		kDWGEntityArcSegmentObject = 84070400, // 0x0502D000
		/// <summary>DWGEntityArcSegmentProxy object.</summary>
		kDWGEntityArcSegmentProxyObject = 84070656, // 0x0502D100
		/// <summary>DWGEntityEllipticalArcSegment object.</summary>
		kDWGEntityEllipticalArcSegmentObject = 84070912, // 0x0502D200
		/// <summary>DWGEntityEllipticalArcSegmentProxy object.</summary>
		kDWGEntityEllipticalArcSegmentProxyObject = 84071168, // 0x0502D300
		/// <summary>DWGEntitySplineSegment object.</summary>
		kDWGEntitySplineSegmentObject = 84071424, // 0x0502D400
		/// <summary>DWGEntitySplineSegmentProxy object.</summary>
		kDWGEntitySplineSegmentProxyObject = 84071680, // 0x0502D500
		/// <summary>HoleClearanceInfo object.</summary>
		kHoleClearanceInfoObject = 84071936, // 0x0502D600
		/// <summary>Shrinkwrap Component Object.</summary>
		kShrinkwrapComponentObject = 84072192, // 0x0502D700
		/// <summary>Shrinkwrap Component Proxy Object.</summary>
		kShrinkwrapComponentProxyObject = 84072194, // 0x0502D702
		/// <summary>Shrinkwrap Components Collection Object.</summary>
		kShrinkwrapComponentsObject = 84072448, // 0x0502D800
		/// <summary>Shrinkwrap Definition Object.</summary>
		kShrinkwrapDefinitionObject = 84072704, // 0x0502D900
		/// <summary>DistancetFromFaceExtent object.</summary>
		kDistanceFromFaceExtentObject = 84072960, // 0x0502DA00
		/// <summary>Partial Chamfer edge object.</summary>
		kPartialChamferEdgeObject = 84073216, // 0x0502DB00
		/// <summary>Partial Chamfer edges object.</summary>
		kPartialChamferEdgesObject = 84073472, // 0x0502DC00
		/// <summary>SplineLengthDimConstraint3D Object.</summary>
		kSplineLengthDimConstraint3DObject = 84073728, // 0x0502DD00
		/// <summary>SplineLengthDimConstraint3DProxy Object.</summary>
		kSplineLengthDimConstraint3DProxyObject = 84073840, // 0x0502DD70
		/// <summary>Helical Curve Object.</summary>
		kHelicalCurveObject = 84073984, // 0x0502DE00
		/// <summary>Helical Curve Proxy Object.</summary>
		kHelicalCurveProxyObject = 84074096, // 0x0502DE70
		/// <summary>Helical Curve Definition Object.</summary>
		kHelicalCurveDefinitionObject = 84074240, // 0x0502DF00
		/// <summary>Helical Curve Constant Shape Definition Object.</summary>
		kHelicalCurveConstantShapeDefinitionObject = 84074496, // 0x0502E000
		/// <summary>Helical Curve Variable Shape Definition Object.</summary>
		kHelicalCurveVariableShapeDefinitionObject = 84074752, // 0x0502E100
		/// <summary>Helical Curve Shape Definition Rows Object.</summary>
		kHelicalCurveShapeDefinitionRowsObject = 84075008, // 0x0502E200
		/// <summary>Helical Curve Shape Definition Row Object.</summary>
		kHelicalCurveShapeDefinitionRowObject = 84075264, // 0x0502E300
		/// <summary>HelicalCurves Object.</summary>
		kHelicalCurvesObject = 84075520, // 0x0502E400
		/// <summary>SolidSweepDefinition Object.</summary>
		kSolidSweepDefinitionObject = 84075776, // 0x0502E500
		/// <summary>Part Unwrap Features Collection Object.</summary>
		kUnwrapFeaturesObject = 84076032, // 0x0502E600
		/// <summary>Part Unwrap Feature Object.</summary>
		kUnwrapFeatureObject = 84076288, // 0x0502E700
		/// <summary>Part Unwrap Feature Proxy Object.</summary>
		kUnwrapFeatureProxyObject = 84076544, // 0x0502E800
		/// <summary>Part Unwrap Definition Object.</summary>
		kUnwrapDefinitionObject = 84076800, // 0x0502E900
		/// <summary>ProjectToSurfaceCurves Object.</summary>
		kProjectToSurfaceCurvesObject = 84077056, // 0x0502EA00
		/// <summary>ProjectToSurfaceCurve Object.</summary>
		kProjectToSurfaceCurveObject = 84077312, // 0x0502EB00
		/// <summary>ProjectToSurfaceCurveProxy Object.</summary>
		kProjectToSurfaceCurveProxyObject = 84077424, // 0x0502EB70
		/// <summary>SketchHatchRegion Object.</summary>
		kSketchHatchRegionObject = 84077568, // 0x0502EC00
		/// <summary>SketchHatchRegions Object.</summary>
		kSketchHatchRegionsObject = 84077824, // 0x0502ED00
		/// <summary>ImportedRVTComponent Object.</summary>
		kImportedRVTComponentObject = 84078080, // 0x0502EE00
		/// <summary>ImportedRVTComponentProxy Object.</summary>
		kImportedRVTComponentProxyObject = 84078336, // 0x0502EF00
		/// <summary>ImportedRVTComponentDefinition Object.</summary>
		kImportedRVTComponentDefinitionObject = 84078592, // 0x0502F000
		/// <summary>ModelDatumTargets Object.</summary>
		kModelDatumTargetsObject = 84078848, // 0x0502F100
		/// <summary>ModelDatumTarget Object.</summary>
		kModelDatumTargetObject = 84079104, // 0x0502F200
		/// <summary>ModelDatumTargetProxy Object.</summary>
		kModelDatumTargetProxyObject = 84079216, // 0x0502F270
		/// <summary>ModelDatumDefinition Object.</summary>
		kModelDatumDefinitionObject = 84079360, // 0x0502F300
		/// <summary>ModelDatumFeaturesEnumerator Object.</summary>
		kModelDatumFeaturesEnumeratorObject = 84079616, // 0x0502F400
		/// <summary>ModelDatumFeature Object.</summary>
		kModelDatumFeatureObject = 84079872, // 0x0502F500
		/// <summary>ModelDatumFeatureProxy Object.</summary>
		kModelDatumFeatureProxyObject = 84079984, // 0x0502F570
		/// <summary>ModelDatums Object.</summary>
		kModelDatumsObject = 84080128, // 0x0502F600
		/// <summary>ModelDatum Object.</summary>
		kModelDatumObject = 84080384, // 0x0502F700
		/// <summary>ModelDatumProxy Object.</summary>
		kModelDatumProxyObject = 84080496, // 0x0502F770
		/// <summary>Part Mark Feature Object.</summary>
		kMarkFeatureObject = 84080640, // 0x0502F800
		/// <summary>Part Mark Feature Proxy Object.</summary>
		kMarkFeatureProxyObject = 84080896, // 0x0502F900
		/// <summary>Part Mark Features Object.</summary>
		kMarkFeaturesObject = 84081152, // 0x0502FA00
		/// <summary>ImportedDataExchangeComponent Object.</summary>
		kImportedDataExchangeComponentObject = 84081408, // 0x0502FB00
		/// <summary>ImportedDataExchangeComponentProxy Object.</summary>
		kImportedDataExchangeComponentProxyObject = 84081664, // 0x0502FC00
		/// <summary>ImportedDataExchangeComponentDefinition Object.</summary>
		kImportedDataExchangeComponentDefinitionObject = 84081920, // 0x0502FD00
		/// <summary>MarkStylesEnumerator Object.</summary>
		kMarkStylesEnumeratorObject = 84082176, // 0x0502FE00
		/// <summary>MarkStyle Object.</summary>
		kMarkStyleObject = 84082432, // 0x0502FF00
		/// <summary>MarkGeometrySet Object.</summary>
		kMarkGeometrySetObject = 84082688, // 0x05030000
		/// <summary>MarkDefinition Object.</summary>
		kMarkDefinitionObject = 84082944, // 0x05030100
		/// <summary>ModelWeldingSymbol Object.</summary>
		kModelWeldingSymbolObject = 84083200, // 0x05030200
		/// <summary>ModelWeldingSymbolProxy Object.</summary>
		kModelWeldingSymbolProxyObject = 84083327, // 0x0503027F
		/// <summary>ModelWeldingSymbolDefinitions Object.</summary>
		kModelWeldingSymbolDefinitionsObject = 84083456, // 0x05030300
		/// <summary>ModelWeldingSymbols Object.</summary>
		kModelWeldingSymbolsObject = 84083712, // 0x05030400
		/// <summary>ModelWeldingSymbolDefinition Object.</summary>
		kModelWeldingSymbolDefinitionObject = 84083968, // 0x05030500
		/// <summary>Finish Feature Object.</summary>
		kFinishFeatureObject = 84084224, // 0x05030600
		/// <summary>Finish Features Object.</summary>
		kFinishFeaturesObject = 84084480, // 0x05030700
		/// <summary>Finish Feature Definition Object.</summary>
		kFinishDefinitionObject = 84084736, // 0x05030800
		/// <summary>Finish Feature Proxy Object.</summary>
		kFinishFeatureProxyObject = 84084992, // 0x05030900
		/// <summary>Replace Face Definition Object.</summary>
		kReplaceFaceDefinitionObject = 84085248, // 0x05030A00
		/// <summary>Part Pattern Boundary Setting Object.</summary>
		kPatternBoundarySettingObject = 84085504, // 0x05030B00
		/// <summary>Assembly Component Definition Object.</summary>
		kAssemblyComponentDefinitionObject = 100663808, // 0x06000200
		/// <summary>Assembly Constraints collection Object.</summary>
		kAssemblyConstraintsObject = 100664320, // 0x06000400
		/// <summary>Assembly Component Definition collection Object.</summary>
		kAssemblyComponentDefinitionsObject = 100664576, // 0x06000500
		/// <summary>Assembly Constraint Enumerator Object.</summary>
		kAssemblyConstraintsEnumeratorObject = 100664832, // 0x06000600
		/// <summary>Angle Assembly Constraint Object.</summary>
		kAngleConstraintObject = 100665088, // 0x06000700
		/// <summary>Insert Assembly Constraint Object.</summary>
		kInsertConstraintObject = 100665344, // 0x06000800
		/// <summary>Tangent Assembly Constraint Object.</summary>
		kTangentConstraintObject = 100665600, // 0x06000900
		/// <summary>Mate Assembly Constraint Object.</summary>
		kMateConstraintObject = 100665856, // 0x06000A00
		/// <summary>Transitional Assembly Constraint Object.</summary>
		kTransitionalConstraintObject = 100666112, // 0x06000B00
		/// <summary>Flush Assembly Constraint Object.</summary>
		kFlushConstraintObject = 100666368, // 0x06000C00
		/// <summary>Rotate-Rotate Assembly motion Constraint Object.</summary>
		kRotateRotateConstraintObject = 100666624, // 0x06000D00
		/// <summary>Rotate-Translate Assembly motion Constraint Object.</summary>
		kRotateTranslateConstraintObject = 100666880, // 0x06000E00
		/// <summary>Translate-Translate Assembly motion Constraint Object.</summary>
		kTranslateTranslateConstraintObject = 100667136, // 0x06000F00
		/// <summary>Assembly Events Object.</summary>
		kAssemblyEventsObject = 100667392, // 0x06001000
		/// <summary>Interference Results Collection Object.</summary>
		kInterferenceResultsObject = 100668160, // 0x06001300
		/// <summary>Interference Result Object.</summary>
		kInterferenceResultObject = 100668416, // 0x06001400
		/// <summary>Gets the Occurrence Patterns collection object that encapsulates all of the Occurrence Patterns defined in this Component Definition.</summary>
		kOccurrencePatternsObject = 100668672, // 0x06001500
		/// <summary>OccurrencePattern Object (abstract base class).</summary>
		kOccurrencePatternObject = 100668928, // 0x06001600
		/// <summary>Feature based occurrence pattern.</summary>
		kFeatureBasedOccurrencePatternObject = 100669184, // 0x06001700
		/// <summary>Feature based occurrence pattern proxy.</summary>
		kFeatureBasedOccurrencePatternProxyObject = 100669312, // 0x06001780
		/// <summary>Rectangular pattern of assembly occurrences.</summary>
		kRectangularOccurrencePatternObject = 100669440, // 0x06001800
		/// <summary>Rectangular pattern of assembly occurrences proxy.</summary>
		kRectangularOccurrencePatternProxyObject = 100669568, // 0x06001880
		/// <summary>Circular pattern of assembly occurrences.</summary>
		kCircularOccurrencePatternObject = 100669696, // 0x06001900
		/// <summary>Circular pattern of assembly occurrences Proxy.</summary>
		kCircularOccurrencePatternProxyObject = 100669824, // 0x06001980
		/// <summary>Elements that result from the creation of a pattern.</summary>
		kOccurrencePatternElementsObject = 100669952, // 0x06001A00
		/// <summary>WeldsComponentDefinition Object.</summary>
		kWeldsComponentDefinitionObject = 100670208, // 0x06001B00
		/// <summary>Object that allows you to get and set the information that specifies an assembly work plane.</summary>
		kAssemblyWorkPlaneDefObject = 100670464, // 0x06001C00
		/// <summary>Object that allows you to get and set the information that specifies an assembly work Axis.</summary>
		kAssemblyWorkAxisDefObject = 100670720, // 0x06001D00
		/// <summary>Object that allows you to get and set the information that specifies an assembly work Point.</summary>
		kAssemblyWorkPointDefObject = 100670976, // 0x06001E00
		/// <summary>Assembly Solver Object.</summary>
		kAssemblySolverObject = 100671232, // 0x06001F00
		/// <summary>Assembly Solver Node Object.</summary>
		kAssemblySolverNodeObject = 100671488, // 0x06002000
		/// <summary>Welds Object.</summary>
		kWeldsObject = 100671744, // 0x06002100
		/// <summary>WeldBeads Collection Object.</summary>
		kWeldBeadsObject = 100672000, // 0x06002200
		/// <summary>CosmeticWelds Collection Object.</summary>
		kCosmeticWeldsObject = 100672256, // 0x06002300
		/// <summary>Weld Object.</summary>
		kWeldObject = 100672512, // 0x06002400
		/// <summary>WeldBead Object.</summary>
		kWeldBeadObject = 100672768, // 0x06002500
		/// <summary>CosmeticWeld Object.</summary>
		kCosmeticWeldObject = 100673024, // 0x06002600
		/// <summary>WeldmentComponentDefinition Object.</summary>
		kWeldmentComponentDefinitionObject = 100673280, // 0x06002700
		/// <summary>Custom Assembly Constraint Object.</summary>
		kCustomConstraintObject = 100673536, // 0x06002800
		/// <summary>BOM Object.</summary>
		kBOMObject = 100673792, // 0x06002900
		/// <summary>BOMViews Object.</summary>
		kBOMViewsObject = 100674048, // 0x06002A00
		/// <summary>BOMView Object.</summary>
		kBOMViewObject = 100674304, // 0x06002B00
		/// <summary>BOMRow Object.</summary>
		kBOMRowObject = 100674816, // 0x06002D00
		/// <summary>VirtualComponentDefinition Object.</summary>
		kVirtualComponentDefinitionObject = 100675072, // 0x06002E00
		/// <summary>Features Object.</summary>
		kFeaturesObject = 100675328, // 0x06002F00
		/// <summary>Angle Assembly Constraint Proxy Object.</summary>
		kAngleConstraintProxyObject = 100675584, // 0x06003000
		/// <summary>Flush Assembly Constraint Proxy Object.</summary>
		kFlushConstraintProxyObject = 100675840, // 0x06003100
		/// <summary>Flush Assembly Constraint Proxy Object.</summary>
		kInsertConstraintProxyObject = 100676096, // 0x06003200
		/// <summary>Mate Assembly Constraint Proxy Object.</summary>
		kMateConstraintProxyObject = 100676352, // 0x06003300
		/// <summary>Rotate-Rotate Assembly motion Constraint Proxy Object.</summary>
		kRotateRotateConstraintProxyObject = 100676608, // 0x06003400
		/// <summary>Rotate-Translate Assembly motion Constraint Proxy Object.</summary>
		kRotateTranslateConstraintProxyObject = 100676864, // 0x06003500
		/// <summary>Tangent Assembly Constraint Proxy Object.</summary>
		kTangentConstraintProxyObject = 100677120, // 0x06003600
		/// <summary>Transitional Assembly Constraint Proxy Object.</summary>
		kTransitionalConstraintProxyObject = 100677376, // 0x06003700
		/// <summary>Custom Assembly Constraint Proxy Object.</summary>
		kCustomConstraintProxyObject = 100677632, // 0x06003800
		/// <summary>Translate-Translate Assembly motion Constraint Proxy Object.</summary>
		kTranslateTranslateConstraintProxyObject = 100677888, // 0x06003900
		/// <summary>Composite Assembly Constraint Object.</summary>
		kCompositeConstraintObject = 100678144, // 0x06003A00
		/// <summary>RepresentationsManager Object.</summary>
		kRepresentationsManagerObject = 100678400, // 0x06003B00
		/// <summary>PositionalRepresentations Object.</summary>
		kPositionalRepresentationsObject = 100678656, // 0x06003C00
		/// <summary>PositionalRepresentation Object.</summary>
		kPositionalRepresentationObject = 100678912, // 0x06003D00
		/// <summary>DesignViewRepresentations Object.</summary>
		kDesignViewRepresentationsObject = 100679168, // 0x06003E00
		/// <summary>DesignViewRepresentation Object.</summary>
		kDesignViewRepresentationObject = 100679424, // 0x06003F00
		/// <summary>LevelOfDetailRepresentations Object.</summary>
		kLevelOfDetailRepresentationsObject = 100679680, // 0x06004000
		/// <summary>LevelOfDetailRepresentation Object.</summary>
		kLevelOfDetailRepresentationObject = 100679936, // 0x06004100
		/// <summary>iAssemblyMember Object.</summary>
		kiAssemblyMemberObject = 100680192, // 0x06004200
		/// <summary>iAssemblyTableRows Object.</summary>
		kiAssemblyTableRowsObject = 100680448, // 0x06004300
		/// <summary>iAssemblyTableRow Object.</summary>
		kiAssemblyTableRowObject = 100680704, // 0x06004400
		/// <summary>iAssemblyFactory Object.</summary>
		kiAssemblyFactoryObject = 100680960, // 0x06004500
		/// <summary>iAssemblyTableCell Object.</summary>
		kiAssemblyTableCellObject = 100681216, // 0x06004600
		/// <summary>iAssemblyTableColumns Object.</summary>
		kiAssemblyTableColumnsObject = 100681472, // 0x06004700
		/// <summary>iAssemblyTableColumn Object.</summary>
		kiAssemblyTableColumnObject = 100681728, // 0x06004800
		/// <summary>BOMRowsEnumerator Object.</summary>
		kBOMRowsEnumeratorObject = 100681984, // 0x06004900
		/// <summary>RigidBodyResults Object.</summary>
		kRigidBodyResultsObject = 100682240, // 0x06004A00
		/// <summary>RigidBodyGroups Object.</summary>
		kRigidBodyGroupsObject = 100686336, // 0x06005A00
		/// <summary>RigidBodyGroup Object.</summary>
		kRigidBodyGroupObject = 100690432, // 0x06006A00
		/// <summary>RigidBodyJoints Object.</summary>
		kRigidBodyJointsObject = 100694528, // 0x06007A00
		/// <summary>RigidBodyJoint Object.</summary>
		kRigidBodyJointObject = 100698624, // 0x06008A00
		/// <summary>Machining Object.</summary>
		kMachiningObject = 100698880, // 0x06008B00
		/// <summary>Preparations Object.</summary>
		kPreparationsObject = 100699136, // 0x06008C00
		/// <summary>Layout Constraint Object.</summary>
		kLayoutConstraintObject = 100699392, // 0x06008D00
		/// <summary>Layout Constraint Proxy Object.</summary>
		kLayoutConstraintProxyObject = 100699648, // 0x06008E00
		/// <summary>ConstraintLimits Object.</summary>
		kConstraintLimitsObject = 100699904, // 0x06008F00
		/// <summary>DriveConstraintSettings Object.</summary>
		kDriveConstraintSettingsObject = 100700160, // 0x06009000
		/// <summary>DynamicSimulation object.</summary>
		kDynamicSimulationObject = 100700416, // 0x06009100
		/// <summary>SimulationComponent Object.</summary>
		kSimulationComponentObject = 100700416, // 0x06009100
		/// <summary>DSJoints object.</summary>
		kDSJointsObject = 100700928, // 0x06009300
		/// <summary>DSJoint object.</summary>
		kDSJointObject = 100701184, // 0x06009400
		/// <summary>SimulationManager object.</summary>
		kSimulationManagerObject = 100701440, // 0x06009500
		/// <summary>DynamicSimulations object.</summary>
		kDynamicSimulationsObject = 100701696, // 0x06009600
		/// <summary>DSLoads object.</summary>
		kDSLoadsObject = 100701952, // 0x06009700
		/// <summary>DSLoad object.</summary>
		kDSLoadObject = 100702208, // 0x06009800
		/// <summary>DSLoadDefinition object.</summary>
		kDSLoadDefinitionObject = 100702464, // 0x06009900
		/// <summary>DSValue object.</summary>
		kDSValueObject = 100702720, // 0x06009A00
		/// <summary>DSValueGraph object.</summary>
		kDSValueGraphObject = 100702976, // 0x06009B00
		/// <summary>DSJointDefinition object.</summary>
		kDSJointDefinitionObject = 100703232, // 0x06009C00
		/// <summary>DSDegreesOfFreedom object.</summary>
		kDSDegreesOfFreedomObject = 100703488, // 0x06009D00
		/// <summary>DSDegreeOfFreedom object.</summary>
		kDSDegreeOfFreedomObject = 100703744, // 0x06009E00
		/// <summary>DSResults object.</summary>
		kDSResultsObject = 100704000, // 0x06009F00
		/// <summary>DSResult object.</summary>
		kDSResultObject = 100704256, // 0x0600A000
		/// <summary>DSSettings object.</summary>
		kDSSettingsObject = 100704512, // 0x0600A100
		/// <summary>VisibleOccurrenceFinder object.</summary>
		kVisibleOccurrenceFinderObject = 100704768, // 0x0600A200
		/// <summary>OGSSceneNode object.</summary>
		kOGSSceneNodeObject = 100705024, // 0x0600A300
		/// <summary>OGSRenderItemsEnumerator object.</summary>
		kOGSSceneNodesEnumeratorObject = 100705280, // 0x0600A400
		/// <summary>OGSRenderItem object.</summary>
		kOGSRenderItemObject = 100705536, // 0x0600A500
		/// <summary>OGSRenderItemsEnumerator object.</summary>
		kOGSRenderItemsEnumeratorObject = 100705792, // 0x0600A600
		/// <summary>AssemblyJoints Object.</summary>
		kAssemblyJointsObject = 100706048, // 0x0600A700
		/// <summary>AssemblyJointDefinition Object.</summary>
		kAssemblyJointDefinitionObject = 100706304, // 0x0600A800
		/// <summary>AssemblyJoint Object.</summary>
		kAssemblyJointObject = 100706560, // 0x0600A900
		/// <summary>DriveSettings Object.</summary>
		kDriveSettingsObject = 100706816, // 0x0600AA00
		/// <summary>AssemblyJointProxy Object.</summary>
		kAssemblyJointProxyObject = 100707072, // 0x0600AB00
		/// <summary>JointOriginFace Object.</summary>
		kJointOriginFaceObject = 100707328, // 0x0600AC00
		/// <summary>JointOriginEdge Object.</summary>
		kJointOriginEdgeObject = 100707329, // 0x0600AC01
		/// <summary>JointOriginSketch Object.</summary>
		kJointOriginSketchObject = 100707330, // 0x0600AC02
		/// <summary>JointOriginDWG Object.</summary>
		kJointOriginDWGObject = 100707331, // 0x0600AC03
		/// <summary>Assembly Joint Enumerator Object.</summary>
		kAssemblyJointsEnumeratorObject = 100707584, // 0x0600AD00
		/// <summary>Symmetry Assembly Constraint Object.</summary>
		kAssemblySymmetryConstraintObject = 100707840, // 0x0600AE00
		/// <summary>Symmetry Assembly Constraint Proxy Object.</summary>
		kAssemblySymmetryConstraintProxyObject = 100708096, // 0x0600AF00
		/// <summary>Cosmetic Weld Definition Object.</summary>
		kCosmeticWeldDefinitionObject = 100708352, // 0x0600B000
		/// <summary>ModelStates Object.</summary>
		kModelStatesObject = 100708608, // 0x0600B100
		/// <summary>ModelState Object.</summary>
		kModelStateObject = 100708864, // 0x0600B200
		/// <summary>ModelStateTable Object.</summary>
		kModelStateTableObject = 100709120, // 0x0600B300
		/// <summary>ModelStateTableRows Object.</summary>
		kModelStateTableRowsObject = 100709376, // 0x0600B400
		/// <summary>ModelStateTableRow Object.</summary>
		kModelStateTableRowObject = 100709632, // 0x0600B500
		/// <summary>ModelStateTableColumns Object.</summary>
		kModelStateTableColumnsObject = 100709888, // 0x0600B600
		/// <summary>ModelStateTableColumn Object.</summary>
		kModelStateTableColumnObject = 100710144, // 0x0600B700
		/// <summary>ModelStateTableCell Object.</summary>
		kModelStateTableCellObject = 100710400, // 0x0600B800
		/// <summary>RevitExports Object.</summary>
		kRevitExportsObject = 100710656, // 0x0600B900
		/// <summary>RevitExport Object.</summary>
		kRevitExportObject = 100710912, // 0x0600BA00
		/// <summary>RevitExportDefinition Object.</summary>
		kRevitExportDefinitionObject = 100711168, // 0x0600BB00
		/// <summary>Mirror pattern of assembly occurrences.</summary>
		kMirrorOccurrencePatternObject = 100711424, // 0x0600BC00
		/// <summary>Mirror pattern of assembly occurrences Proxy.</summary>
		kMirrorOccurrencePatternProxyObject = 100711552, // 0x0600BC80
		/// <summary>Apprentice Server Drawing Document Object.</summary>
		kApprenticeServerDrawingDocumentObject = 117441024, // 0x07000200
		/// <summary>Drawing Sheet Object.</summary>
		kSheetObject = 117441280, // 0x07000300
		/// <summary>Drawing View Object.</summary>
		kDrawingViewObject = 117441536, // 0x07000400
		/// <summary>Drafting Standard Object.</summary>
		kDraftingStandardObject = 117441792, // 0x07000500
		/// <summary>Drawing Views collection object.</summary>
		kDrawingViewsObject = 117442304, // 0x07000700
		/// <summary>Drawing Sheets collection object.</summary>
		kSheetsObject = 117442560, // 0x07000800
		/// <summary>Drawing Sketch Object.</summary>
		kDrawingSketchObject = 117443328, // 0x07000B00
		/// <summary>Drawing Sketches Collection Object.</summary>
		kDrawingSketchesObject = 117443584, // 0x07000C00
		/// <summary>PartsLists Collection Object.</summary>
		kPartsListsObject = 117443840, // 0x07000D00
		/// <summary>PartsList Object.</summary>
		kPartsListObject = 117444096, // 0x07000E00
		/// <summary>PartsListColumns Collection Object.</summary>
		kPartsListColumnsObject = 117444352, // 0x07000F00
		/// <summary>PartsListColumn Object.</summary>
		kPartsListColumnObject = 117444608, // 0x07001000
		/// <summary>PartsListRows Collection Object.</summary>
		kPartsListRowsObject = 117444864, // 0x07001100
		/// <summary>PartsListRow Object.</summary>
		kPartsListRowObject = 117445120, // 0x07001200
		/// <summary>PartsListCell Object.</summary>
		kPartsListCellObject = 117445376, // 0x07001300
		/// <summary>TextBoxes Collection Object.</summary>
		kTextBoxesObject = 117445632, // 0x07001400
		/// <summary>TextBox Object.</summary>
		kTextBoxObject = 117445888, // 0x07001500
		/// <summary>BorderDefinitions Collection Object.</summary>
		kBorderDefinitionsObject = 117446144, // 0x07001600
		/// <summary>BorderDefinition Object.</summary>
		kBorderDefinitionObject = 117446400, // 0x07001700
		/// <summary>Border Object.</summary>
		kBorderObject = 117446656, // 0x07001800
		/// <summary>TitleBlockDefinitions Collection Object.</summary>
		kTitleBlockDefinitionsObject = 117446912, // 0x07001900
		/// <summary>TitleBlockDefinition Object.</summary>
		kTitleBlockDefinitionObject = 117447168, // 0x07001A00
		/// <summary>TitleBlock Object.</summary>
		kTitleBlockObject = 117447424, // 0x07001B00
		/// <summary>TextStyles Collection Object.</summary>
		kTextStylesObject = 117447680, // 0x07001C00
		/// <summary>TextStyle Object.</summary>
		kTextStyleObject = 117447936, // 0x07001D00
		/// <summary>SketchedSymbolDefinitions Object.</summary>
		kSketchedSymbolDefinitionsObject = 117448192, // 0x07001E00
		/// <summary>SketchedSymbolDefinition Object.</summary>
		kSketchedSymbolDefinitionObject = 117448448, // 0x07001F00
		/// <summary>SketchedSymbol Object.</summary>
		kSketchedSymbolObject = 117448704, // 0x07002000
		/// <summary>SketchedSymbols Object.</summary>
		kSketchedSymbolsObject = 117449472, // 0x07002300
		/// <summary>DefaultBorder Object.</summary>
		kDefaultBorderObject = 117449728, // 0x07002400
		/// <summary>DrawingSettings Object.</summary>
		kDrawingSettingsObject = 117449984, // 0x07002500
		/// <summary>Wrapper for the drawing view events sink.</summary>
		kDrawingViewEventsObject = 117450496, // 0x07002700
		/// <summary>DimensionStyle Object.</summary>
		kDimensionStyleObject = 117452800, // 0x07003000
		/// <summary>DimensionStylesEnumerator Object.</summary>
		kDimensionStylesEnumeratorObject = 117453056, // 0x07003100
		/// <summary>DrawingStylesManager Object.</summary>
		kDrawingStylesManagerObject = 117453568, // 0x07003300
		/// <summary>DrawingStandardStylesEnumerator Object.</summary>
		kDrawingStandardStylesEnumeratorObject = 117453824, // 0x07003400
		/// <summary>DrawingStandardStyle Object.</summary>
		kDrawingStandardStyleObject = 117454080, // 0x07003500
		/// <summary>ObjectDefaultsStyle Object.</summary>
		kObjectDefaultsStyleObject = 117454336, // 0x07003600
		/// <summary>DrawingDimensions Object.</summary>
		kDrawingDimensionsObject = 117454592, // 0x07003700
		/// <summary>DrawingDimension Object.</summary>
		kDrawingDimensionObject = 117454848, // 0x07003800
		/// <summary>GeneralDimensions Object.</summary>
		kGeneralDimensionsObject = 117455104, // 0x07003900
		/// <summary>GeneralDimension Object.</summary>
		kGeneralDimensionObject = 117456896, // 0x07004000
		/// <summary>CustomTables Object.</summary>
		kCustomTablesObject = 117460992, // 0x07005000
		/// <summary>CustomTable Object.</summary>
		kCustomTableObject = 117461248, // 0x07005100
		/// <summary>Columns Object.</summary>
		kColumnsObject = 117461504, // 0x07005200
		/// <summary>Column Object.</summary>
		kColumnObject = 117461760, // 0x07005300
		/// <summary>Rows Object.</summary>
		kRowsObject = 117462016, // 0x07005400
		/// <summary>Row Object.</summary>
		kRowObject = 117462272, // 0x07005500
		/// <summary>Cell Object.</summary>
		kCellObject = 117462528, // 0x07005600
		/// <summary>Balloons Object.</summary>
		kBalloonsObject = 117462784, // 0x07005700
		/// <summary>Balloon Object.</summary>
		kBalloonObject = 117463040, // 0x07005800
		/// <summary>Drawing Section View Object.</summary>
		kSectionDrawingViewObject = 117463296, // 0x07005900
		/// <summary>BalloonValueSets Object.</summary>
		kBalloonValueSetsObject = 117463552, // 0x07005A00
		/// <summary>BalloonValueSet Object.</summary>
		kBalloonValueSetObject = 117463808, // 0x07005B00
		/// <summary>ObjectDefaultsStylesEnumerator Object.</summary>
		kObjectDefaultsStylesEnumeratorObject = 117464064, // 0x07005C00
		/// <summary>DimensionText Object.</summary>
		kDimensionTextObject = 117464320, // 0x07005D00
		/// <summary>GeneralDimensions enumerator Object.</summary>
		kGeneralDimensionsEnumeratorObject = 117464576, // 0x07005E00
		/// <summary>Wrapper for the drawing events object.</summary>
		kDrawingEventsObject = 117464832, // 0x07005F00
		/// <summary>TableFormat Object.</summary>
		kTableFormatObject = 117465088, // 0x07006000
		/// <summary>TextStylesEnumerator Collection Object.</summary>
		kTextStylesEnumeratorObject = 117465600, // 0x07006200
		/// <summary>LayersEnumerator Collection Object.</summary>
		kLayersEnumeratorObject = 117465856, // 0x07006300
		/// <summary>Layer Object.</summary>
		kLayerObject = 117466112, // 0x07006400
		/// <summary>RevisionTables Collection Object.</summary>
		kRevisionTablesObject = 117466368, // 0x07006500
		/// <summary>RevisionTable Object.</summary>
		kRevisionTableObject = 117466624, // 0x07006600
		/// <summary>RevisionTableColumns Collection Object.</summary>
		kRevisionTableColumnsObject = 117466880, // 0x07006700
		/// <summary>RevisionTableColumn Object.</summary>
		kRevisionTableColumnObject = 117467136, // 0x07006800
		/// <summary>RevisionTableRows Collection Object.</summary>
		kRevisionTableRowsObject = 117467392, // 0x07006900
		/// <summary>RevisionTableRow Object.</summary>
		kRevisionTableRowObject = 117469184, // 0x07007000
		/// <summary>RevisionTableCell Object.</summary>
		kRevisionTableCellObject = 117469440, // 0x07007100
		/// <summary>HoleTables Collection Object.</summary>
		kHoleTablesObject = 117469696, // 0x07007200
		/// <summary>HoleTable Object.</summary>
		kHoleTableObject = 117469952, // 0x07007300
		/// <summary>HoleTableColumns Collection Object.</summary>
		kHoleTableColumnsObject = 117470208, // 0x07007400
		/// <summary>HoleTableColumn Object.</summary>
		kHoleTableColumnObject = 117470464, // 0x07007500
		/// <summary>HoleTableRows Collection Object.</summary>
		kHoleTableRowsObject = 117470720, // 0x07007600
		/// <summary>HoleTableRow Object.</summary>
		kHoleTableRowObject = 117470976, // 0x07007700
		/// <summary>HoleTableCell Object.</summary>
		kHoleTableCellObject = 117471232, // 0x07007800
		/// <summary>HoleTag Object.</summary>
		kHoleTagObject = 117471488, // 0x07007900
		/// <summary>DrawingNotes Collection Object.</summary>
		kDrawingNotesObject = 117471744, // 0x07007A00
		/// <summary>DrawingNote Object.</summary>
		kDrawingNoteObject = 117472000, // 0x07007B00
		/// <summary>GeneralNotes Collection Object.</summary>
		kGeneralNotesObject = 117472256, // 0x07007C00
		/// <summary>GeneralNote Colection Object.</summary>
		kGeneralNoteObject = 117472512, // 0x07007D00
		/// <summary>LeaderNotes Collection Object.</summary>
		kLeaderNotesObject = 117472768, // 0x07007E00
		/// <summary>LeaderNote Collection Object.</summary>
		kLeaderNoteObject = 117473024, // 0x07007F00
		/// <summary>CallOut Symbol Object.</summary>
		kCallOutSymbolObject = 117473280, // 0x07008000
		/// <summary>DrawingCurvesEnumerator Object.</summary>
		kDrawingCurvesEnumeratorObject = 117473536, // 0x07008100
		/// <summary>DrawingCurve Object.</summary>
		kDrawingCurveObject = 117473792, // 0x07008200
		/// <summary>GeometryIntent Object.</summary>
		kGeometryIntentObject = 117474048, // 0x07008300
		/// <summary>Drawing Detail View Object.</summary>
		kDetailDrawingViewObject = 117474304, // 0x07008400
		/// <summary>LinearGeneralDimension Object.</summary>
		kLinearGeneralDimensionObject = 117474560, // 0x07008500
		/// <summary>AngularGeneralDimension Object.</summary>
		kAngularGeneralDimensionObject = 117474816, // 0x07008600
		/// <summary>RadiusGeneralDimension Object.</summary>
		kRadiusGeneralDimensionObject = 117475072, // 0x07008700
		/// <summary>DiameterGeneralDimension Object.</summary>
		kDiameterGeneralDimensionObject = 117475328, // 0x07008800
		/// <summary>Leader Object.</summary>
		kLeaderObject = 117475584, // 0x07008900
		/// <summary>LeaderNode Object.</summary>
		kLeaderNodeObject = 117477376, // 0x07009000
		/// <summary>LeaderNodesEnumerator Object.</summary>
		kLeaderNodesEnumeratorObject = 117477632, // 0x07009100
		/// <summary>DrawingCurveSegments Object.</summary>
		kDrawingCurveSegmentsObject = 117477888, // 0x07009200
		/// <summary>DrawingCurveSegment Object.</summary>
		kDrawingCurveSegmentObject = 117478144, // 0x07009300
		/// <summary>Style Object.</summary>
		kStyleObject = 117478400, // 0x07009400
		/// <summary>Styles Object.</summary>
		kStylesObject = 117478656, // 0x07009500
		/// <summary>Centermarks Object.</summary>
		kCentermarksObject = 117478912, // 0x07009600
		/// <summary>Centermark Object.</summary>
		kCentermarkObject = 117479168, // 0x07009700
		/// <summary>Centerlines Object.</summary>
		kCenterlinesObject = 117479424, // 0x07009800
		/// <summary>Centerline Object.</summary>
		kCenterlineObject = 117479680, // 0x07009900
		/// <summary>DrawingViewLabel Object.</summary>
		kDrawingViewLabelObject = 117479936, // 0x07009A00
		/// <summary>LeaderStylesEnumerator Collection Object.</summary>
		kLeaderStylesEnumeratorObject = 117480192, // 0x07009B00
		/// <summary>LeaderStyle Object.</summary>
		kLeaderStyleObject = 117480448, // 0x07009C00
		/// <summary>BalloonStylesEnumerator Collection Object.</summary>
		kBalloonStylesEnumeratorObject = 117480704, // 0x07009D00
		/// <summary>BalloonStyle Object.</summary>
		kBalloonStyleObject = 117480960, // 0x07009E00
		/// <summary>FeatureControlFrameStylesEnumerator Collection Object.</summary>
		kFeatureControlFrameStylesEnumeratorObject = 117481216, // 0x07009F00
		/// <summary>FeatureControlFrameStyle Object.</summary>
		kFeatureControlFrameStyleObject = 117481472, // 0x0700A000
		/// <summary>SurfaceTextureStylesEnumerator Object.</summary>
		kSurfaceTextureStylesEnumeratorObject = 117481728, // 0x0700A100
		/// <summary>SurfaceTextureStyle Object.</summary>
		kSurfaceTextureStyleObject = 117481984, // 0x0700A200
		/// <summary>SheetFormats Collection Object.</summary>
		kSheetFormatsObject = 117482240, // 0x0700A300
		/// <summary>SheetFormat Object.</summary>
		kSheetFormatObject = 117482496, // 0x0700A400
		/// <summary>Collection object provides access to all the feature control frame symbols on a sheet and provides methods to create additional ones.</summary>
		kFeatureControlFramesObject = 117482752, // 0x0700A500
		/// <summary>Object represents a feature control frame symbol on a sheet.</summary>
		kFeatureControlFrameObject = 117483008, // 0x0700A600
		/// <summary>Object provides access to all rows of a feature control frame.  This object is also used to define the rows during feature control frame creation.</summary>
		kFeatureControlFrameRowsObject = 117483264, // 0x0700A700
		/// <summary>Object represents a row within a feature control frame symbol.</summary>
		kFeatureControlFrameRowObject = 117483520, // 0x0700A800
		/// <summary>SurfaceTextureSymbols Object.</summary>
		kSurfaceTextureSymbolsObject = 117483776, // 0x0700A900
		/// <summary>SurfaceTextureSymbol Object.</summary>
		kSurfaceTextureSymbolObject = 117484032, // 0x0700AA00
		/// <summary>OriginIndicator Object.</summary>
		kOriginIndicatorObject = 117484288, // 0x0700AB00
		/// <summary>OrdinateDimensions Object.</summary>
		kOrdinateDimensionsObject = 117484544, // 0x0700AC00
		/// <summary>OrdinateDimension Object.</summary>
		kOrdinateDimensionObject = 117484800, // 0x0700AD00
		/// <summary>HoleTableStylesEnumerator Object.</summary>
		kHoleTableStylesEnumeratorObject = 117485056, // 0x0700AE00
		/// <summary>HoleTableStyle Object.</summary>
		kHoleTableStyleObject = 117485312, // 0x0700AF00
		/// <summary>DrawingBOMs object represents the collection of all locally stored BOMs in the drawing document.</summary>
		kDrawingBOMsObject = 117485568, // 0x0700B000
		/// <summary>DrawingBOM object represents a locally stored BOM in the drawing document.</summary>
		kDrawingBOMObject = 117485824, // 0x0700B100
		/// <summary>DrawingBOMColumns object represents the collection of all columns of a drawing BOM.</summary>
		kDrawingBOMColumnsObject = 117486080, // 0x0700B200
		/// <summary>DrawingBOMColumn object represents a column within the drawing BOM.</summary>
		kDrawingBOMColumnObject = 117486336, // 0x0700B300
		/// <summary>DrawingBOMRows object represents the collection of all rows of a drawing BOM.</summary>
		kDrawingBOMRowsObject = 117486592, // 0x0700B400
		/// <summary>DrawingBOMRow object represents a row within the drawing BOM.</summary>
		kDrawingBOMRowObject = 117486848, // 0x0700B500
		/// <summary>The DrawingBOMCell object represents a cell within the drawing BOM.</summary>
		kDrawingBOMCellObject = 117487104, // 0x0700B600
		/// <summary>The HoleNote object represents Hole Note.</summary>
		kHoleNoteObject = 117487360, // 0x0700B700
		/// <summary>The BendNote object represents Bend Note.</summary>
		kBendNoteObject = 117487616, // 0x0700B800
		/// <summary>The ThreadNote object represents Thread Note.</summary>
		kThreadNoteObject = 117487872, // 0x0700B900
		/// <summary>The PunchNote object represents Punch Note.</summary>
		kPunchNoteObject = 117488128, // 0x0700BA00
		/// <summary>The ChamferNote object represents Chamfer Note.</summary>
		kChamferNoteObject = 117488384, // 0x0700BB00
		/// <summary>The BaselineDimensionSets collection object provides access to all of the baseline dimension sets on the sheet.</summary>
		kBaselineDimensionSetsObject = 117488640, // 0x0700BC00
		/// <summary>The BaselineDimensionSet object represents a baseline dimension set placed on a sheet.</summary>
		kBaselineDimensionSetObject = 117488896, // 0x0700BD00
		/// <summary>The OrdinateDimensionSets collection object provides access to all of the ordinate dimension sets on the sheet.</summary>
		kOrdinateDimensionSetsObject = 117489152, // 0x0700BE00
		/// <summary>The OrdinateDimensionSet object represents an ordinate dimension set placed on a sheet.</summary>
		kOrdinateDimensionSetObject = 117489408, // 0x0700BF00
		/// <summary>OrdinateDimensions enumerator Object.</summary>
		kOrdinateDimensionsEnumeratorObject = 117489664, // 0x0700C000
		/// <summary>AutomatedCenterlineSettings Object.</summary>
		kAutomatedCenterlineSettingsObject = 117489920, // 0x0700C100
		/// <summary>CentermarkStyles Enumerator Object.</summary>
		kCentermarkStylesEnumeratorObject = 117490176, // 0x0700C200
		/// <summary>CentermarkStyle Object.</summary>
		kCentermarkStyleObject = 117490432, // 0x0700C300
		/// <summary>ChamferNotes Collection Object.</summary>
		kChamferNotesObject = 117490688, // 0x0700C400
		/// <summary>BendNotes Collection Object.</summary>
		kBendNotesObject = 117490944, // 0x0700C500
		/// <summary>PunchNotes Collection Object.</summary>
		kPunchNotesObject = 117491200, // 0x0700C600
		/// <summary>HoleThreadNotes Collection Object.</summary>
		kHoleThreadNotesObject = 117491456, // 0x0700C700
		/// <summary>HoleThreadNote Object.</summary>
		kHoleThreadNoteObject = 117491712, // 0x0700C800
		/// <summary>BreakOperations object represents all break operations applied to a drawing view.</summary>
		kBreakOperationsObject = 117491968, // 0x0700C900
		/// <summary>BreakOperation object represents a break applied to a drawing view.</summary>
		kBreakOperationObject = 117492224, // 0x0700CA00
		/// <summary>BreakOutOperations object represents all break out operations applied to a drawing view.</summary>
		kBreakOutOperationsObject = 117492480, // 0x0700CB00
		/// <summary>BreakOutOperation object represents a break out applied to a drawing view.</summary>
		kBreakOutOperationObject = 117492736, // 0x0700CC00
		/// <summary>ChainDimensionSets Object.</summary>
		kChainDimensionSetsObject = 117492992, // 0x0700CD00
		/// <summary>ChainDimensionSet Object.</summary>
		kChainDimensionSetObject = 117493248, // 0x0700CE00
		/// <summary>PartsListStylesEnumerator Object.</summary>
		kPartsListStylesEnumeratorObject = 117493504, // 0x0700CF00
		/// <summary>PartsListStyle Object.</summary>
		kPartsListStyleObject = 117493760, // 0x0700D000
		/// <summary>RevisionTableStylesEnumerator Object.</summary>
		kRevisionTableStylesEnumeratorObject = 117494016, // 0x0700D100
		/// <summary>RevisionTableStyle Object.</summary>
		kRevisionTableStyleObject = 117494272, // 0x0700D200
		/// <summary>TableStylesEnumerator Object.</summary>
		kTableStylesEnumeratorObject = 117494528, // 0x0700D300
		/// <summary>TableStyle Object.</summary>
		kTableStyleObject = 117494784, // 0x0700D400
		/// <summary>ACADBlockDefinitions Object.</summary>
		kACADBlockDefinitionsObject = 117495040, // 0x0700D500
		/// <summary>ACADBlockDefinition Object.</summary>
		kACADBlockDefinitionObject = 117495296, // 0x0700D600
		/// <summary>ACADBlockReferences Object.</summary>
		kACADBlocksObject = 117495552, // 0x0700D700
		/// <summary>ACADBlockReference Object.</summary>
		kACADBlockObject = 117495808, // 0x0700D800
		/// <summary>ACADBlockDefinitionsEnumerator.</summary>
		kACADBlockDefinitionsEnumeratorObject = 117496064, // 0x0700D900
		/// <summary>SketchedSymbolDefinitionLibraries Object.</summary>
		kSketchedSymbolDefinitionLibrariesObject = 117496320, // 0x0700DA00
		/// <summary>SketchedSymbolDefinitionLibrary Object.</summary>
		kSketchedSymbolDefinitionLibraryObject = 117496576, // 0x0700DB00
		/// <summary>SurfaceTextureSymbolDefinition Object.</summary>
		kSurfaceTextureSymbolDefinitionObject = 117496832, // 0x0700DC00
		/// <summary>SurfaceTextureANSIDefinition Object.</summary>
		kSurfaceTextureANSIDefinitionObject = 117497088, // 0x0700DD00
		/// <summary>SurfaceTextureBSIDefinition Object.</summary>
		kSurfaceTextureBSIDefinitionObject = 117497344, // 0x0700DE00
		/// <summary>SurfaceTextureDINDefinition Object.</summary>
		kSurfaceTextureDINDefinitionObject = 117497600, // 0x0700DF00
		/// <summary>SurfaceTextureGBDefinition Object.</summary>
		kSurfaceTextureGBDefinitionObject = 117497856, // 0x0700E000
		/// <summary>SurfaceTextureGOSTDefinition Object.</summary>
		kSurfaceTextureGOSTDefinitionObject = 117498112, // 0x0700E100
		/// <summary>SurfaceTextureISODefinition Object.</summary>
		kSurfaceTextureISODefinitionObject = 117498368, // 0x0700E200
		/// <summary>SurfaceTextureJISDefinition Object.</summary>
		kSurfaceTextureJISDefinitionObject = 117498624, // 0x0700E300
		/// <summary>LibrarySketchedSymbolDefinition Object.</summary>
		kLibrarySketchedSymbolDefinitionObject = 117498880, // 0x0700E400
		/// <summary>LibrarySketchedSymbolDefinitions Object.</summary>
		kLibrarySketchedSymbolDefinitionsObject = 117499136, // 0x0700E500
		/// <summary>LibraryFolder Object.</summary>
		kLibraryFolderObject = 117499392, // 0x0700E600
		/// <summary>LibraryFolders Object.</summary>
		kLibraryFoldersObject = 117499648, // 0x0700E700
		/// <summary>UnitAttributes Object.</summary>
		kUnitAttributesObject = 117499904, // 0x0700E800
		/// <summary>DrawingHatchPattern Object.</summary>
		kDrawingHatchPatternObject = 117500160, // 0x0700E900
		/// <summary>DrawingHatchPatternsManager Object.</summary>
		kDrawingHatchPatternsManagerObject = 117500416, // 0x0700EA00
		/// <summary>UnitsFormatting Object.</summary>
		kUnitsFormattingObject = 117500672, // 0x0700EB00
		/// <summary>DrawingViewHatchRegion Object.</summary>
		kDrawingViewHatchRegionObject = 117500928, // 0x0700EC00
		/// <summary>DrawingViewHatchRegions Object.</summary>
		kDrawingViewHatchRegionsObject = 117501184, // 0x0700ED00
		/// <summary>DrawingViewHatchArea Object.</summary>
		kDrawingViewHatchAreaObject = 117501440, // 0x0700EE00
		/// <summary>DrawingViewHatchAreas Object.</summary>
		kDrawingViewHatchAreasObject = 117501696, // 0x0700EF00
		/// <summary>The AuxiliaryFeatureIndicators object allows to access the auxiliary feature indicators within a feature control frame symbol.</summary>
		kAuxiliaryFeatureIndicatorsObject = 117501952, // 0x0700F000
		/// <summary>The AuxiliaryFeatureIndicator object.</summary>
		kAuxiliaryFeatureIndicatorObject = 117502208, // 0x0700F100
		/// <summary>WeldSymbolStyle object.</summary>
		kWeldSymbolStyleObject = 117502464, // 0x0700F200
		/// <summary>WeldSymbolStylesEnumerator object.</summary>
		kWeldSymbolStylesEnumeratorObject = 117502720, // 0x0700F300
		/// <summary>DrawingWeldingSymbols object.</summary>
		kDrawingWeldingSymbolsObject = 117502976, // 0x0700F400
		/// <summary>DrawingWeldingSymbolDefinition object.</summary>
		kDrawingWeldingSymbolDefinitionObject = 117503232, // 0x0700F500
		/// <summary>WeldSymbol object.</summary>
		kWeldSymbolObject = 117503488, // 0x0700F600
		/// <summary>DrawingWeldingSymbol object.</summary>
		kDrawingWeldingSymbolObject = 117503744, // 0x0700F700
		/// <summary>DrawingWeldingSymbolDefinitions object.</summary>
		kDrawingWeldingSymbolDefinitionsObject = 117504000, // 0x0700F800
		/// <summary>EdgeSymbolStylesEnumerator object.</summary>
		kEdgeSymbolStylesEnumeratorObject = 117504256, // 0x0700F900
		/// <summary>EdgeSymbolStyle object.</summary>
		kEdgeSymbolStyleObject = 117504512, // 0x0700FA00
		/// <summary>RevisionCloudDefinition object.</summary>
		kRevisionCloudDefinitionObject = 117504768, // 0x0700FB00
		/// <summary>RevisionCloudControlPoint object.</summary>
		kRevisionCloudControlPointObject = 117505024, // 0x0700FC00
		/// <summary>RevisionCloudControlPoints object.</summary>
		kRevisionCloudControlPointsObject = 117505280, // 0x0700FD00
		/// <summary>RevisionCloud object.</summary>
		kRevisionCloudObject = 117505536, // 0x0700FE00
		/// <summary>RevisionClouds object.</summary>
		kRevisionCloudsObject = 117505792, // 0x0700FF00
		/// <summary>EdgeSymbol object.</summary>
		kEdgeSymbolObject = 117506048, // 0x07010000
		/// <summary>EdgeSymbols object.</summary>
		kEdgeSymbolsObject = 117506304, // 0x07010100
		/// <summary>EdgeSymbolDefinition object.</summary>
		kEdgeSymbolDefinitionObject = 117506560, // 0x07010200
		/// <summary>PartsListFilterSettings object.</summary>
		kPartsListFilterSettingsObject = 117506816, // 0x07010300
		/// <summary>PartsListFilterItem object.</summary>
		kPartsListFilterItemObject = 117507072, // 0x07010400
		/// <summary>PartsListSortSettings object.</summary>
		kPartsListSortSettingsObject = 117507328, // 0x07010500
		/// <summary>RevisionIndexDefaults object.</summary>
		kRevisionIndexDefaultsObject = 117507584, // 0x07010600
		/// <summary>Presentation Exploded Views Collection Object.</summary>
		kExplodedViewsObject = 134218496, // 0x08000300
		/// <summary>Presentation Exploded View Object.</summary>
		kExplodedViewObject = 134218752, // 0x08000400
		/// <summary>TrailsEnumerator Object.</summary>
		kTrailsEnumeratorObject = 134219008, // 0x08000500
		/// <summary>Presentation Trail Object.</summary>
		kTrailObject = 134219264, // 0x08000600
		/// <summary>Presentation Trail Segment Object.</summary>
		kTrailSegmentObject = 134219520, // 0x08000700
		/// <summary>Publications collection object.</summary>
		kPublicationsObject = 134219776, // 0x08000800
		/// <summary>Publication object.</summary>
		kPublicationObject = 134220032, // 0x08000900
		/// <summary>PublicationComponents collection object.</summary>
		kPublicationComponentsObject = 134220288, // 0x08000A00
		/// <summary>PublicationComponent object.</summary>
		kPublicationComponentObject = 134220544, // 0x08000B00
		/// <summary>PublicationComponentsEnumerator object.</summary>
		kPublicationComponentsEnumeratorObject = 134220800, // 0x08000C00
		/// <summary>Storyboards collection object.</summary>
		kStoryboardsObject = 134221056, // 0x08000D00
		/// <summary>Storyboard object.</summary>
		kStoryboardObject = 134221312, // 0x08000E00
		/// <summary>PublicationTweaks collection object.</summary>
		kPublicationTweaksObject = 134221568, // 0x08000F00
		/// <summary>PublicationTweakDefinition object.</summary>
		kPublicationTweakDefinitionObject = 134221824, // 0x08001000
		/// <summary>PublicationTweak object.</summary>
		kPublicationTweakObject = 134222080, // 0x08001100
		/// <summary>PublicationMarkedViews object.</summary>
		kPublicationMarkedViewsObject = 134222336, // 0x08001200
		/// <summary>PublicationMarkedView object.</summary>
		kPublicationMarkedViewObject = 134222592, // 0x08001300
		/// <summary>PublicationBodiesEnumerator object.</summary>
		kPublicationBodiesEnumeratorObject = 134222848, // 0x08001400
		/// <summary>PublicationBody object.</summary>
		kPublicationBodyObject = 134223104, // 0x08001500
		/// <summary>PublicationFacesEnumerator object.</summary>
		kPublicationFacesEnumeratorObject = 134223360, // 0x08001600
		/// <summary>PublicationFace object.</summary>
		kPublicationFaceObject = 134223616, // 0x08001700
		/// <summary>PublicationFaceShellsEnumerator object.</summary>
		kPublicationFaceShellsEnumeratorObject = 134223872, // 0x08001800
		/// <summary>PublicationFaceShell object.</summary>
		kPublicationFaceShellObject = 134224128, // 0x08001900
		/// <summary>PublicationEdgesEnumerator object.</summary>
		kPublicationEdgesEnumeratorObject = 134224384, // 0x08001A00
		/// <summary>PublicationEdge object.</summary>
		kPublicationEdgeObject = 134224640, // 0x08001B00
		/// <summary>PresentationSequences object.</summary>
		kPresentationSequencesObject = 134224896, // 0x08001C00
		/// <summary>PresentationSequence object.</summary>
		kPresentationSequenceObject = 134225152, // 0x08001D00
		/// <summary>PresentationTasks object.</summary>
		kPresentationTasksObject = 134225408, // 0x08001E00
		/// <summary>PresentationTask object.</summary>
		kPresentationTaskObject = 134225664, // 0x08001F00
		/// <summary>PresentationTweaks object.</summary>
		kPresentationTweaksObject = 134225920, // 0x08002000
		/// <summary>PresentationTweak object.</summary>
		kPresentationTweakObject = 134226176, // 0x08002100
		/// <summary>TrailSegmentsEnumerator Object.</summary>
		kTrailSegmentsEnumeratorObject = 134226432, // 0x08002200
		/// <summary>PresentationEvents Object.</summary>
		kPresentationEventsObject = 134226688, // 0x08002300
		/// <summary>PublicationVerticesEnumerator Object.</summary>
		kPublicationVerticesEnumeratorObject = 134226944, // 0x08002400
		/// <summary>PublicationVertex Object.</summary>
		kPublicationVertexObject = 134227200, // 0x08002500
		/// <summary>PublicationTweakPaths collection object.</summary>
		kPublicationTweakPathsObject = 134227456, // 0x08002600
		/// <summary>PublicationTweakPath object.</summary>
		kPublicationTweakPathObject = 134227712, // 0x08002700
		/// <summary>PublicationTrails object.</summary>
		kPublicationTrailsObject = 134227968, // 0x08002800
		/// <summary>PublicationTrail object.</summary>
		kPublicationTrailObject = 134228224, // 0x08002900
		/// <summary>PublicationTrailSegmentsEnumerator object.</summary>
		kPublicationTrailSegmentsEnumeratorObject = 134228480, // 0x08002A00
		/// <summary>PublicationTrailSegments object.</summary>
		kPublicationTrailSegmentsObject = 134228736, // 0x08002B00
		/// <summary>PublicationTrailSegment object.</summary>
		kPublicationTrailSegmentObject = 134228992, // 0x08002C00
		/// <summary>PublicationMeshFace Object.</summary>
		kPublicationMeshFaceObject = 134229248, // 0x08002D00
		/// <summary>PublicationMeshEdge Object.</summary>
		kPublicationMeshEdgeObject = 134229504, // 0x08002E00
		/// <summary>PublisherServer Object.</summary>
		kPublisherServerObject = 134229760, // 0x08002F00
		/// <summary>PresentationScenes Object.</summary>
		kPresentationScenesObject = 134230016, // 0x08003000
		/// <summary>PresentationScene Object.</summary>
		kPresentationSceneObject = 134230272, // 0x08003100
		/// <summary>PresentationSnapshotViews Object.</summary>
		kPresentationSnapshotViewsObject = 134230528, // 0x08003200
		/// <summary>PresentationSnapshotView Object.</summary>
		kPresentationSnapshotViewObject = 134230784, // 0x08003300
		/// <summary>PresentationComponentsEnumerator Object.</summary>
		kPresentationComponentsEnumeratorObject = 134231040, // 0x08003400
		/// <summary>PresentationComponent Object.</summary>
		kPresentationComponentObject = 134231296, // 0x08003500
		/// <summary>PresentationBodiesEnumerator Object.</summary>
		kPresentationBodiesEnumeratorObject = 134231552, // 0x08003600
		/// <summary>PresentationBody Object.</summary>
		kPresentationBodyObject = 134231808, // 0x08003700
		/// <summary>PresentationFacesEnumerator Object.</summary>
		kPresentationFacesEnumeratorObject = 134232064, // 0x08003800
		/// <summary>PresentationFace Object.</summary>
		kPresentationFaceObject = 134232320, // 0x08003900
		/// <summary>PresentationEdgesEnumerator Object.</summary>
		kPresentationEdgesEnumeratorObject = 134232576, // 0x08003A00
		/// <summary>PresentationEdge Object.</summary>
		kPresentationEdgeObject = 134232832, // 0x08003B00
		/// <summary>PresentationVerticesEnumerator Object.</summary>
		kPresentationVerticesEnumeratorObject = 134233088, // 0x08003C00
		/// <summary>PresentationVertex Object.</summary>
		kPresentationVertexObject = 134233344, // 0x08003D00
		/// <summary>PresentationTrails Object.</summary>
		kPresentationTrailsObject = 134233600, // 0x08003E00
		/// <summary>PresentationTrail Object.</summary>
		kPresentationTrailObject = 134233856, // 0x08003F00
		/// <summary>PresentationTrailSegmentsEnumerator Object.</summary>
		kPresentationTrailSegmentsEnumeratorObject = 134234112, // 0x08004000
		/// <summary>PresentationTrailSegments Object.</summary>
		kPresentationTrailSegmentsObject = 134234368, // 0x08004100
		/// <summary>PresentationTrailSegment Object.</summary>
		kPresentationTweakTrailSegmentObject = 134234624, // 0x08004200
		/// <summary>PresentationMeshFeatureSetsEnumerator Object.</summary>
		kPresentationMeshFeatureSetsEnumeratorObject = 134234880, // 0x08004300
		/// <summary>PresentationMeshFeatureSet Object.</summary>
		kPresentationMeshFeatureSetObject = 134235136, // 0x08004400
		/// <summary>PresentationMeshFeature Object.</summary>
		kPresentationMeshFeatureObject = 134235392, // 0x08004500
		/// <summary>PresentationMeshFeatureEntitiesEnumerator Object.</summary>
		kPresentationMeshFeatureEntitiesEnumeratorObject = 134235648, // 0x08004600
		/// <summary>PresentationMeshFeatureEntity Object.</summary>
		kPresentationMeshFeatureEntityObject = 134235904, // 0x08004700
		/// <summary>SheetMetal Component Definition Object.</summary>
		kSheetMetalComponentDefinitionObject = 150995200, // 0x09000100
		/// <summary>SheetMetal Features Object.</summary>
		kSheetMetalFeaturesObject = 150995456, // 0x09000200
		/// <summary>Sheet Metal Styles Object.</summary>
		kSheetMetalStylesObject = 150995712, // 0x09000300
		/// <summary>Sheet Metal Style Object.</summary>
		kSheetMetalStyleObject = 150995968, // 0x09000400
		/// <summary>Unfold Methods Object.</summary>
		kUnfoldMethodsObject = 150996224, // 0x09000500
		/// <summary>Unfold Method Object.</summary>
		kUnfoldMethodObject = 150996480, // 0x09000600
		/// <summary>Corner Features Object.</summary>
		kCornerFeaturesObject = 150996736, // 0x09000700
		/// <summary>Corner Feature Object.</summary>
		kCornerFeatureObject = 150996992, // 0x09000800
		/// <summary>Bend Features Object.</summary>
		kBendFeaturesObject = 150997248, // 0x09000900
		/// <summary>Bend Feature Object.</summary>
		kBendFeatureObject = 150997504, // 0x09000A00
		/// <summary>Cut Features Object.</summary>
		kCutFeaturesObject = 150997760, // 0x09000B00
		/// <summary>Cut Feature Object.</summary>
		kCutFeatureObject = 150998016, // 0x09000C00
		/// <summary>Hem Features Object.</summary>
		kHemFeaturesObject = 150998272, // 0x09000D00
		/// <summary>Hem Feature Object.</summary>
		kHemFeatureObject = 150998528, // 0x09000E00
		/// <summary>Fold Features Object.</summary>
		kFoldFeaturesObject = 150998784, // 0x09000F00
		/// <summary>Fold Feature Object.</summary>
		kFoldFeatureObject = 150999296, // 0x09001100
		/// <summary>Corner Round Features Object.</summary>
		kCornerRoundFeaturesObject = 150999552, // 0x09001200
		/// <summary>Corner Round Feature Object.</summary>
		kCornerRoundFeatureObject = 150999808, // 0x09001300
		/// <summary>Corner Chamfer Features Object.</summary>
		kCornerChamferFeaturesObject = 151000064, // 0x09001400
		/// <summary>Corner Chamfer Feature Object.</summary>
		kCornerChamferFeatureObject = 151000320, // 0x09001500
		/// <summary>Face Features Object.</summary>
		kFaceFeaturesObject = 151000576, // 0x09001600
		/// <summary>Face Feature Object.</summary>
		kFaceFeatureObject = 151000832, // 0x09001700
		/// <summary>Flange Features Object.</summary>
		kFlangeFeaturesObject = 151001088, // 0x09001800
		/// <summary>Flange Feature Object.</summary>
		kFlangeFeatureObject = 151001344, // 0x09001900
		/// <summary>Counter Flange Features Object.</summary>
		kContourFlangeFeaturesObject = 151001600, // 0x09001A00
		/// <summary>Counter Flange Feature Object.</summary>
		kContourFlangeFeatureObject = 151001856, // 0x09001B00
		/// <summary>Punch Tool Features Object.</summary>
		kPunchToolFeaturesObject = 151002112, // 0x09001C00
		/// <summary>Punch Tool Feature Object.</summary>
		kPunchToolFeatureObject = 151002368, // 0x09001D00
		/// <summary>FlatPattern Object.</summary>
		kFlatPatternObject = 151002624, // 0x09001E00
		/// <summary>Corner Feature Proxy Object.</summary>
		kCornerFeatureProxyObject = 151002880, // 0x09001F00
		/// <summary>Bend Feature Proxy Object.</summary>
		kBendFeatureProxyObject = 151003136, // 0x09002000
		/// <summary>Cut Feature Proxy Object.</summary>
		kCutFeatureProxyObject = 151003392, // 0x09002100
		/// <summary>Hem Feature Proxy Object.</summary>
		kHemFeatureProxyObject = 151003648, // 0x09002200
		/// <summary>Fold Feature Proxy Object.</summary>
		kFoldFeatureProxyObject = 151003904, // 0x09002300
		/// <summary>Corner Round Feature Proxy Object.</summary>
		kCornerRoundFeatureProxyObject = 151004160, // 0x09002400
		/// <summary>Corner Chamfer Feature Proxy Object.</summary>
		kCornerChamferFeatureProxyObject = 151004416, // 0x09002500
		/// <summary>Face Feature Proxy Object.</summary>
		kFaceFeatureProxyObject = 151004672, // 0x09002600
		/// <summary>Flange Feature Proxy Object.</summary>
		kFlangeFeatureProxyObject = 151004928, // 0x09002700
		/// <summary>Counter Flange Feature Proxy Object.</summary>
		kContourFlangeFeatureProxyObject = 151005184, // 0x09002800
		/// <summary>Punch Tool Feature Proxy Object.</summary>
		kPunchToolFeatureProxyObject = 151005440, // 0x09002900
		/// <summary>FlatBendResults Object.</summary>
		kFlatBendResultsObject = 151005696, // 0x09002A00
		/// <summary>FlatBendResult Object.</summary>
		kFlatBendResultObject = 151005952, // 0x09002B00
		/// <summary>FlatPunchResults Object.</summary>
		kFlatPunchResultsObject = 151006208, // 0x09002C00
		/// <summary>FlatPunchResult Object.</summary>
		kFlatPunchResultObject = 151006464, // 0x09002D00
		/// <summary>Face Feature Definition Object.</summary>
		kFaceFeatureDefinitionObject = 151006720, // 0x09002E00
		/// <summary>Bend Definition Object.</summary>
		kBendDefinitionObject = 151006976, // 0x09002F00
		/// <summary>Bend Options Object.</summary>
		kBendOptionsObject = 151009792, // 0x09003A00
		/// <summary>Corner Definition Object.</summary>
		kCornerOptionsObject = 151010048, // 0x09003B00
		/// <summary>Cut modeling feature definition object.</summary>
		kCutDefinitionObject = 151010304, // 0x09003C00
		/// <summary>Fold modeling feature definition object.</summary>
		kFoldDefinitionObject = 151010560, // 0x09003D00
		/// <summary>Corner modeling feature definition object.</summary>
		kCornerDefinitionObject = 151010816, // 0x09003E00
		/// <summary>CornerRound Definition Object.</summary>
		kCornerRoundDefinitionObject = 151011072, // 0x09003F00
		/// <summary>Corner round edge set Object.</summary>
		kCornerRoundEdgeSetObject = 151011328, // 0x09004000
		/// <summary>CornerChamfer Definition Object.</summary>
		kCornerChamferDefinitionObject = 151011584, // 0x09004100
		/// <summary>Flange Definition Object.</summary>
		kFlangeDefinitionObject = 151011840, // 0x09004200
		/// <summary>ContourFlange definition object.</summary>
		kContourFlangeDefinitionObject = 151014144, // 0x09004B00
		/// <summary>Hem definition object.</summary>
		kHemDefinitionObject = 151014400, // 0x09004C00
		/// <summary>Object defines all of the inputs that are unique to a single style of hem feature.</summary>
		kSingleHemDefinitionObject = 151014656, // 0x09004D00
		/// <summary>Object defines all of the inputs that are unique to a tear drop style of hem feature.</summary>
		kTeardropHemDefinitionObject = 151014912, // 0x09004E00
		/// <summary>Object defines all of the inputs that are unique to a rolled style of hem feature.</summary>
		kRolledHemDefinitionObject = 151015168, // 0x09004F00
		/// <summary>Object defines all of the inputs that are unique to a double style of hem feature.</summary>
		kDoubleHemDefinitionObject = 151015424, // 0x09005000
		/// <summary>Lofted Flange Feature Object.</summary>
		kLoftedFlangeFeatureObject = 151015680, // 0x09005100
		/// <summary>Lofted Flange Features Collection Object.</summary>
		kLoftedFlangeFeaturesObject = 151015936, // 0x09005200
		/// <summary>Lofted Flange Feature Proxy Object.</summary>
		kLoftedFlangeFeatureProxyObject = 151016192, // 0x09005300
		/// <summary>Unfold Object.</summary>
		kUnfoldFeatureObject = 151016448, // 0x09005400
		/// <summary>Unfolds Object.</summary>
		kUnfoldFeaturesObject = 151016704, // 0x09005500
		/// <summary>Unfold Feature Proxy Object.</summary>
		kUnfoldFeatureProxyObject = 151016960, // 0x09005600
		/// <summary>Refold Object.</summary>
		kRefoldFeatureObject = 151017216, // 0x09005700
		/// <summary>Refolds Object.</summary>
		kRefoldFeaturesObject = 151017472, // 0x09005800
		/// <summary>Refold Feature Proxy Object.</summary>
		kRefoldFeatureProxyObject = 151017728, // 0x09005900
		/// <summary>Bend.</summary>
		kBendObject = 151019520, // 0x09006000
		/// <summary>BendsEnumerator.</summary>
		kBendsEnumeratorObject = 151019776, // 0x09006100
		/// <summary>Rip Feature Object.</summary>
		kRipFeatureObject = 151020032, // 0x09006200
		/// <summary>Rip Feature Proxy Object.</summary>
		kRipFeatureProxyObject = 151020288, // 0x09006300
		/// <summary>Rip Features Object.</summary>
		kRipFeaturesObject = 151020544, // 0x09006400
		/// <summary>ContourRoll Feature Object.</summary>
		kContourRollFeatureObject = 151020800, // 0x09006500
		/// <summary>ContourRoll Feature Proxy Object.</summary>
		kContourRollFeatureProxyObject = 151021056, // 0x09006600
		/// <summary>ContourRoll Features Object.</summary>
		kContourRollFeaturesObject = 151021312, // 0x09006700
		/// <summary>Cosmetic Bend Feature Object.</summary>
		kCosmeticBendFeatureObject = 151021824, // 0x09006900
		/// <summary>Cosmetic Bend Feature Proxy Object.</summary>
		kCosmeticBendFeatureProxyObject = 151022080, // 0x09006A00
		/// <summary>Cosmetic Bend Features Object.</summary>
		kCosmeticBendFeaturesObject = 151022336, // 0x09006B00
		/// <summary>The RipDefinition object represents all of the information that defines a rip feature.</summary>
		kRipDefinitionObject = 151022592, // 0x09006C00
		/// <summary>SinglePointRipTypeDef Object.</summary>
		kSinglePointRipTypeDefObject = 151022848, // 0x09006D00
		/// <summary>PointToPointRipTypeDef Object.</summary>
		kPointToPointRipTypeDefObject = 151023104, // 0x09006E00
		/// <summary>LoftedFlangeDefinition Object.</summary>
		kLoftedFlangeDefinitionObject = 151023360, // 0x09006F00
		/// <summary>FlatPatternOrientations Object.</summary>
		kFlatPatternOrientationsObject = 151023616, // 0x09007000
		/// <summary>FlatPatternOrientation Object.</summary>
		kFlatPatternOrientationObject = 151023872, // 0x09007100
		/// <summary>FlatPatternPlate Object.</summary>
		kFlatPatternPlateObject = 151024128, // 0x09007200
		/// <summary>FlatPatternPlates Object.</summary>
		kFlatPatternPlatesObject = 151024384, // 0x09007300
		/// <summary>ASideDefinitions Object.</summary>
		kASideDefinitions = 151024896, // 0x09007500
		/// <summary>ASideDefinition Object.</summary>
		kASideDefinition = 151025152, // 0x09007600
		/// <summary>Flange edge set Object.</summary>
		kFlangeEdgeSetObject = 151025408, // 0x09007700
		/// <summary>Edges object.</summary>
		kEdgesObject = 1073963056, // 0x40036030
		/// <summary>Generic collection of weakly-typed Object (IDispatch).</summary>
		kObjectCollectionObject = 2130706443, // 0x7F00000B
		/// <summary>Generic collection of weakly-typed Object (IDispatch).</summary>
		kObjectCollectionByVariantObject = 2130706444, // 0x7F00000C
		/// <summary>Generic object. Weakly-typed (IDispatch).</summary>
		kGenericObject = 2130706445, // 0x7F00000D
		/// <summary>Generic enumerator for a group of objects. This enumerator will also support the IEnumxxx-style interface (IEnumUnknown, unless otherwise specified).</summary>
		kObjectsEnumeratorObject = 2130706450, // 0x7F000012
		/// <summary>Generic enumerator for a group of objects. This enumerator will also support the IEnumxxx-style interface (IEnumUnknown, unless otherwise specified).</summary>
		kObjectsEnumeratorByVariantObject = 2130706451, // 0x7F000013
		/// <summary>Oblikovati object of unknown/unsupported type.</summary>
		kUnknownObject = 2130706483, // 0x7F000033
	};
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- END
}
