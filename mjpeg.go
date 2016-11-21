/*

Package mjpeg contains an MJPEG video format writer.

*/
package mjpeg

import (
	"encoding/binary"
	"image"
	"log"
	"os"
	"time"
)

// An empty 4-byte int value
var emptyInt = make([]byte, 4)

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
	width int32
	// height is the height of the video
	height int32
	// fps is the frames/second (the "speed") of the video
	fps int32

	// avif is the avi file descriptor
	avif *os.File
	// idxFile is the name of the index file
	idxFile string
	// idxf is the index file descriptor
	idxf *os.File

	// writeErr holds the last encountered write error (to avif)
	err error

	// lengthFields contains the file positions of the length fields
	// that are filled later
	lengthFields []int64

	// Position of the frames count fields
	framesCountFieldPos, framesCountFieldPos2 int64
	// Position of the MOVI chunk
	moviPos int64

	// frames is the number of frames written to the AVI file
	frames int

	// General buffers used to write int values.
	buf4, buf2 []byte
}

// New returns a new AviWriter.
// The Close() method of the AviWriter must be called to finalize the video file.
func New(aviFile string, width, height, fps int32) (awr AviWriter, err error) {
	aw := &aviWriter{
		aviFile:      aviFile,
		width:        width,
		height:       height,
		fps:          fps,
		idxFile:      aviFile + ".idx_",
		lengthFields: make([]int64, 0, 8),
		buf4:         make([]byte, 4),
		buf2:         make([]byte, 2),
	}

	defer func() {
		if err == nil {
			return
		}
		logErr := func(e error) {
			if e != nil {
				log.Printf("Error: %v\n", e)
			}
		}
		if aw.avif != nil {
			logErr(aw.avif.Close())
			logErr(os.Remove(aviFile))
		}
		if aw.idxf != nil {
			logErr(aw.idxf.Close())
			logErr(os.Remove(aw.idxFile))
		}
	}()

	aw.avif, err = os.Create(aviFile)
	if err != nil {
		return nil, err
	}
	aw.idxf, err = os.Create(aw.idxFile)
	if err != nil {
		return nil, err
	}

	writeStr, writeInt, writeShort, writeLengthField, finalizeLengthField :=
		aw.writeStr, aw.writeInt32, aw.writeInt16, aw.writeLengthField, aw.finalizeLengthField

	// Write AVI header
	writeStr("RIFF")        // RIFF type
	writeLengthField()      // File length (remaining bytes after this field) (nesting level 0)
	writeStr("AVI ")        // AVI signature
	writeStr("LIST")        // LIST chunk: data encoding
	writeLengthField()      // Chunk length (nesting level 1)
	writeStr("hdrl")        // LIST chunk type
	writeStr("avih")        // avih sub-chunk
	writeInt(0x38)          // Sub-chunk length excluding the first 8 bytes of avih signature and size
	writeInt(1000000 / fps) // Frame delay time in microsec
	writeInt(0)             // dwMaxBytesPerSec (maximum data rate of the file in bytes per second)
	writeInt(0)             // Reserved
	writeInt(0x10)          // dwFlags, 0x10 bit: AVIF_HASINDEX (the AVI file has an index chunk at the end of the file - for good performance); Windows Media Player can't even play it if index is missing!
	aw.framesCountFieldPos = aw.currentPos()
	writeInt(0)      // Number of frames
	writeInt(0)      // Initial frame for non-interleaved files; non interleaved files should set this to 0
	writeInt(1)      // Number of streams in the video; here 1 video, no audio
	writeInt(0)      // dwSuggestedBufferSize
	writeInt(width)  // Image width in pixels
	writeInt(height) // Image height in pixels
	writeInt(0)      // Reserved
	writeInt(0)
	writeInt(0)
	writeInt(0)

	// Write stream information
	writeStr("LIST")   // LIST chunk: stream headers
	writeLengthField() // Chunk size (nesting level 2)
	writeStr("strl")   // LIST chunk type: stream list
	writeStr("strh")   // Stream header
	writeInt(56)       // Length of the strh sub-chunk
	writeStr("vids")   // fccType - type of data stream - here 'vids' for video stream
	writeStr("MJPG")   // MJPG for Motion JPEG
	writeInt(0)        // dwFlags
	writeInt(0)        // wPriority, wLanguage
	writeInt(0)        // dwInitialFrames
	writeInt(1)        // dwScale
	writeInt(fps)      // dwRate, Frame rate for video streams (the actual FPS is calculated by dividing this by dwScale)
	writeInt(0)        // usually zero
	aw.framesCountFieldPos2 = aw.currentPos()
	writeInt(0)   // dwLength, playing time of AVI file as defined by scale and rate (set equal to the number of frames)
	writeInt(0)   // dwSuggestedBufferSize for reading the stream (typically, this contains a value corresponding to the largest chunk in a stream)
	writeInt(-1)  // dwQuality, encoding quality given by an integer between (0 and 10,000.  If set to -1, drivers use the default quality value)
	writeInt(0)   // dwSampleSize, 0 means that each frame is in its own chunk
	writeShort(0) // left of rcFrame if stream has a different size than dwWidth*dwHeight(unused)
	writeShort(0) //   ..top
	writeShort(0) //   ..right
	writeShort(0) //   ..bottom
	// end of 'strh' chunk, stream format follows
	writeStr("strf")             // stream format chunk
	writeLengthField()           // Chunk size (nesting level 3)
	writeInt(40)                 // biSize, write header size of BITMAPINFO header structure; applications should use this size to determine which BITMAPINFO header structure is being used, this size includes this biSize field
	writeInt(width)              // biWidth, width in pixels
	writeInt(height)             // biWidth, height in pixels (may be negative for uncompressed video to indicate vertical flip)
	writeShort(1)                // biPlanes, number of color planes in which the data is stored
	writeShort(24)               // biBitCount, number of bits per pixel #
	writeStr("MJPG")             // biCompression, type of compression used (uncompressed: NO_COMPRESSION=0)
	writeInt(width * height * 3) // biSizeImage (buffer size for decompressed mage) may be 0 for uncompressed data
	writeInt(0)                  // biXPelsPerMeter, horizontal resolution in pixels per meter
	writeInt(0)                  // biYPelsPerMeter, vertical resolution in pixels per meter
	writeInt(0)                  // biClrUsed (color table size; for 8-bit only)
	writeInt(0)                  // biClrImportant, specifies that the first x colors of the color table (0: all the colors are important, or, rather, their relative importance has not been computed)
	finalizeLengthField()        //'strf' chunk finished (nesting level 3)
	if aw.err != nil {
		return nil, aw.err
	}

	writeStr("strn") // Use 'strn' to provide a zero terminated text string describing the stream
	name := "Recorded with https://github.com/icza/mjpeg" +
		" at " + time.Now().Format("2006-01-02 15:04:05 MST")
	// Name must be 0-terminated and stream name length (the length of the chunk) must be even
	if len(name)&0x01 == 0 {
		name = name + " \000" // padding space plus terminating 0
	} else {
		name = name + "\000" // terminating 0
	}
	writeInt(int32(len(name))) // Length of the strn sub-CHUNK (must be even)
	writeStr(name)
	finalizeLengthField() // LIST 'strl' finished (nesting level 2)
	finalizeLengthField() // LIST 'hdrl' finished (nesting level 1)

	writeStr("LIST")      // The second LIST chunk, which contains the actual data
	aw.writeLengthField() // Chunk length (nesting level 1)
	aw.moviPos = aw.currentPos()
	writeStr("movi") // LIST chunk type: 'movi'

	if aw.err != nil {
		return nil, aw.err
	}

	return aw, nil
}

