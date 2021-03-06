package service

import (
	"context"
	"fmt"

	pb "github.com/mahmud3253/Project/User_Service/genproto"
	l "github.com/mahmud3253/Project/User_Service/pkg/logger"
	"github.com/mahmud3253/Project/User_Service/pkg/messagebroker"
	cl "github.com/mahmud3253/Project/User_Service/service/grpc_client"
	storage "github.com/mahmud3253/Project/User_Service/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	//"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
)

//UserService ...
type UserService struct {
	storage   storage.IStorage
	logger    l.Logger
	client    cl.GrpcClientI
	publisher map[string]messagebroker.Publisher
}

//NewUserService ...
func NewUserService(db *sqlx.DB, log l.Logger, client cl.GrpcClientI, publisher map[string]messagebroker.Publisher) *UserService {
	return &UserService{
		storage:   storage.NewStoragePg(db),
		logger:    log,
		client:    client,
		publisher: publisher,
	}
}

func (s *UserService) publishUserMessage(userData pb.User) error {
	data, err := userData.Marshal()
	if err != nil {
		return err
	}

	logData := userData.String()

	err = s.publisher["user"].Publish([]byte("user"), data, logData)
	if err != nil {
		return err
	}
	return nil
}

func (s *UserService) CreateUser(ctx context.Context, req *pb.User) (*pb.User, error) {

	user, err := s.storage.User().CreateUser(req)
	if err != nil {
		s.logger.Error("failed while inserting user", l.Error(err))
		return nil, err
	}

	if req.Posts != nil {
		for _, post := range req.Posts {
			post.UserId = user.Id
			postss, err := s.client.PostService().Create(ctx, post)
			if err != nil {
				s.logger.Error("failed while inserting user post", l.Error(err))
				return nil, err
			}
			user.Posts = append(user.Posts, postss)
		}
	}

	err = s.publishUserMessage(*user)
	fmt.Println(err,"+++++++++++++++++++++++++++++++++++++++++++++")
	if err != nil {
		s.logger.Error("failed while publishing info", l.Error(err))
		return nil, status.Error(codes.Internal, "failed while publishing info")
	}
	fmt.Println(err,"+++++++++++++++++++++++++++++++++++++++++++++==========")


	return user, nil
}

func (s *UserService) RegisterUser(ctx context.Context, req *pb.CreateUserAuthReqBody) (*pb.CreateUserAuthResBody, error) {
	user, err := s.storage.User().RegisterUser(req)
	if err != nil {
		s.logger.Error("failed while register user", l.Error(err))
		return nil, err
	}

	return user, nil
}

func (s *UserService) GetByIdUser(ctx context.Context, req *pb.GetByIdRequest) (*pb.User, error) {
	user, err := s.storage.User().GetByIdUser(req.UserId)
	if err != nil {
		s.logger.Error("failed get user posts", l.Error(err))
		return nil, err
	}
	userPosts, err := s.client.PostService().GetAllUserPosts(ctx, &pb.GetUserPostsrequest{
		UserId: req.UserId,
	})
	if err != nil {
		s.logger.Error("failed get user posts", l.Error(err))
		return nil, err
	}
	user.Posts = userPosts.Posts

	return user, err
}

func (s *UserService) GetAllUsers(ctx context.Context, req *pb.Empty) (*pb.GetAllUser, error) {
	users, err := s.storage.User().GetAllUsers()
	if err != nil {
		s.logger.Error("failed get user posts", l.Error(err))
		return nil, err
	}
	user := users.Users
	for _, user := range user {

		userPosts, err := s.client.PostService().GetAllUserPosts(ctx, &pb.GetUserPostsrequest{
			UserId: user.Id,
		})
		if err != nil {
			s.logger.Error("failed get user posts", l.Error(err))
			return nil, err
		}
		user.Posts = userPosts.Posts
	}
	return users, err
}

func (s *UserService) DeleteById(ctx context.Context, req *pb.DeleteByIdReq) (*pb.Empty, error) {
	_, err := s.storage.User().DeleteById(req.UserId)
	if err != nil {
		s.logger.Error("error while deleting", l.Error(err))
		return nil, err
	}
	_, err = s.client.PostService().DeleteByIdPost(ctx, &pb.DeleteByIdPostreq{
		PostId: req.UserId,
	})
	if err != nil {
		s.logger.Error("failed get user posts", l.Error(err))
		return nil, err
	}

	return &pb.Empty{}, err
}

func (s *UserService) UpdateById(ctx context.Context, req *pb.UpdateByIdReq) (*pb.Empty, error) {
	_, err := s.storage.User().UpdateById(req)
	if err != nil {
		s.logger.Error("error while updating", l.Error(err))
		return nil, err
	}
	_, err = s.client.PostService().UpdateByIdPost(ctx, &pb.UpdateByIdPostreq{
		PostId: req.UserId,
		Post:   req.Users.Posts,
	})
	if err != nil {
		s.logger.Error("failed updating user posts", l.Error(err))
		return nil, err
	}

	return &pb.Empty{}, err
}

func (s *UserService) ListUser(ctx context.Context, req *pb.ListUserReq) (*pb.ListUserResponse, error) {
	users, err := s.storage.User().ListUser(req)
	if err != nil {
		s.logger.Error("failed get user posts", l.Error(err))
		return nil, err
	}
	user := users.Users
	for _, user := range user {

		userPosts, err := s.client.PostService().GetAllUserPosts(ctx, &pb.GetUserPostsrequest{
			UserId: user.Id,
		})
		if err != nil {
			s.logger.Error("failed get user posts", l.Error(err))
			return nil, err
		}
		user.Posts = userPosts.Posts
		fmt.Println(userPosts.Posts)
	}
	return &pb.ListUserResponse{
		Users: user,
		Count: users.Count,
	}, err
}

func (s *UserService) CheckField(ctx context.Context, req *pb.CheckFieldRequest) (*pb.CheckFieldResponse, error) {
	check, err := s.storage.User().CheckField(req.Field, req.Value)
	if err != nil {
		s.logger.Error("failed while getting user", l.Error(err))
		return nil, status.Error(codes.Internal, "failed while getting user")

	}
	return &pb.CheckFieldResponse{
		Check: check,
	}, err
}

func (s UserService) LoginUser(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	user, err := s.storage.User().LoginUser(req)
	if err != nil {
		s.logger.Error("failed while logging in user ", l.Error(err))
		return nil, status.Error(codes.Internal, "your password is wrong,please check and retype")
	}
	return user, nil
}

func (s *UserService) LoginUserAuth(ctx context.Context, req *pb.GetByIdRequest) (*pb.LoginResponse, error) {
	user, err := s.storage.User().LoginUserAuth(req.UserId)
	if err != nil {
		s.logger.Error("failed while logging in user ", l.Error(err))
		return nil, status.Error(codes.Internal, "your password is wrong,please check and retype")
	}
	return user, nil
}
