package native

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

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

// Stop sends SIGTERM to the process recorded in cfg.PidFile.
func Stop(name string, cfg *registry.ServerConfig) error {
	if cfg.PidFile == "" {
		return fmt.Errorf("no pid_file configured for %s", name)
	}
	pid, err := readPID(cfg.PidFile)
	if err != nil {
		return err
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find process %d: %w", pid, err)
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("sigterm %d: %w", pid, err)
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
