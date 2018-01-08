package utilities

import (
	"os"

	log "github.com/sirupsen/logrus"
)

// EnsureDirectory ensure that dir is a directory
func EnsureDirectory(dir string) error {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		log.Debugf("Creating directory %s", dir)
		err = os.MkdirAll(dir, 0744)
		if err != nil {
			return err
		}
	}
	return nil
}

// I64ToPI64 returns the passed int as a pointer
func I64ToPI64(i int64) *int64 {
	return &i
}
