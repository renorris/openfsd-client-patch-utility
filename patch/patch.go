package patch

import "os"

type Patch interface {
	Name() string

	// Run performs the patch on a given file.
	Run(file *os.File) (err error)
}
