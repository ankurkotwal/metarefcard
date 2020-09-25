package data

// Config contains all the configuration data for the app
type Config struct {
	AppName            string                  `yaml:"AppName"`
	Version            string                  `yaml:"Version"`
	DebugOutput        bool                    `yaml:"DebugOutput"`
	VerboseOutput      bool                    `yaml:"VerboseOutput"`
	DevicesModel       string                  `yaml:"DevicesModel"`
	FontsDir           string                  `yaml:"FontsDir"`
	InputFont          string                  `yaml:"InputFont"`
	InputFontSize      float64                 `yaml:"InputFontSize"`
	InputPixelInset    int                     `yaml:"InputPixelInset"`
	PixelMultiplier    float64                 `yaml:"PixelMultiplier"`
	BackgroundColour   string                  `yaml:"BackgroundColour"`
	ForegroundColour   string                  `yaml:"ForegroundColour"`
	AlternateColours   []string                `yaml:"AlternateColours"`
	ImagesDir          string                  `yaml:"ImagesDir"`
	DefaultImageWidth  int                     `yaml:"DefaultImageWidth"`
	DefaultImageHeight int                     `yaml:"DefaultImageHeight"`
	ImageSizeOverride  map[string]Dimensions2d `yaml:"ImageSizeOverride"` // Device Name -> Dimensions2d
}

// Dimensions2d contains width and height
type Dimensions2d struct {
	Width  int `yaml:"Width"`
	Height int `yaml:"Height"`
}
