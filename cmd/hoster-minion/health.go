package main

import (
	"bufio"
	"context"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/artpar/hoster/internal/core/minion"
	"github.com/docker/docker/client"
)

// pingCmd handles the "ping" command.
// It tests the connection to Docker and returns version info.
func pingCmd() error {
	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		outputError("ping", minion.ErrCodeConnectionFailed, "failed to create docker client: "+err.Error())
		return err
	}
	defer cli.Close()

	// Get Docker version info
	version, err := cli.ServerVersion(ctx)
	if err != nil {
		outputError("ping", minion.ErrCodeConnectionFailed, "failed to connect to docker: "+err.Error())
		return err
	}

	info := minion.PingInfo{
		DockerVersion: version.Version,
		APIVersion:    version.APIVersion,
		OS:            version.Os,
		Arch:          runtime.GOARCH,
	}
	outputSuccess(info)
	return nil
}

// systemInfoCmd handles the "system-info" command.
// It collects host-level CPU, memory, and disk metrics using /proc and syscall.
func systemInfoCmd() error {
	info := minion.SystemInfo{
		CPUCores: float64(runtime.NumCPU()),
	}

	// Read memory info from /proc/meminfo
	if memTotal, memAvail, err := readMemInfo(); err == nil {
		info.MemoryTotalMB = memTotal / 1024 // /proc/meminfo reports in kB
		info.MemoryUsedMB = (memTotal - memAvail) / 1024
	}

	// Read disk info for root filesystem
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err == nil {
		totalBytes := stat.Blocks * uint64(stat.Bsize)
		freeBytes := stat.Bavail * uint64(stat.Bsize)
		info.DiskTotalMB = int64(totalBytes / (1024 * 1024))
		info.DiskUsedMB = int64((totalBytes - freeBytes) / (1024 * 1024))
	}

	// Read CPU usage from /proc/stat (two samples 100ms apart)
	info.CPUUsedPct = readCPUPercent()

	outputSuccess(info)
	return nil
}

// readMemInfo reads MemTotal and MemAvailable from /proc/meminfo (values in kB).
func readMemInfo() (total, available int64, err error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var gotTotal, gotAvail bool
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			total = parseMemInfoValue(line)
			gotTotal = true
		} else if strings.HasPrefix(line, "MemAvailable:") {
			available = parseMemInfoValue(line)
			gotAvail = true
		}
		if gotTotal && gotAvail {
			break
		}
	}
	return total, available, scanner.Err()
}

// parseMemInfoValue parses a /proc/meminfo line like "MemTotal:       16384000 kB" → 16384000.
func parseMemInfoValue(line string) int64 {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return 0
	}
	val, _ := strconv.ParseInt(parts[1], 10, 64)
	return val
}

// readCPUPercent reads two /proc/stat samples 100ms apart and computes CPU usage %.
func readCPUPercent() float64 {
	idle1, total1 := readCPUSample()
	if total1 == 0 {
		return 0
	}

	// Sleep 100ms between samples for a meaningful delta
	// Use a simple busy-wait alternative since we want to keep imports minimal
	// Actually, time.Sleep is fine — it's stdlib
	sleepForSample()

	idle2, total2 := readCPUSample()
	if total2 == 0 {
		return 0
	}

	idleDelta := float64(idle2 - idle1)
	totalDelta := float64(total2 - total1)
	if totalDelta == 0 {
		return 0
	}

	return (1.0 - idleDelta/totalDelta) * 100.0
}

// sleepForSample pauses briefly between CPU samples.
func sleepForSample() {
	time.Sleep(100 * time.Millisecond)
}

// readCPUSample reads the first "cpu" line from /proc/stat and returns idle + total ticks.
func readCPUSample() (idle, total uint64) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			return 0, 0
		}
		// fields: cpu user nice system idle iowait irq softirq steal guest guest_nice
		for i := 1; i < len(fields); i++ {
			val, _ := strconv.ParseUint(fields[i], 10, 64)
			total += val
			if i == 4 { // idle is the 4th field (0-indexed field 4)
				idle = val
			}
		}
		return idle, total
	}
	return 0, 0
}
