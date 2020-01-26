package egen

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
)

func imgDimensions(filePath string) (width, height int, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return -1, -1, err
	}

	c, _, err := image.DecodeConfig(file)
	if err != nil {
		return -1, -1, err
	}

	return c.Width, c.Height, nil
}
