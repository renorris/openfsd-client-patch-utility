package patch

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"github.com/renorris/openfsd-client-patch-utility/patchfile"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

type VPilotConfigPatch struct {
	patchFile *patchfile.PatchFile
	patch     *patchfile.VPilotConfigPatch
}

func NewVPilotConfigPatch(patchFile *patchfile.PatchFile, patch *patchfile.VPilotConfigPatch) *VPilotConfigPatch {
	return &VPilotConfigPatch{patchFile, patch}
}

func (p *VPilotConfigPatch) Run(_ *os.File) (err error) {
	configFilePath := filepath.Join(p.patchFile.GetTargetFileDirectory(), "vPilotConfig.xml")
	file, err := os.Open(configFilePath)
	if err != nil {
		return
	}
	defer file.Close()

	obfuscatedNetworkStatusURL, err := p.obfuscateFieldToBase64([]byte(p.patch.NetworkStatusURL), vPilotConfigObfuscatorKey)
	if err != nil {
		return
	}

	var obfuscatedCachedServerList []string
	for _, cachedServer := range p.patch.CachedServerList {
		var obfuscatedCachedServer []byte
		if obfuscatedCachedServer, err = p.obfuscateFieldToBase64(
			[]byte(cachedServer),
			vPilotConfigObfuscatorKey,
		); err != nil {
			return
		}
		obfuscatedCachedServerList = append(obfuscatedCachedServerList, string(obfuscatedCachedServer))
	}

	var fileContents []byte
	if _, err = io.ReadFull(file, fileContents); err != nil {
		return
	}

	newFileContents, err := p.replaceXmlFields(fileContents, string(obfuscatedNetworkStatusURL), obfuscatedCachedServerList)
	if err != nil {
		return
	}

	// Erase file
	if _, err = file.Seek(0, 0); err != nil {
		return
	}
	if err = file.Truncate(0); err != nil {
		return
	}

	// Re-write
	if _, err = io.Copy(file, bytes.NewReader(newFileContents)); err != nil {
		return
	}

	return
}

var vPilotConfigObfuscatorKey = generatevPilotConfigObfuscatorKey()

func generatevPilotConfigObfuscatorKey() []byte {
	const guid = "5575ac09-f2de-4a1e-808b-e3398e17f8bf"
	sum := md5.Sum([]byte(guid))

	key := make([]byte, 24)
	copy(key[:16], sum[:])
	copy(key[16:], sum[:8])
	return key
}

func (p *VPilotConfigPatch) encrypt(plaintext []byte, key []byte) (ciphertext []byte, err error) {
	return tripleDESEncrypt(plaintext, key)
}

func (p *VPilotConfigPatch) decrypt(ciphertext []byte, key []byte) (plaintext []byte, err error) {
	return tripleDESDecrypt(ciphertext, key)
}

func (p *VPilotConfigPatch) obfuscateFieldToBase64(plaintext []byte, key []byte) (ciphertextBase64 []byte, err error) {
	var ciphertext []byte
	if ciphertext, err = p.encrypt(plaintext, key); err != nil {
		return
	}

	ciphertextBase64 = []byte(base64.StdEncoding.EncodeToString(ciphertext))
	return
}

func (p *VPilotConfigPatch) deobfuscateFieldFromBase64(ciphertextBase64 []byte, key []byte) (plaintext []byte, err error) {
	var ciphertext []byte
	if ciphertext, err = base64.StdEncoding.DecodeString(string(ciphertextBase64)); err != nil {
		return
	}

	if plaintext, err = p.decrypt(ciphertext, key); err != nil {
		return
	}

	return
}

// replaceXmlFields replaces the network status and cached server fields.
// It clears the NetworkLogin and NetworkPassword fields.
// The passed fields must be properly obfuscated.
func (p *VPilotConfigPatch) replaceXmlFields(xmlFileData []byte, networkStatusURL string, cachedServers []string) ([]byte, error) {
	// Prepare regex patterns to match the relevant XML elements
	networkStatusURLPattern := regexp.MustCompile(`<NetworkStatusURL>[\s\S]*?</NetworkStatusURL>`)
	cachedServersPattern := regexp.MustCompile(`<CachedServers>[\s\S]*?</CachedServers>`)

	// Replace NetworkStatusURL with the new value
	newNetworkStatusURL := fmt.Sprintf("<NetworkStatusURL>%s</NetworkStatusURL>", networkStatusURL)
	xmlFileData = networkStatusURLPattern.ReplaceAll(xmlFileData, []byte(newNetworkStatusURL))

	// Construct the new CachedServers XML
	cachedServersXML := "<CachedServers>\n"
	for _, server := range cachedServers {
		cachedServersXML += fmt.Sprintf("    <string>%s</string>\n", server)
	}
	cachedServersXML += "  </CachedServers>"

	// Replace CachedServers with the new value
	xmlFileData = cachedServersPattern.ReplaceAll(xmlFileData, []byte(cachedServersXML))

	// Clear CID and password fields
	networkLoginPattern := regexp.MustCompile(`<NetworkLogin>[\s\S]*?</NetworkLogin>`)
	networkPasswordPattern := regexp.MustCompile(`<NetworkPassword>[\s\S]*?</NetworkPassword>`)

	xmlFileData = networkLoginPattern.ReplaceAll(xmlFileData, []byte("<NetworkLogin></NetworkLogin>"))
	xmlFileData = networkPasswordPattern.ReplaceAll(xmlFileData, []byte("<NetworkPassword></NetworkPassword>"))

	return xmlFileData, nil
}

func (p *VPilotConfigPatch) Name() string {
	return "vPilot Config Patch"
}
