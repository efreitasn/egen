package egen

import (
	"bytes"
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
		jpeg.Encode(&buff, resizedImg, nil)
	case "png":
		png.Encode(&buff, resizedImg)
	}

	return buff.Bytes(), nil
}
