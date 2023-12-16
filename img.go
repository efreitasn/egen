package egen

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"

	"github.com/nfnt/resize"
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

func resizeImg(width int, filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	img, format, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	var buff bytes.Buffer
	resizedImg := resize.Resize(uint(width), 0, img, resize.Bilinear)

	switch format {
	case "jpeg":
		err := jpeg.Encode(&buff, resizedImg, nil)
		if err != nil {
			return nil, fmt.Errorf("encoding jpeg: %w", err)
		}
	case "png":
		err := png.Encode(&buff, resizedImg)
		if err != nil {
			return nil, fmt.Errorf("encoding png: %w", err)
		}
	}

	return buff.Bytes(), nil
}
