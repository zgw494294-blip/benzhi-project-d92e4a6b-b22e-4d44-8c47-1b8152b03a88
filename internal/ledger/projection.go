package ledger

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func writeProjectionAtomic(directory, path string, projection Projection) error {
	payload, err := json.MarshalIndent(projection, "", "  ")
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".projection-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	keep := false
	defer func() {
		_ = temporary.Close()
		if !keep {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(0o640); err != nil {
		return err
	}
	if _, err := temporary.Write(payload); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	keep = true
	directoryFile, err := os.Open(directory)
	if err != nil {
		return err
	}
	defer directoryFile.Close()
	return directoryFile.Sync()
}

func (s *Store) compareOrRepairProjection() error {
	payload, err := os.ReadFile(s.projectionPath)
	if errors.Is(err, os.ErrNotExist) {
		return writeProjectionAtomic(s.directory, s.projectionPath, s.projection)
	}
	if err != nil {
		return fmt.Errorf("读取投影: %w", err)
	}
	var existing Projection
	if err := json.Unmarshal(payload, &existing); err != nil {
		return writeProjectionAtomic(s.directory, s.projectionPath, s.projection)
	}
	if existing.SchemaVersion != SchemaVersion || existing.LastSequence != s.projection.LastSequence || existing.LastDigest != s.projection.LastDigest {
		return writeProjectionAtomic(s.directory, s.projectionPath, s.projection)
	}
	canonicalExisting, _ := json.Marshal(existing)
	canonicalRecovered, _ := json.Marshal(s.projection)
	if !bytes.Equal(canonicalExisting, canonicalRecovered) {
		return writeProjectionAtomic(s.directory, s.projectionPath, s.projection)
	}
	return nil
}

func projectionFileName(directory string) string {
	return filepath.Join(directory, "projection.json")
}
