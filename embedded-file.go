package supago

import (
	"fmt"
	"github.com/docker/docker/api/types/mount"
	"github.com/google/uuid"
	"github.com/train360-corp/supago/internal/utils"
	"os"
	"path/filepath"
)

type EmbeddedFile struct {
	Data []byte
	Name string
	Path string
}

func (f EmbeddedFile) Mount() (*mount.Mount, error) {

	uniqueFileName := fmt.Sprintf("%s-%s", uuid.New().String(), f.Name)
	localPath := filepath.Join(utils.GetTempDir(), uniqueFileName)

	if err := os.WriteFile(localPath, f.Data, 0o444); err != nil {
		return nil, fmt.Errorf("failed to write temp-file \"%s\": %v", f.Name, err)
	}

	return &mount.Mount{
		Type:     mount.TypeBind,
		Source:   localPath,
		Target:   f.Path,
		ReadOnly: true,
	}, nil
}
