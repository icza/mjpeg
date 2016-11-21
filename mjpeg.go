package mjpeg

import "image"

// AviWriter is an *.avi video writer.
// The video codec is MJPEG.
type AviWriter interface {
	// AddJpegFile adds a frame from a JPEG file.
	AddJpegFile(name string) error

	// AddJpeg adds a frame from a JPEG encoded data slice.
	AddJpeg(data []byte) error

	// AddImage adds a frame by encoding the specified Image.
	AddImage(img image.Image) error
}
