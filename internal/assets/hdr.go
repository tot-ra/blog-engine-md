package assets

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

var (
	isoGainMapNamespace   = []byte("urn:iso:std:iso:ts:21496:-1\x00")
	adobeGainMapNamespace = []byte("http://ns.adobe.com/hdr-gain-map/1.0/")
	appleGainMapVersion   = []byte("HDRGainMapVersion")
)

// hasHDRGainMap reports whether a JPEG advertises an HDR gain map in its
// metadata. Detection is intentionally limited to APP segments so ordinary
// image pixels or comments containing the same words cannot disable processing.
func hasHDRGainMap(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	var signature [2]byte
	if _, err := io.ReadFull(file, signature[:]); err != nil {
		return false, err
	}
	if signature != [2]byte{0xff, 0xd8} {
		return false, nil
	}

	for {
		marker, err := nextJPEGMarker(file)
		if err != nil {
			if err == io.EOF {
				return false, nil
			}
			return false, err
		}

		// Start of Scan begins entropy-coded pixels. All metadata relevant to
		// gain-map signalling is stored in APP segments before this point.
		if marker == 0xda || marker == 0xd9 {
			return false, nil
		}
		if marker == 0x01 || marker >= 0xd0 && marker <= 0xd7 {
			continue
		}

		var lengthBytes [2]byte
		if _, err := io.ReadFull(file, lengthBytes[:]); err != nil {
			return false, err
		}
		segmentLength := int(binary.BigEndian.Uint16(lengthBytes[:]))
		if segmentLength < 2 {
			return false, fmt.Errorf("invalid JPEG segment length %d", segmentLength)
		}
		payloadLength := segmentLength - 2

		if marker != 0xe1 && marker != 0xe2 {
			if _, err := io.CopyN(io.Discard, file, int64(payloadLength)); err != nil {
				return false, err
			}
			continue
		}

		payload := make([]byte, payloadLength)
		if _, err := io.ReadFull(file, payload); err != nil {
			return false, err
		}
		if marker == 0xe2 && bytes.HasPrefix(payload, isoGainMapNamespace) {
			return true, nil
		}
		if marker == 0xe1 && hasXMPOrAppleGainMapMetadata(payload) {
			return true, nil
		}
	}
}

func nextJPEGMarker(reader io.Reader) (byte, error) {
	var current [1]byte
	for {
		if _, err := io.ReadFull(reader, current[:]); err != nil {
			return 0, err
		}
		if current[0] != 0xff {
			continue
		}

		for {
			if _, err := io.ReadFull(reader, current[:]); err != nil {
				return 0, err
			}
			if current[0] == 0xff {
				continue
			}
			if current[0] != 0x00 {
				return current[0], nil
			}
			break
		}
	}
}

func hasXMPOrAppleGainMapMetadata(payload []byte) bool {
	if bytes.Contains(payload, adobeGainMapNamespace) && bytes.Contains(payload, []byte("hdrgm:Version")) {
		return true
	}
	// Apple may store headroom in EXIF rather than XMP, so the gain-map version
	// attribute alone is the stable XMP signal used by the reference decoder.
	return bytes.Contains(payload, appleGainMapVersion)
}
