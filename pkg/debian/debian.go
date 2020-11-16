// Package debian implements the vks.PackageManager interface for Debian like operating systems.
package debian

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type Debian struct{}

// Installed implemetns the vks.PackageManager interface.
func (d *Debian) Install(name, _ string) error {
	cmd := exec.Command("apt-get", "-qq", "--force-yes", "install", name)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	return nil
}

func (d *Debian) UnitFile(name string) (string, error) {
	cmd := exec.Command("dpkg", "-L", name)
	buf, err := cmd.Output()
	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(bytes.NewReader(buf))
	for scanner.Scan() {
		if !strings.HasPrefix(scanner.Text(), "/lib/systemd/system/") {
			continue
		}
		if strings.HasSuffix(scanner.Text(), ".service") {
			return scanner.Text(), nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("no unit found")
}
