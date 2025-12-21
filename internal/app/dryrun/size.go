package dryrun

import "fmt"

// FormatSize converts a byte count to a human-readable string representation.
// The output uses appropriate units (B, KB, MB, GB) based on the size.
//
// Examples:
//   - 0 bytes -> "0 B"
//   - 256 bytes -> "256 B"
//   - 1024 bytes -> "1.0 KB"
//   - 1536 bytes -> "1.5 KB"
//   - 1048576 bytes -> "1.0 MB"
//
// Negative values are treated as 0 bytes.
func FormatSize(bytes int64) string {
	// Handle edge cases
	if bytes <= 0 {
		return "0 B"
	}

	const (
		kilobyte = 1024
		megabyte = kilobyte * 1024
		gigabyte = megabyte * 1024
	)

	switch {
	case bytes >= gigabyte:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(gigabyte))
	case bytes >= megabyte:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(megabyte))
	case bytes >= kilobyte:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(kilobyte))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