// writeStr writes a string to the file.
func (aw *aviWriter) writeStr(s string) {
	if aw.err != nil {
		return
	}
	_, aw.err = aw.avif.WriteString(s)
}

// writeInt writes a 32-bit int value to the file.
func (aw *aviWriter) writeInt32(n int32) {
	if aw.err != nil {
		return
	}
	binary.LittleEndian.PutUint32(aw.buf4, uint32(n))
	_, aw.err = aw.avif.Write(aw.buf4)
}

// writeIntToIdx writes a 32-bit int value to the index file.
func (aw *aviWriter) writeIntToIdx(n int) {
	if aw.err != nil {
		return
	}
	binary.LittleEndian.PutUint32(aw.buf4, uint32(n))
	_, aw.err = aw.idxf.Write(aw.buf4)
}

// writeShort writes a 16-bit int value to the index file.
func (aw *aviWriter) writeInt16(n int16) {
	if aw.err != nil {
		return
	}
	binary.LittleEndian.PutUint16(aw.buf2, uint16(n))
	_, aw.err = aw.avif.Write(aw.buf2)
}

// writeLengthField writes an empty int field to the avi file, and saves
// the current file position as it will be filled later.
func (aw *aviWriter) writeLengthField() {
	if aw.err != nil {
		return
	}
	pos := aw.currentPos()
	if aw.err != nil {
		return
	}
	aw.lengthFields = append(aw.lengthFields, pos)

	_, aw.err = aw.avif.Write(emptyInt)
}

/**
 * Finalizes the last length field.
 */
// finalizeLengthField finalizes the last length field.
func (aw *aviWriter) finalizeLengthField() {
	pos := aw.currentPos()

	_, aw.err = aw.avif.Seek(aw.lengthFields[len(aw.lengthFields)-1], 0)
	aw.lengthFields = aw.lengthFields[:len(aw.lengthFields)-1]
	if aw.err != nil {
		return
	}
	aw.writeInt32(int32(pos - 4))

	// Seek "back" but align to a 2-byte boundary
	if pos&0x01 != 0 {
		pos++
	}
	_, aw.err = aw.avif.Seek(pos, 0)
}

// currentPos returns the current file position of the AVI file.
func (aw *aviWriter) currentPos() (pos int64) {
	pos, aw.err = aw.avif.Seek(0, 1) // Seek relative to current pos
	return
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

// Close implements AviWriter.Close().
func (aw *aviWriter) Close() (err error) {
	// TODO
	if err = aw.avif.Close(); err != nil {
		return
	}
	if err = aw.idxf.Close(); err != nil {
		return
	}
	if err = os.Remove(aw.idxFile); err != nil {
		return
	}
	return nil
}
