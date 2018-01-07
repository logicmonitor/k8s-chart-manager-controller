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
