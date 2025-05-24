package patch

import (
	"github.com/renorris/openfsd-client-patch-utility/patchfile"
	"os"
)

type SectionOverwritePatch struct {
	patchFile *patchfile.PatchFile
	patch     *patchfile.SectionOverwritePatch
}

func NewSectionOverwritePatch(patchFile *patchfile.PatchFile, patch *patchfile.SectionOverwritePatch) *SectionOverwritePatch {
	return &SectionOverwritePatch{patchFile, patch}
}

func (p *SectionOverwritePatch) Run(file *os.File) (err error) {
	section, err := p.patchFile.GetSection(p.patch.Section)
	if err != nil {
		return
	}

	rawOffset := section.RawOffset + (p.patch.SectionAddress - section.VirtualStart)

	if _, err = file.Seek(rawOffset, 0); err != nil {
		return
	}

	if _, err = file.Write(p.patch.NewBytes); err != nil {
		return
	}

	return
}

func (p *SectionOverwritePatch) Name() string {
	return p.patch.Name
}
