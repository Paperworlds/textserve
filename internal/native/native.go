package native

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/paperworlds/textserve/internal/docker"
	"github.com/paperworlds/textserve/internal/health"
	"github.com/paperworlds/textserve/internal/registry"
)

// Start launches the native server in the background and writes its PID to cfg.PidFile.
func Start(name string, cfg *registry.ServerConfig) error {
	envVars, err := docker.ResolveEnv(name, cfg)
	if err != nil {
		return fmt.Errorf("resolve env for %s: %w", name, err)
	}

	// Auto-sync venv if native_venv is set but missing.
	if cfg.NativeVenv != "" {
		venv := os.ExpandEnv(cfg.NativeVenv)
		if _, statErr := os.Stat(venv); os.IsNotExist(statErr) {
			serverDir := filepath.Dir(venv)
			fmt.Printf("venv missing — running uv sync in %s\n", serverDir)
			syncCmd := exec.Command("uv", "sync")
			syncCmd.Dir = serverDir
			syncCmd.Stdout = os.Stdout
			syncCmd.Stderr = os.Stderr
			if syncErr := syncCmd.Run(); syncErr != nil {
				return fmt.Errorf("uv sync for %s: %w", name, syncErr)
			}
		}
	}

	// Build PATH: prepend <native_venv>/bin if set.
	env := os.Environ()
	if cfg.NativeVenv != "" {
		venv := os.ExpandEnv(cfg.NativeVenv)
		venvBin := filepath.Join(venv, "bin")
		for i, kv := range env {
			if strings.HasPrefix(kv, "PATH=") {
				env[i] = "PATH=" + venvBin + string(os.PathListSeparator) + kv[5:]
				break
			}
		}
	}
	// Overlay resolved env vars.
	env = append(env, envVars...)

	// Expand ${HOME}/os vars in native_args.
	args := make([]string, len(cfg.NativeArgs))
	for i, a := range cfg.NativeArgs {
		args[i] = os.ExpandEnv(a)
	}

	// If a venv is configured, resolve the command against the venv's bin dir
	// so the venv Python (not the system Python) is used. exec.Command resolves
	// executables using the parent's PATH, not cmd.Env, so we must do this explicitly.
	nativeCmd := cfg.NativeCmd
	if cfg.NativeVenv != "" {
		venvBin := filepath.Join(os.ExpandEnv(cfg.NativeVenv), "bin")
		candidate := filepath.Join(venvBin, cfg.NativeCmd)
		if _, err := os.Stat(candidate); err == nil {
			nativeCmd = candidate
		}
	}
	logFile, logErr := openLogFile(name)
	if logErr != nil {
		logFile = os.Stderr
	}

	cmd := exec.Command(nativeCmd, args...)
	cmd.Env = env
	cmd.Stdin = nil
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", name, err)
	}

	if cfg.PidFile != "" {
		pidData := strconv.Itoa(cmd.Process.Pid) + "\n"
		if err := os.WriteFile(cfg.PidFile, []byte(pidData), 0o644); err != nil {
			return fmt.Errorf("write pid file: %w", err)
		}
	}

	return nil
}

// Stop sends SIGTERM to the process group recorded in cfg.PidFile, killing
// the launcher and all child processes (e.g. uvicorn workers spawned by uv run).
// Falls back to killing just the recorded PID for processes not in their own group.
func Stop(name string, cfg *registry.ServerConfig) error {
	if cfg.PidFile == "" {
		return fmt.Errorf("no pid_file configured for %s", name)
	}
	pid, err := readPID(cfg.PidFile)
	if err != nil {
		return err
	}
	// Try process group first (works when started with Setpgid: true).
	if err := syscall.Kill(-pid, syscall.SIGTERM); err != nil {
		// Fall back to single-process SIGTERM (legacy processes not in own group).
		proc, findErr := os.FindProcess(pid)
		if findErr != nil {
			return fmt.Errorf("find process %d: %w", pid, findErr)
		}
		if err := proc.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("sigterm %d: %w", pid, err)
		}
	}
	// Wait for the process to exit so the port is released before the caller starts a new instance.
	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		if syscall.Kill(pid, 0) != nil {
			break
		}
	}
	return os.Remove(cfg.PidFile)
}

// Status returns health.StatusRunning or health.StatusStopped.
func Status(name string, cfg *registry.ServerConfig) (string, error) {
	if cfg.PidFile == "" {
		return health.StatusStopped, nil
	}
	pid, err := readPID(cfg.PidFile)
	if err != nil {
		return health.StatusStopped, nil
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return health.StatusStopped, nil
	}
	// Signal 0 checks existence without sending a real signal.
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return health.StatusStopped, nil
	}
	return health.StatusRunning, nil
}

// openLogFile returns a file to capture process stdout/stderr.
// Path: ~/.cache/textserve/<name>.log
func openLogFile(name string) (*os.File, error) {
	dir := filepath.Join(os.Getenv("HOME"), ".cache", "textserve")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return os.OpenFile(filepath.Join(dir, name+".log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
}

func readPID(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read pid file %q: %w", path, err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("parse pid from %q: %w", path, err)
	}
	return pid, nil
}
