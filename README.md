# openfsd client patch utility

Patch utility for flight simulation clients enabling custom FSD server compatibility.

Currently provided patchfiles:

| Client          | Sum                                                           | 
|-----------------|---------------------------------------------------------------|
| Euroscope 3.2.9 | Euroscope.exe SHA1 `dfb1caf3d73e897b2a04964dc35867b4059bc537` |
| vatSys 1.4.19   | vatSys.exe SHA1 = `b8050748bc436ce4b870d95a37a2eeb20532ab30`  |
| xPilot 3.0.1    | xPilot.exe SHA1 `1ae61e1d4a624751124a49cd992c90f948c31d37`    |
| vPilot 3.11.1   | vPilot.exe SHA1 `19edcce42b0f9dddc0de0c5cf5c79ed1c7018728`    |

## Features:

- **Arbitrary contiguous binary patching**: Modify specific bytes in the client executable at specific virtual section offsets.
- **Modify existing strings in binary sections with padding**: Update strings, such as URLs, and pad them with zeroes.
- **Modify CIL #US strings in compiled .NET binaries**

## Configuration:

To configure the patch, copy the desiired YAML patch files from the `example_patchfiles` directory into `enabled_patchfiles`.

## Usage:

To use this utility, you need to install the [Go Programming Language](https://go.dev/dl/). Follow these steps to build and apply patches:

1. **Clone the repository** to your local machine.
2. **Prepare patch files**: Copy desired patch files from `example_patchfiles` to `enabled_patchfiles`. Modify these files to match your custom FSD server’s settings (e.g., update URLs, string lengths, etc.)
3. **Build the utility**:
    - On Windows:
      ```
      go build -o openfsd-patch.exe .
      ```
    - On UNIX (to build a Windows executable):
      ```
      GOOS=windows GOARCH=amd64 go build -o openfsd-patch.exe .
      ```
    All patches placed into the `enabled_patches` directory will automatically be embedded into the patch.exe file.

4. **Select a patch**: The program lists available patches from the `enabled_patchfiles` directory. Enter the number corresponding to the desired patch.
5. **Apply patches**: The utility verifies the target file’s checksum, creates a backup, and applies the patches.
6. **Revert patches**: To undo changes, run the executable again. If the target file's checksum indicates it has been patched, the utility will restore the original file from the backup.

## Clients

Wiki entires:

- [Euroscope](https://github.com/renorris/openfsd-client-patch-utility/wiki/Euroscope)
- [vatSys](https://github.com/renorris/openfsd-client-patch-utility/wiki/vatSys)
- [xPilot](https://github.com/renorris/openfsd-client-patch-utility/wiki/xPilot)
- [vPilot](https://github.com/renorris/openfsd-client-patch-utility/wiki/vPilot)
