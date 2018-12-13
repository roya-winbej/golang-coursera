package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"log"
	"net"
	"strings"
	"time"
)

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные

type myMicroservice struct {
	Acls []*ACL
	logger chan *Event
}

func (s myMicroservice) Check(ctx context.Context, stub *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

func (s myMicroservice) Add(ctx context.Context, stub *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

func (s myMicroservice) Test(ctx context.Context, stub *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

var logs = []Event{
	Event{
		Host: "event 1",
	},
	Event{
		Host: "event 2",
	},
	Event{
		Host: "event 3",
	},
	Event{
		Host: "event 4",
	},
}

func (s myMicroservice) Logging(stub *Nothing, stream Admin_LoggingServer) error {

	select {
	case event := <- s.logger:
		log.Println("send to client")
		if err := stream.Send(event); err != nil {
			return err
		}
	}

	return nil
}

func (s myMicroservice) Statistics(statInterval *StatInterval, stream Admin_StatisticsServer) error {
	return nil
}

type ACL struct {
	User string
	Methods []string
}

func (s *myMicroservice) checkPermissions(ctx context.Context) (bool, error) {

	md, _ := metadata.FromIncomingContext(ctx)

	method, _ := grpc.Method(ctx)

	consumer, ok := md["consumer"]
	if !ok {
		return false, status.Error(codes.Unauthenticated, "unauthenticated consumer")
	}

	var permissionsGranted = false

	for _, acl := range s.Acls {
		if consumer[0] == acl.User {
			for _, aclMethod := range acl.Methods {
				if strings.Contains(aclMethod, "*") {
					rootMethod := strings.Replace(aclMethod, "*", "", -1)

					if strings.Contains(method, rootMethod) {
						permissionsGranted = true
					}
				}

				if aclMethod == method {
					permissionsGranted = true
				}
			}
		}
	}

	if !permissionsGranted {
		return false, status.Error(codes.Unauthenticated, "unauthenticated consumer")
	}

	return permissionsGranted, nil
}

func parseACL(aclJSON string) ([]*ACL, error) {
	var rawACL map[string]*json.RawMessage
	var acls []*ACL

	err := json.Unmarshal([]byte(aclJSON), &rawACL)
	if err != nil {
		return nil, err
	}

	replacer := strings.NewReplacer("[", "", "]", "", "\"", "")

	for key, value := range rawACL {
		valueJSON, err := json.Marshal(&value)
		if err != nil {
			return nil, err
		}

		strValue := replacer.Replace(string(valueJSON))

		acls = append(acls, &ACL{
			User: key,
			Methods: strings.Split(strValue, ","),
		})
	}

	return acls, nil
}

func StartMyMicroservice(ctx context.Context, address string, aclData string) error {
	acls, err := parseACL(aclData)
	if err != nil {
		return err
	}

	go func() {
		if err := bootServer(ctx, address, acls); err != nil {
			log.Fatalf("Failed to serve %v", err)
		}
	}()

	return nil
}

func aclInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	granted, err := info.Server.(*myMicroservice).checkPermissions(ctx)
	if !granted {
		return nil, err
	}

	reply, err := handler(ctx, req)

	return reply, err
}

func loggerInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	method, _ := grpc.Method(ctx)
	md, _ := metadata.FromIncomingContext(ctx)

	consumer, _ := md["consumer"]
	host, _ := md[":authority"]

	evt := &Event{
		Method: method,
		Timestamp: time.Now().Unix(),
		Consumer: consumer[0],
		Host: host[0],
	}

	reply, err := handler(ctx, req)

	log.Println("send to logger")
	info.Server.(*myMicroservice).logger <- evt

	return reply, err
}

func aclInterceptorStream(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	granted, err := srv.(*myMicroservice).checkPermissions(ss.Context())
	if !granted {
		return err
	}

	err = handler(srv, ss)
	if err != nil {
		return err
	}

	return nil
}

func loggerInterceptorStream(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {

	method, _ := grpc.Method(ss.Context())

	fmt.Println(method)

	// md, _ := metadata.FromIncomingContext(ss.Context())
	//
	// consumer, _ := md["consumer"]
	// host, _ := md[":authority"]
	//
	// evt := &Event{
	// 	Method: method,
	// 	Timestamp: time.Now().Unix(),
	// 	Consumer: consumer[0],
	// 	Host: host[0],
	// }

	err := handler(srv, ss)
	if err != nil {
		return err
	}

	log.Println("send to stream")
	// srv.(*myMicroservice).logger <- evt

	return nil
}

func bootServer(ctx context.Context, address string, acl []*ACL) error {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	s := grpc.NewServer(
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			aclInterceptorStream,
			loggerInterceptorStream,
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			aclInterceptor,
			loggerInterceptor,
		)),
	)

	service := &myMicroservice{
		Acls: acl,
		logger: make(chan *Event),
	}

	RegisterAdminServer(s, service)
	RegisterBizServer(s, service)

	go listenShutdownServer(ctx, s)

	err = s.Serve(lis)
	if err != nil {
		return err
	}

	return nil
}



func listenShutdownServer(ctx context.Context, server *grpc.Server) {
	for {
		select {
		case <-ctx.Done():
			server.Stop()
			return
		}
	}
}