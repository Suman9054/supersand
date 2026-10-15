package container

import (
	"log/slog"
	"os"
	"testing"
)

func TestSandbox(t *testing.T) {
	if len(os.Args) > 1 && os.Args[1] == "child" {

		if err := Runcontainer(os.Args[2]); err != nil {
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

	
}
