/*

Package mjpeg contains an MJPEG video format writer.

*/
package mjpeg

import (
	"image"
	"os"
)

// AviWriter is an *.avi video writer.
// The video codec is MJPEG.
type AviWriter interface {
	// AddJpegFile adds a frame from a JPEG file.
	AddJpegFile(name string) error

	// AddJpeg adds a frame from a JPEG encoded data slice.
	AddJpeg(data []byte) error

	// AddImage adds a frame by encoding the specified Image.
	AddImage(img image.Image) error

	// Close finalizes and closes the avi file.
	Close() error
}

// aviWriter is the AviWriter implementation.
type aviWriter struct {
	// aviFile is the name of the file to write the result to
	aviFile string
	// width is the width of the video
	width int
	// height is the height of the video
	height int
	// fps is the frames/second (the "speed") of the video
	fps float64

	// af is the avi file descriptor
	af *os.File
}

// New returns a new AviWriter.
// The Close() method of the AviWriter must be called to finalize the video file.
func New(aviFile string, width, height int, fps float64) (awr AviWriter, err error) {
	aw := &aviWriter{
		aviFile: aviFile,
		width:   width,
		height:  height,
		fps:     fps,
	}

	aw.af, err = os.Create(aviFile)
	if err != nil {
		return nil, err
	}
	return aw, nil
}

// AddJpegFile implements AviWriter.AddJpegFile().
func (aw *aviWriter) AddJpegFile(name string) error {
	// TODO
	return nil
}

// AddJpeg implements AviWriter.AddJpeg().
func (aw *aviWriter) AddJpeg(data []byte) error {
	// TODO
	return nil
}

// AddImage implements AviWriter.AddImage().
func (aw *aviWriter) AddImage(mg image.Image) error {
	// TODO
	return nil
}

func (aw *aviWriter) Close() error {
	// TODO
	return nil
}
