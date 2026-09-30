package store

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

func totalMemory() uint64 {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 4 << 30
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fs := strings.Fields(sc.Text())
		if len(fs) >= 2 && fs[0] == "MemTotal:" {
			kb, _ := strconv.ParseUint(fs[1], 10, 64)
			return kb * 1024
		}
	}
	return 4 << 30
}
