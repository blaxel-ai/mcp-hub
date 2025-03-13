package build

import (
	"os"

	"github.com/beamlit/mcp-hub/internal/catalog"
	"github.com/beamlit/mcp-hub/internal/errors"
)

func (b *Build) Clean() {
	os.RemoveAll(tmpDir)
	errors.HandleError("create temp directory", os.MkdirAll(tmpDir, 0755))
	os.RemoveAll(catalog.CatalogDir)
	errors.HandleError("create catalog directory", os.MkdirAll(catalog.CatalogDir, 0755))
}
