package patch

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/renorris/openfsd-client-patch-utility/patchfile"
	"golang.org/x/text/encoding/unicode"
	"io"
	"os"
	"unicode/utf16"
)

// Adapted from https://github.com/renorris/vpilot-patch-utility/blob/main/pe/userstring

type DotnetUserstringPatch struct {
	patchFile *patchfile.PatchFile
	patch     *patchfile.DotnetUserstringPatch
}

func NewDotnetUserstringPatch(patchFile *patchfile.PatchFile, patch *patchfile.DotnetUserstringPatch) *DotnetUserstringPatch {
	return &DotnetUserstringPatch{patchFile, patch}
}

func (p *DotnetUserstringPatch) Run(file *os.File) (err error) {
	// Verify new string length does not exceed original
	existingStr, err := p.readString(file)
	if err != nil {
		return
	}
	if len(p.patch.NewString) > len(existingStr) {
		err = fmt.Errorf("new string cannot exceed available bytes (%d > %d)", len(p.patch.NewString), len(existingStr))
		return
	}

	// Write the new string
	if err = p.writeString(file, p.patch.NewString); err != nil {
		return
	}

	return
}

// readString reads a UTF-16 string from the #US heap at the specified
// file offset, then returns the UTF-8 representation of that string.
func (p *DotnetUserstringPatch) readString(file *os.File) (str string, err error) {
	section, err := p.patchFile.GetSection(p.patch.Section)
	if err != nil {
		return
	}
	rawOffset := section.RawOffset + (p.patch.SectionAddress - section.VirtualStart)

	if _, err = file.Seek(rawOffset, io.SeekStart); err != nil {
		return
	}

	lengthHeader := make([]byte, 4)
	if _, err = io.ReadFull(file, lengthHeader); err != nil {
		return
	}

	var dataLength, headerSize int
	if dataLength, headerSize, err = p.decodeLength([4]byte(lengthHeader)); err != nil {
		return
	}

	if dataLength%2 != 1 {
		err = errors.New("user string data length should be odd")
		return
	}

	// Seek to the byte right after the length header
	if _, err = file.Seek(rawOffset+int64(headerSize), io.SeekStart); err != nil {
		return
	}

	strData := make([]byte, dataLength)
	if _, err = io.ReadFull(file, strData); err != nil {
		return
	}

	// The last byte is a terminal byte. Ignore it.
	utf16Str := make([]uint16, (len(strData)-1)/2)
	for i := 0; i < len(strData)-1; i += 2 {
		utf16Str[i/2] = binary.LittleEndian.Uint16(strData[i : i+2])
	}

	// Decode UTF-16 to runes
	runes := utf16.Decode(utf16Str)

	// Convert runes to UTF-8 string
	utf8String := string(runes)

	str = utf8String
	return
}

