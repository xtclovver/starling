package grpcclient

import (
	"time"

	commentpb "github.com/usedcvnt/microtwitter/gen/go/comment/v1"
	mediapb "github.com/usedcvnt/microtwitter/gen/go/media/v1"
	postpb "github.com/usedcvnt/microtwitter/gen/go/post/v1"
	userpb "github.com/usedcvnt/microtwitter/gen/go/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

type Clients struct {
	User    userpb.UserServiceClient
	Post    postpb.PostServiceClient
	Comment commentpb.CommentServiceClient
	Media   mediapb.MediaServiceClient

	conns []*grpc.ClientConn
}

const roundRobinServiceConfig = `{"loadBalancingPolicy":"round_robin"}`

func dnsAddr(addr string) string {
	return "dns:///" + addr
}

func New(userAddr, postAddr, commentAddr, mediaAddr string) (*Clients, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(roundRobinServiceConfig),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	}

	userConn, err := grpc.NewClient(dnsAddr(userAddr), opts...)
	if err != nil {
		return nil, err
	}

	postConn, err := grpc.NewClient(dnsAddr(postAddr), opts...)
	if err != nil {
		userConn.Close()
		return nil, err
	}

	commentConn, err := grpc.NewClient(dnsAddr(commentAddr), opts...)
	if err != nil {
		userConn.Close()
		postConn.Close()
		return nil, err
	}

	mediaConn, err := grpc.NewClient(dnsAddr(mediaAddr), opts...)
	if err != nil {
		userConn.Close()
		postConn.Close()
		commentConn.Close()
		return nil, err
	}

	return &Clients{
		User:    userpb.NewUserServiceClient(userConn),
		Post:    postpb.NewPostServiceClient(postConn),
		Comment: commentpb.NewCommentServiceClient(commentConn),
		Media:   mediapb.NewMediaServiceClient(mediaConn),
		conns:   []*grpc.ClientConn{userConn, postConn, commentConn, mediaConn},
	}, nil
}

func (c *Clients) Close() {
	for _, conn := range c.conns {
		conn.Close()
	}
}
