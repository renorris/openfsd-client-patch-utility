package patchfile

import (
	"errors"
	"github.com/goccy/go-yaml"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type PatchFile struct {
	Name             string `yaml:"name"`
	ExpectedSum      string `yaml:"expected_sum"`
	ExpectedLocation string `yaml:"expected_location"`
	Type             string `yaml:"type"`

	Sections                   []Section                  `yaml:"sections"`
	SectionOverwritePatches    []SectionOverwritePatch    `yaml:"section_overwrite_patches"`
	SectionPaddedStringPatches []SectionPaddedStringPatch `yaml:"section_padded_string_patches"`
	DotnetUserstringPatches    []DotnetUserstringPatch    `yaml:"section_padded_string_patches"`
}

// Section defines a binary section like .text or .data.
type Section struct {
	// Name of the section e.g., .text
	Name string `yaml:"name"`

	// RawOffset of the section. This is the address of the byte in the raw binary file where the section begins.
	RawOffset int64 `yaml:"raw_offset"`

	// VirtualStart specifies the starting virtual address of the section
	VirtualStart int64 `yaml:"virtual_start"`
}

// SectionOverwritePatch overwrites some bytes at a given section address.
type SectionOverwritePatch struct {
	Name           string `yaml:"name"`
	Section        string `yaml:"section"`
	SectionAddress int64  `yaml:"section_address"`
	NewBytes       []byte `yaml:"new_bytes"`
}

// SectionPaddedStringPatch overwrites some bytes at a given section address, padding any unused bytes with NULL characters.
type SectionPaddedStringPatch struct {
	Name           string `yaml:"name"`
	Section        string `yaml:"section"`
	SectionAddress int64  `yaml:"section_address"`
	AvailableBytes int64  `yaml:"available_bytes"`
	NewString      string `yaml:"new_string"`
	Encoding       string `yaml:"encoding"`
}

// DotnetUserstringPatch overwrites .NET #US strings according to
// https://ecma-international.org/wp-content/uploads/ECMA-335_6th_edition_june_2012.pdf
// II.24.2.4 #US and #Blob heaps.
type DotnetUserstringPatch struct {
	Name           string `yaml:"name"`
	Section        string `yaml:"section"`
	SectionAddress int64  `yaml:"section_address"`
	NewString      string `yaml:"new_string"`
}

// VPilotConfigPatch patches an obfuscated vPilotConfig.xml file
type VPilotConfigPatch struct {
	NetworkStatusURL string   `yaml:"network_status_url"`
	CachedServerList []string `yaml:"cached_server_list"`
}

func UnmarshalPatchFile(file io.Reader) (patchFile *PatchFile, err error) {
	decoder := yaml.NewDecoder(file)
	patchFile = &PatchFile{}
	if err = decoder.Decode(patchFile); err != nil {
		return
	}

	if patchFile.ExpectedLocation, err = formatPath(patchFile.ExpectedLocation); err != nil {
		return
	}

	return
}

func formatPath(path string) (formattedPath string, err error) {
	// Replace $HOME_DIR placeholder with user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}
	formattedPath = strings.ReplaceAll(path, "$HOME_DIR", homeDir)

	return
}

var NoSectionFoundErr = errors.New("error: no section found")

func (f *PatchFile) GetSection(name string) (section *Section, err error) {
	for i := range f.Sections {
		if name == f.Sections[i].Name {
			section = &f.Sections[i]
			return
		}
	}
	err = NoSectionFoundErr
	return
}

func (f *PatchFile) OpenTargetFile() (file *os.File, err error) {
	if file, err = os.OpenFile(f.ExpectedLocation, os.O_RDWR, 0666); err != nil {
		return
	}
	return
}

func (f *PatchFile) GetTargetFileDirectory() (dir string) {
	dir = filepath.Dir(f.ExpectedLocation)
	return
}
