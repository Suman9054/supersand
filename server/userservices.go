package server

import (
	"bufio"
	"context"
	"log/slog"
	"os"
	"path/filepath"

	rpc_api "github.com/suman9054/supersand/rpc"
	"github.com/suman9054/supersand/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Userservice struct {
	rpc_api.UnimplementedUserservicesServer
	store *store.Store
}

func (s *Userservice) Login(ctx context.Context, req *rpc_api.UserLoginRequest) (*rpc_api.UserLoginRespons, error) {
	homedir, e := os.UserHomeDir()
	if e != nil {
		slog.Error("unabel to locate homedir:%v", e)
		return nil, status.Error(codes.Internal, "failed to locate home dir")
	}

	filepath := filepath.Join(homedir, "sand.in")

	file, err := os.Open(filepath)
	if err != nil {
		slog.Error("unabel to open sand.in file %v", err)
		return nil, status.Error(codes.Internal, "faild to open the auth file")
	}

	defer file.Close()

	buf := bufio.NewReader(file)
	key, err := buf.ReadString('\n')
	if err != nil {
		slog.Error("buf io problem %v", err)
		return nil, status.Error(codes.Internal, "faild to open the auth file")
	}

	if key != req.Authkey {
		slog.Error("un authonticated")
		return nil, status.Error(codes.Unauthenticated, "give an valid auth token")
	}

	return nil, status.Error(codes.Internal, "i dont know check the systm logs")
}
