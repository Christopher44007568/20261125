package eyebw

import "fmt"

const (
	Byte     uint64 = 1
	KibiByte        = 1024 * Byte
	MebiByte        = 1024 * KibiByte
	GibiByte        = 1024 * MebiByte
)

// FormatBytes uses binary units because GPU/memory budgets are normally MiB/GiB.
func FormatBytes(n uint64) string {
	switch {
	case n >= GibiByte:
		return fmt.Sprintf("%.2f GiB", float64(n)/float64(GibiByte))
	case n >= MebiByte:
		return fmt.Sprintf("%.2f MiB", float64(n)/float64(MebiByte))
	case n >= KibiByte:
		return fmt.Sprintf("%.2f KiB", float64(n)/float64(KibiByte))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

func clampUint64(v int) uint64 {
	if v < 0 {
		return 0
	}
	return uint64(v)
}

func mulSize(a, b int) uint64 {
	if a <= 0 || b <= 0 {
		return 0
	}
	return uint64(a) * uint64(b)
}
