package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Availability struct {
	HasFirejail bool
	HasBwrap    bool
}

func Detect() Availability {
	_, errFirejail := exec.LookPath("firejail")
	_, errBwrap := exec.LookPath("bwrap")
	return Availability{
		HasFirejail: errFirejail == nil,
		HasBwrap:    errBwrap == nil,
	}
}

func GenerateExec(originalExec, binaryPath, tool string, allowNetwork bool) string {
	if originalExec == "" {
		originalExec = binaryPath + " %U"
	}

	switch strings.ToLower(tool) {
	case "firejail":
		networkFlag := ""
		if !allowNetwork {
			networkFlag = "--net=none "
		}
		return fmt.Sprintf("firejail %s--noprofile \"%s\" %%U", networkFlag, binaryPath)

	case "bwrap":
		home, _ := os.UserHomeDir()
		networkFlag := ""
		if !allowNetwork {
			networkFlag = "--unshare-net "
		}
		return fmt.Sprintf("bwrap %s--ro-bind / / --dev /dev --proc /proc --bind \"%s\" \"%s\" \"%s\" %%U",
			networkFlag, home, home, binaryPath)

	default:
		return originalExec
	}
}
