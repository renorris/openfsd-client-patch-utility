package main

import (
	"bufio"
	"context"
	"crypto/sha1"
	"embed"
	"encoding/hex"
	"fmt"
	"github.com/renorris/openfsd-client-patch-utility/patch"
	"github.com/renorris/openfsd-client-patch-utility/patchfile"
	"io"
	"os"
	"path/filepath"
	"strings"
)

//go:embed enabled_patchfiles
var enabledPatchfiles embed.FS

func runFlow(ctx context.Context) {
	defer bufio.NewReader(os.Stdin).ReadString('\n')

	patchFile, err := selectPatchfile()
	if err != nil {
		fmt.Println("error selecting patchfile")
		fmt.Println(err)
		return
	}

	targetFile, err := patchFile.OpenTargetFile()
	if err != nil {
		fmt.Printf("error opening target file: %s\n", err.Error())
		return
	}
	defer targetFile.Close()

	var ok bool
	if ok, err = verifyChecksum(targetFile, patchFile.ExpectedSum); err != nil {
		fmt.Printf("error validating checksum: %s\n", err.Error())
		return
	} else if !ok {
		if err = restoreBackup(targetFile); err != nil {
			fmt.Printf("error restoring backup: %s\n\nPlease reinstall your openfsd client.", err.Error())
			return
		}

		// Restore backups for secondary files
		for _, fileName := range patchFile.MakeBackupsFor {
			var file *os.File
			if file, err = os.Open(fileName); err != nil {
				return
			}
			if err = restoreBackup(file); err != nil {
				fmt.Printf("error restoring backup for secondary file: %s\n", err.Error())
				return
			}
			file.Close()
		}

		fmt.Println("Reverted patches.")
		return
	}

	// Make backups for secondary files
	for _, fileName := range patchFile.MakeBackupsFor {
		var file *os.File
		if file, err = os.Open(fileName); err != nil {
			return
		}
		if err = makeBackup(file); err != nil {
			fmt.Printf("error making backup: %s\n", err.Error())
			return
		}
		file.Close()
	}

	if err = makeBackup(targetFile); err != nil {
		fmt.Printf("error making backup: %s\n", err.Error())
		return
	}

	patches, err := extractPatches(patchFile)
	if err != nil {
		fmt.Printf("error extracting patches: %s\n", err.Error())
		return
	}

	fmt.Println("Executing patches...")
	for i, p := range patches {
		fmt.Printf("%d - %s... ", i+1, p.Name())
		if err = p.Run(targetFile); err != nil {
			fmt.Printf("failed: %s\n", err.Error())
			return
		}
		fmt.Printf("done\n")
	}

	fmt.Println("\nApplied all patches. Run this program again to revert.")
}

func selectPatchfile() (selected *patchfile.PatchFile, err error) {
	files, err := loadPatchfiles(enabledPatchfiles)
	if err != nil {
		fmt.Println("Error loading patchfiles: " + err.Error())
		return
	}

	fmt.Print("Select a patch:\n\n")
	for i, file := range files {
		fmt.Printf("[%d] %s\n", i, file.Name)
	}
	fmt.Print("\n> ")

	var selection int
	if _, err = fmt.Scanf("%d", &selection); err != nil {
		return
	}

	if selection > len(files)-1 {
		err = fmt.Errorf("invalid selection (out-of-range): %d", selection)
		return
	}

	fmt.Printf("Selected %s\n\n", files[selection].Name)

	selected = files[selection]
	return
}

func loadPatchfiles(fs embed.FS) (files []*patchfile.PatchFile, err error) {
	const pathPrefix = "enabled_patchfiles"

	entries, err := fs.ReadDir(pathPrefix)
	if err != nil {
		return
	}

	for i := range entries {
		entry := entries[i]

		if !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		fullPath := filepath.Join(pathPrefix, entry.Name())

		var rawPatchfile *os.File
		if rawPatchfile, err = os.Open(fullPath); err != nil {
			return
		}

		var patchFile *patchfile.PatchFile
		if patchFile, err = patchfile.UnmarshalPatchFile(rawPatchfile); err != nil {
			return
		}

		files = append(files, patchFile)
	}

	return
}

func extractPatches(patchFile *patchfile.PatchFile) (patches []patch.Patch, err error) {
	for _, p := range patchFile.SectionOverwritePatches {
		patches = append(patches, patch.NewSectionOverwritePatch(patchFile, &p))
	}
	for _, p := range patchFile.SectionPaddedStringPatches {
		patches = append(patches, patch.NewSectionPaddedStringPatch(patchFile, &p))
	}
	for _, p := range patchFile.DotnetUserstringPatches {
		patches = append(patches, patch.NewDotnetUserstringPatch(patchFile, &p))
	}
	if patchFile.VPilotConfigPatch != nil {
		patches = append(patches, patch.NewVPilotConfigPatch(patchFile, patchFile.VPilotConfigPatch))
	}

	return
}

func verifyChecksum(file *os.File, checksum string) (ok bool, err error) {
	hasher := sha1.New()

	if _, err = file.Seek(0, 0); err != nil {
		return
	}

	if _, err = io.Copy(hasher, file); err != nil {
		return
	}

	sum := hasher.Sum(nil)
	if hex.EncodeToString(sum) != checksum {
		return
	}

	ok = true
	return
}

// makeBackup makes a backup of the provided file in the same directory.
func makeBackup(file *os.File) (err error) {
	backupFile, err := os.Create(file.Name() + ".orig")
	if err != nil {
		return
	}
	defer backupFile.Close()

	if _, err = file.Seek(0, 0); err != nil {
		return
	}

	if _, err = io.Copy(backupFile, file); err != nil {
		return
	}

	return
}

// restoreBackup restores a backup for a given patched file
func restoreBackup(file *os.File) (err error) {
	backupFile, err := os.Open(file.Name() + ".orig")
	if err != nil {
		return
	}
	defer backupFile.Close()

	if err = file.Truncate(0); err != nil {
		return
	}

	if _, err = file.Seek(0, 0); err != nil {
		return
	}

	if _, err = io.Copy(file, backupFile); err != nil {
		return
	}

	if err = backupFile.Close(); err != nil {
		return
	}

	if err = os.Remove(file.Name() + ".orig"); err != nil {
		return
	}

	return
}
