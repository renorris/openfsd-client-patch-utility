package patch

import (
	"fmt"
	"github.com/renorris/openfsd-client-patch-utility/patchfile"
	"os"
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

	if int64(len(p.patch.NewString)) > p.patch.TotalLength {
		err = fmt.Errorf("new string cannot exceed total length (%d > %d)", len(p.patch.NewString), p.patch.TotalLength)
		return
	}

	rawOffset := section.RawOffset + (p.patch.SectionAddress - section.VirtualStart)

	if _, err = file.Seek(rawOffset, 0); err != nil {
		return
	}

	if _, err = file.WriteString(p.patch.NewString); err != nil {
		return
	}

	zeroesToPad := p.patch.TotalLength - int64(len(p.patch.NewString))
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
