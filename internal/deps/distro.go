package deps

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
	if len(prefix) > 0 {
		full := append([]string{}, prefix...)
		full = append(full, packageManager)
		full = append(full, args...)
		return run(full[0], full[1:]...)
	}
	return run(packageManager, args...)
}
