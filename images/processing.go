package images

import (
	"image"
	"image/draw"
	"image/jpeg"
	"net/http"

	images "github.com/vitali-fedulov/images"
)

// IsSameImage compares the two images
func IsSameImage(imgA *image.RGBA, imgB *image.RGBA) bool {
	hashA, imgSizeA := images.Hash(imgA)
	hashB, imgSizeB := images.Hash(imgB)
	return images.Similar(hashA, hashB, imgSizeA, imgSizeB)
}

// DownloadJPG downloads jpeg image
func DownloadJPG(url string) (*image.RGBA, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	img, err := jpeg.Decode(resp.Body)
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	m := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(m, m.Bounds(), img, b.Min, draw.Src)
	return m, nil
}
