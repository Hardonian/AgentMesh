package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestCLI_VersionCommand(t *testing.T) {
	cmd := newRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"version"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("version command failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "AgentMesh v") {
		t.Errorf("expected version string, got: %s", out)
	}
}

func TestCLI_VersionJSON(t *testing.T) {
	cmd := newRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"version", "--json"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("version --json failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, `"version":`) || !strings.Contains(out, `"gitCommit":`) {
		t.Errorf("expected JSON version output, got: %s", out)
	}
}

func TestCLI_DoctorCommand(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"doctor"})

	// Doctor executes cleanly without error
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("doctor command returned error: %v", err)
	}
}

func TestCLI_DemoRunCommand(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"demo", "run"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("demo run command returned error: %v", err)
	}
}

func TestCLI_InitCommand(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agentmesh-cli-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	origWd, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origWd) }()

	cmd := newRootCmd()
	cmd.SetArgs([]string{"init"})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	if _, err := os.Stat("agent.contract.yaml"); os.IsNotExist(err) {
		t.Errorf("agent.contract.yaml was not generated")
	}
}

func TestCLI_InvalidCommand(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"unknown-command-xyz"})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for unknown command")
	}
}
