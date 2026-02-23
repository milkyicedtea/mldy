package main

import (
	"os"
	"strings"
)

func detectLinuxDistro() (id string, idLike string, err error) {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "", "", err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "ID=") {
			id = strings.Trim(strings.TrimPrefix(line, "ID="), `"`)
		}
		if strings.HasPrefix(line, "ID_LIKE=") {
			idLike = strings.Trim(strings.TrimPrefix(line, "ID_LIKE="), `"`)
		}
	}

	return id, idLike, nil
}

func runPackageManager(prefix []string, packageManager string, args ...string) error {
	cmd := packageManager
	cmdArgs := args

	if len(prefix) > 0 {
		cmd = prefix[0]
		cmdArgs = append(prefix[1:], packageManager)
		cmdArgs = append(cmdArgs, cmdArgs...)
	}

	return run(cmd, cmdArgs...)
}
