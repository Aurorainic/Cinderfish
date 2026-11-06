//go:build !linux

package httpapi

// The server-info endpoint reports what the platform exposes. Only Linux reads
// /proc and statfs here; other platforms return the zero values and the widget
// shows "未知" instead of inventing numbers.
func readCPUStat() (total, idle uint64, ok bool) { return 0, 0, false }

func readMemInfo() (total, available uint64, ok bool) { return 0, 0, false }

func readCPUModel() string { return "" }

func readLoadAvg() (one, five, fifteen float64, ok bool) { return 0, 0, 0, false }

func readFSStat(path string) (total, used uint64, ok bool) { return 0, 0, false }
