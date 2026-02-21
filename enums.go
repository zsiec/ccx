package ccx

// PenSize represents the size of the caption text pen.
type PenSize int

const (
	PenSizeSmall    PenSize = 0
	PenSizeStandard PenSize = 1
	PenSizeLarge    PenSize = 2
)

// FontTag identifies the font style for CEA-708 captions.
type FontTag int

const (
	FontDefault   FontTag = 0
	FontMonoSerif FontTag = 1
	FontPropSerif FontTag = 2
	FontMonoSans  FontTag = 3
	FontPropSans  FontTag = 4
	FontCasual    FontTag = 5
	FontCursive   FontTag = 6
	FontSmallCaps FontTag = 7
)

// PenOffset represents vertical text offset (subscript/superscript).
type PenOffset int

const (
	PenOffsetSubscript   PenOffset = 0
	PenOffsetNormal      PenOffset = 1
	PenOffsetSuperscript PenOffset = 2
)

// EdgeType defines the text edge rendering style.
type EdgeType int

const (
	EdgeNone            EdgeType = 0
	EdgeRaised          EdgeType = 1
	EdgeDepressed       EdgeType = 2
	EdgeUniform         EdgeType = 3
	EdgeLeftDropShadow  EdgeType = 4
	EdgeRightDropShadow EdgeType = 5
)

// Opacity controls the transparency of colors.
type Opacity int

const (
	OpacitySolid       Opacity = 0
	OpacityFlash       Opacity = 1
	OpacityTranslucent Opacity = 2
	OpacityTransparent Opacity = 3
)

// BorderType defines the window border rendering style.
type BorderType int

const (
	BorderNone            BorderType = 0
	BorderRaised          BorderType = 1
	BorderDepressed       BorderType = 2
	BorderUniform         BorderType = 3
	BorderLeftDropShadow  BorderType = 4
	BorderRightDropShadow BorderType = 5
)

// PrintDirection controls the direction text flows within a window.
type PrintDirection int

const (
	PrintDirLeftToRight PrintDirection = 0
	PrintDirRightToLeft PrintDirection = 1
	PrintDirTopToBottom PrintDirection = 2
	PrintDirBottomToTop PrintDirection = 3
)

// ScrollDirection controls the direction content scrolls within a window.
type ScrollDirection int

const (
	ScrollDirLeftToRight ScrollDirection = 0
	ScrollDirRightToLeft ScrollDirection = 1
	ScrollDirTopToBottom ScrollDirection = 2
	ScrollDirBottomToTop ScrollDirection = 3
)

// DisplayEffect controls how a window appears on screen.
type DisplayEffect int

const (
	EffectSnap DisplayEffect = 0
	EffectFade DisplayEffect = 1
	EffectWipe DisplayEffect = 2
)

// TextJustification controls text alignment within a window.
type TextJustification int

const (
	JustifyLeft   TextJustification = 0
	JustifyRight  TextJustification = 1
	JustifyCenter TextJustification = 2
	JustifyFull   TextJustification = 3
)

// AnchorID identifies the anchor point of a CEA-708 window.
type AnchorID int

const (
	AnchorUpperLeft    AnchorID = 0
	AnchorUpperCenter  AnchorID = 1
	AnchorUpperRight   AnchorID = 2
	AnchorMiddleLeft   AnchorID = 3
	AnchorMiddleCenter AnchorID = 4
	AnchorMiddleRight  AnchorID = 5
	AnchorLowerLeft    AnchorID = 6
	AnchorLowerCenter  AnchorID = 7
	AnchorLowerRight   AnchorID = 8
)
