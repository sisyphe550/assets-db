package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"os"

	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/sisyphus550/assets-db/backend/pkg/dept"
	"github.com/sisyphus550/assets-db/backend/pkg/middleware"
	"github.com/sisyphus550/assets-db/backend/service/user/model"
	pb "github.com/sisyphus550/assets-db/backend/service/user/rpc/userpb"
)

type userServer struct {
	pb.UnimplementedUserServer
	db           *sql.DB
	accessSecret string
}

func (s *userServer) FindUser(ctx context.Context, req *pb.FindUserReq) (*pb.FindUserResp, error) {
	u, err := findUser(ctx, s.db, req.UserId, req.Username)
	if err != nil {
		return nil, err
	}
	return &pb.FindUserResp{
		Id: u.ID, Username: u.Username, PasswordHash: u.PasswordHash,
		RealName: u.RealName, RoleLevel: int32(u.RoleLevel),
		DepartmentId: u.DepartmentID, Status: int32(u.Status),
	}, nil
}

func (s *userServer) GetDeptSubtree(ctx context.Context, req *pb.GetDeptSubtreeReq) (*pb.GetDeptSubtreeResp, error) {
	dm := model.NewDeptModel(s.db)
	all, err := dm.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	ids, err := dept.SubtreeIDs(model.ToDeptSlice(all), req.DeptId)
	if err != nil {
		return &pb.GetDeptSubtreeResp{DeptIds: []int64{req.DeptId}}, nil
	}
	return &pb.GetDeptSubtreeResp{DeptIds: ids}, nil
}

func (s *userServer) ValidateToken(ctx context.Context, req *pb.ValidateTokenReq) (*pb.ValidateTokenResp, error) {
	claims, err := middleware.ParseRefreshToken(req.Token, s.accessSecret)
	if err != nil {
		return &pb.ValidateTokenResp{Valid: false}, nil
	}
	return &pb.ValidateTokenResp{
		Valid: true, UserId: claims.UID,
		RoleLevel: int32(claims.RoleLevel), DeptId: claims.DeptID,
	}, nil
}

func findUser(ctx context.Context, db *sql.DB, userID int64, username string) (*model.SysUser, error) {
	um := model.NewUserModel(db)
	if userID > 0 {
		return um.FindByID(ctx, userID)
	}
	return um.FindByUsername(ctx, username)
}

func main() {
	dsn := getEnv("POSTGRES_DSN", "postgres://fams:fams_dev_pass@localhost:5432/fams_core?sslmode=disable")
	accessSecret := getEnv("JWT_ACCESS_SECRET", "dev_access_secret_change_me_in_prod")

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer db.Close()

	lis, err := net.Listen("tcp", ":"+getEnv("RPC_PORT", "8081"))
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterUserServer(s, &userServer{db: db, accessSecret: accessSecret})
	reflection.Register(s)

	log.Printf("user-rpc (gRPC) listening on :%s", getEnv("RPC_PORT", "8081"))
	if err := s.Serve(lis); err != nil {
		log.Fatal(err)
	}
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" { return v }
	return def
}
