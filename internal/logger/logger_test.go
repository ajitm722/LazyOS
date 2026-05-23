package logger

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultLogPath_XDGStateHome(t *testing.T) {
	mockStateDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", mockStateDir)

	expectedPath := filepath.Join(mockStateDir, "lazyos", "lazyos.log")

	path, err := DefaultLogPath()
	if err != nil {
		t.Fatalf("DefaultLogPath failed: %v", err)
	}

	if path != expectedPath {
		t.Errorf("expected %s, got %s", expectedPath, path)
	}
}

func TestDefaultLogPath_DefaultFallback(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")

	path, err := DefaultLogPath()
	if err != nil {
		t.Fatalf("DefaultLogPath failed: %v", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	expected := filepath.Join(homeDir, ".local", "state", "lazyos", "lazyos.log")
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestDefaultLogPath_UserHomeDirError(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")

	orig := userHomeDir
	userHomeDir = func() (string, error) {
		return "", os.ErrNotExist
	}
	t.Cleanup(func() { userHomeDir = orig })

	_, err := DefaultLogPath()
	if err == nil {
		t.Fatal("expected error from DefaultLogPath, got nil")
	}
}

func TestSetupFile_DefaultPath(t *testing.T) {
	baseTempDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", baseTempDir)

	log, file, resolvedPath, err := SetupFile("")
	if err != nil {
		t.Fatalf("SetupFile failed: %v", err)
	}
	defer file.Close()

	if log == nil {
		t.Error("expected logger to not be nil")
	}

	expectedPath := filepath.Join(baseTempDir, "lazyos", "lazyos.log")
	if resolvedPath != expectedPath {
		t.Errorf("expected resolved path %s, got %s", expectedPath, resolvedPath)
	}

	stat, err := os.Stat(resolvedPath)
	if err != nil {
		t.Fatalf("expected log file to exist on disk: %v", err)
	}
	if stat.IsDir() {
		t.Error("expected log path to be a file, but it is a directory")
	}
}

func TestSetupFile_DefaultLogPathError(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")

	orig := userHomeDir
	userHomeDir = func() (string, error) {
		return "", os.ErrNotExist
	}
	t.Cleanup(func() { userHomeDir = orig })

	_, _, _, err := SetupFile("")
	if err == nil {
		t.Fatal("expected error from SetupFile, got nil")
	}
}

func TestSetupFile_CustomPathCreatesDirectories(t *testing.T) {
	baseTempDir := t.TempDir()
	customLogPath := filepath.Join(baseTempDir, "some", "deep", "nested", "path", "lazyos.log")

	log, file, resolvedPath, err := SetupFile(customLogPath)
	if err != nil {
		t.Fatalf("SetupFile failed: %v", err)
	}
	defer file.Close()

	if log == nil {
		t.Error("expected logger to not be nil")
	}
	if resolvedPath != customLogPath {
		t.Errorf("expected resolved path %s, got %s", customLogPath, resolvedPath)
	}

	stat, err := os.Stat(customLogPath)
	if err != nil {
		t.Fatalf("expected log file to exist on disk, got error: %v", err)
	}
	if stat.IsDir() {
		t.Error("expected log path to be a file, but it is a directory")
	}
}

func TestSetupFile_MkdirAllError(t *testing.T) {
	baseTempDir := t.TempDir()
	// Place a file where a directory component needs to be
	blockPath := filepath.Join(baseTempDir, "block")
	if err := os.WriteFile(blockPath, []byte("block"), 0644); err != nil {
		t.Fatal(err)
	}

	logPath := filepath.Join(baseTempDir, "block", "nested", "lazyos.log")

	_, _, _, err := SetupFile(logPath)
	if err == nil {
		t.Fatal("expected error from SetupFile, got nil")
	}
}

func TestSetupFile_OpenFileError(t *testing.T) {
	baseTempDir := t.TempDir()
	logDir := filepath.Join(baseTempDir, "readonly")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(logDir, 0555); err != nil {
		t.Fatal(err)
	}

	logPath := filepath.Join(logDir, "lazyos.log")

	_, _, _, err := SetupFile(logPath)
	if err == nil {
		t.Fatal("expected error from SetupFile, got nil")
	}
}

func TestContextPropagation(t *testing.T) {
	emptyCtx := context.Background()
	defaultLog := FromContext(emptyCtx)
	if defaultLog == nil {
		t.Fatal("expected fallback logger from empty context, got nil")
	}

	customLog := Setup(os.Stdout, 0)
	ctxWithLog := WithLogger(context.Background(), customLog)

	retrievedLog := FromContext(ctxWithLog)
	if retrievedLog != customLog {
		t.Error("expected to retrieve the exact logger pointer we put in the context")
	}
}