// writeString writes a string on the #US heap at the specified
// file offset using the provided UTF-8 encoded string `str`.
func (p *DotnetUserstringPatch) writeString(file *os.File, str string) (err error) {
	// Convert UTF-8 characters into UTF-16 string
	encoder := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder()

	var utf16Bytes []byte
	if utf16Bytes, err = encoder.Bytes([]byte(str)); err != nil {
		return
	}

	// Check if the last byte needs to be set:
	//
	// https://ecma-international.org/wp-content/uploads/ECMA-335_6th_edition_june_2012.pdf
	// II.24.2.4 #US and #Blob heaps
	//
	// "Strings in the #US (user string) heap are encoded using 16-bit Unicode encodings. The count on each
	// string is the number of bytes (not characters) in the string. Furthermore, there is an additional terminal
	// byte (so all byte counts are odd, not even). This final byte holds the value 1 if and only if any UTF16
	// character within the string has any bit set in its top byte, or its low byte is any of the following: 0x01–
	// 0x08, 0x0E–0x1F, 0x27, 0x2D, 0x7F. Otherwise, it holds 0. The 1 signifies Unicode characters that
	// require handling beyond that normally provided for 8-bit encoding sets."

	setTerminalBit := false
	for i := 1; i < len(utf16Bytes); i += 2 {
		if utf16Bytes[i] > 0 {
			setTerminalBit = true
			break
		}
		switch utf16Bytes[i-1] {
		case 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
			0x0E, 0x0F, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15,
			0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D,
			0x1E, 0x1F, 0x27, 0x2D, 0x7F:
			setTerminalBit = true
			break
		}
	}

	if setTerminalBit {
		utf16Bytes = append(utf16Bytes, []byte{0x01}...)
	} else {
		utf16Bytes = append(utf16Bytes, []byte{0x00}...)
	}

	// Encode the length of utf16Bytes
	var header []byte
	if header, err = p.encodeLength(len(utf16Bytes)); err != nil {
		return
	}

	// Get raw offset
	section, err := p.patchFile.GetSection(p.patch.Section)
	if err != nil {
		return
	}
	rawOffset := section.RawOffset + (p.patch.SectionAddress - section.VirtualStart)

	// Seek to the file offset
	if _, err = file.Seek(rawOffset, io.SeekStart); err != nil {
		return
	}

	// Write the header
	if _, err = io.Copy(file, bytes.NewReader(header)); err != nil {
		return err
	}

	// Write the utf16 string bytes
	if _, err = io.Copy(file, bytes.NewReader(utf16Bytes)); err != nil {
		return err
	}

	return nil
}

// decodeLength decodes the length of a #US or #Blob string.
// https://ecma-international.org/wp-content/uploads/ECMA-335_6th_edition_june_2012.pdf
// II.24.2.4 #US and #Blob heaps
func (p *DotnetUserstringPatch) decodeLength(header [4]byte) (length int, headerSize int, err error) {
	if header[0]>>7 == 0 {
		// Length is the 7 LSBs of header[0]
		length = int(header[0])
		headerSize = 1
		return
	}

	if header[0]>>6 == 0b10 {
		// Length is (header[0] bbbbbb2 << 8 + header[1])
		length = (int(header[0]&0b00111111) << 8) + int(header[1])
		headerSize = 2
		return
	}

	if header[0]>>5 == 0b110 {
		// Length is (header[0] bbbbb2 << 24 + header[1] << 16 + header[2] << 8 + header[3])
		length = (int(header[0]&0b00011111) << 24) +
			(int(header[1]) << 16) +
			(int(header[2]) << 8) +
			(int(header[3]))
		headerSize = 4
		return
	}

	length = -1
	err = errors.New("invalid length header")
	return
}

// encodeLength encodes the length of a #US or #Blob string.
// https://ecma-international.org/wp-content/uploads/ECMA-335_6th_edition_june_2012.pdf
// II.24.2.4 #US and #Blob heaps
func (p *DotnetUserstringPatch) encodeLength(length int) (header []byte, err error) {
	if length < 0 {
		err = errors.New("cannot encode negative length")
		return
	}

	// If the length fits into 7 bits, encode the header as a single byte
	// Encode a single byte if the length is < 128
	if length < 128 {
		header = []byte{byte(length)}
		return
	}

	// If the length fits into 14 bits, encode into 2 bytes
	// Set 0b10 header for the first byte to indicate that we're
	// using 2 bytes
	if length < 16384 {
		header = []byte{0b10000000 | byte(length>>8), byte(length)}
		return
	}

	// Max length is 2^29 bits
	// If the length fits into 29 bits, encode into 4 bytes
	// Set 0b110 header to indicate that we're using 4 bytes
	if length < 536870912 {
		header = []byte{0b11000000 | byte(length>>24), byte(length >> 16), byte(length >> 8), byte(length)}
		return
	}

	length = -1
	err = errors.New("cannot encode length at or above 536870912 (29 bits max)")
	return
}

func (p *DotnetUserstringPatch) Name() string {
	return p.patch.Name
}
