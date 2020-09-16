package data

// Config contains all the configuration data for the app
type Config struct {
	AppName           string                  `yaml:"AppName"`
	Version           string                  `yaml:"Version"`
	DevicesModel      string                  `yaml:"DevicesModel"`
	FontsDir          string                  `yaml:"FontsDir"`
	InputFont         string                  `yaml:"InputFont"`
	InputFontSize     int                     `yaml:"InputFontSize"`
	InputPixelInset   int                     `yaml:"InputPixelInset"`
	PixelMultiplier   float32                 `yaml:"PixelMultiplier"`
	ImagesDir         string                  `yaml:"ImagesDir"`
	DefaultImageSize  Dimensions2d            `yaml:"DefaultImageSize"`
	ImageSizeOverride map[string]Dimensions2d `yaml:"ImageSizeOverride"` // Device Name -> Dimensions2d
}

// Dimensions2d contains width and height
type Dimensions2d struct {
	Width  int `yaml:"Width"`
	Height int `yaml:"Height"`
}
