package patch

import (
	"encoding/binary"
	"fmt"
	"github.com/renorris/openfsd-client-patch-utility/patchfile"
	"os"
	"unicode/utf16"
)

type SectionPaddedStringPatch struct {
	patchFile *patchfile.PatchFile
	patch     *patchfile.SectionPaddedStringPatch
}

func NewSectionPaddedStringPatch(patchFile *patchfile.PatchFile, patch *patchfile.SectionPaddedStringPatch) *SectionPaddedStringPatch {
	return &SectionPaddedStringPatch{patchFile, patch}
}

func (p *SectionPaddedStringPatch) Run(file *os.File) (err error) {
	section, err := p.patchFile.GetSection(p.patch.Section)
	if err != nil {
		return
	}

	var strVal string
	switch p.patch.Encoding {
	case "utf8":
		strVal = p.patch.NewString
	case "utf16le":
		strVal = string(encodeUTF16LE(p.patch.NewString))
	default:
		err = fmt.Errorf("unknown encoding: %s", p.patch.Encoding)
		return
	}

	if int64(len(strVal)) > p.patch.AvailableBytes {
		err = fmt.Errorf("new string cannot exceed available bytes (%d > %d)", len(p.patch.NewString), p.patch.AvailableBytes)
		return
	}

	rawOffset := section.RawOffset + (p.patch.SectionAddress - section.VirtualStart)

	if _, err = file.Seek(rawOffset, 0); err != nil {
		return
	}

	if _, err = file.WriteString(strVal); err != nil {
		return
	}

	zeroesToPad := p.patch.AvailableBytes - int64(len(strVal))
	var zeroes []byte
	for range zeroesToPad {
		zeroes = append(zeroes, 0x00)
	}

	if _, err = file.Write(zeroes); err != nil {
		return
	}

	return
}

func (p *SectionPaddedStringPatch) Name() string {
	return p.patch.Name
}

func encodeUTF16LE(s string) []byte {
	runes := []rune(s)
	utf16Values := utf16.Encode(runes)

	buf := make([]byte, len(utf16Values)*2)
	for i, r := range utf16Values {
		binary.LittleEndian.PutUint16(buf[i*2:], r)
	}
	buf = append(buf, 0x00, 0x00) // Append null terminator

	return buf
}
