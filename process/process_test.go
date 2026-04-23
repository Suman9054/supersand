package process

import (
	"log/slog"
	"os"
	"testing"
)

func TestSandbox(t *testing.T) {
	if len(os.Args) > 1 && os.Args[1] == "child" {

		if err := RunContainer(); err != nil {
			slog.Error("error in running container:", "error", err)
		}
		return
	}
	s := NewSandbox()
	err := s.CreateNewContainer()
	if err != nil {
		t.Fatalf("failed to create container: %v", err)
	}

	networkErr := s.SetupNetwork("10.0.0.1")
	if networkErr != nil {
		t.Fatalf("failed to setup network: %v", networkErr)
	}

	networkOutput, networkErr := s.RunCommand("ip addr")
	if networkErr != nil {
		t.Fatalf("failed to execute network command: %v", networkErr)
	}

	expectedNetworkOutput := "inet 10.0.0.1/24"
	if networkOutput != expectedNetworkOutput {
		t.Fatalf("expected network output %q but got %q", expectedNetworkOutput, networkOutput)
	}

	output, err := s.RunCommand("echo hellow world")
	if err != nil {
		t.Fatalf("failed to execute command: %v", err)
	}

	expected := "hello world"
	if output != expected {
		t.Fatalf("expected output %q but got %q", expected, output)
	}

	err = s.StopContainer()
	if err != nil {
		t.Fatalf("failed to stop container: %v", err)
	}

	err = s.ResumeContainer()
	if err != nil {
		t.Fatalf("failed to resume container: %v", err)
	}

	err = s.KillContainer()
	if err != nil {
		t.Fatalf("failed to kill container: %v", err)
	}
}
