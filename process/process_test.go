package process

import (
	"log/slog"
	"os"
	"testing"
)

func TestSandbox(t *testing.T) {
	if len(os.Args) > 1 && os.Args[1] == "child" {

		if err := Runcontaner(os.Args[2]); err != nil {
			slog.Error("error in running container:", "error", err)
		}
		return
	}
	s := NewSandbox()
	err, _ := s.CreateNewContainer()
	if err != nil {
		t.Fatalf("failed to create container: %v", err)
	}

	networkErr := s.SetupNetwork()
	if networkErr != nil {
		t.Fatalf("failed to setup network: %v", networkErr)
	}

	networkoutput, networkErr := s.RunCommand("ip addr")
	if networkErr != nil {
		t.Fatalf("failed to execute network command: %v", networkErr)
	}
	t.Log(networkoutput)

	output, err := s.RunCommand("echo hello world")
	if err != nil {
		t.Fatalf("failed to execute command: %v", err)
	}
	t.Log(output)
	expected := "hello world"
	if output != expected {
		t.Fatalf("expected output %q but got %q", expected, output)
	}
}
