package leodocker

import (
	"github.com/sirupsen/logrus"
	"leodocker/container"
	"os"
)

func Run(tty bool, cmd string) {

	parent := container.NewParentProcess(tty, cmd)
	if err := parent.Start(); err != nil {
		logrus.Error(err)
	}
	parent.Wait()
	os.Exit(-1)
}
