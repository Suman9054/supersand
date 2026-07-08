package process

import (
	"fmt"
	"os"
)

func SetupFilesystem(UUid string) (bool, error) {
	rootfs := fmt.Sprintf("sandinternal/v1_supersand/%s_rootfs", UUid)
	if err := os.MkdirAll(rootfs, 0o755); err != nil {
		return false, err
	}
	folders := [8]string{"bin", "dev", "etc", "proc", "sys", "tmp", "usr", "var"}
	for _, fl := range folders {
		fls := fmt.Sprintf("%s/%s", rootfs, fl)
		if erro := os.MkdirAll(fls, 0o755); erro != nil {
			return false, erro
		}
	}
	return true, nil
}
