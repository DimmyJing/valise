package mockfs

import (
	"context"
	"fmt"
	"sort"

	"cloud.google.com/go/firestore"
	pb "cloud.google.com/go/firestore/apiv1/firestorepb"
	gsrv "github.com/weathersource/go-gsrv"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/runtime/protoimpl"
)

// New creates a new Firestore Client and MockServer.
func New() (*firestore.Client, *MockServer, error) {
	srv, err := newServer()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Firestore server: %w", err)
	}

	conn, err := grpc.Dial(srv.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Firestore connection: %w", err)
	}

	client, err := firestore.NewClient(context.Background(), "projectID", option.WithGRPCConn(conn))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Firestore client: %w", err)
	}

	return client, srv, nil
}

// MockServer mocks the pb.FirestoreServer interface
// (https://godoc.org/google.golang.org/genproto/googleapis/firestore/v1beta1#FirestoreServer)
type MockServer struct {
	Addr     string
	reqItems []reqItem
	resps    []interface{}
}

var _ pb.FirestoreServer = (*MockServer)(nil)

type reqItem struct {
	wantReq proto.Message
	adjust  func(gotReq proto.Message)
}

func newServer() (*MockServer, error) {
	srv, err := gsrv.NewServer()
	if err != nil {
		return nil, fmt.Errorf("error creating new server: %w", err)
	}

	//nolint:exhaustruct
	mock := &MockServer{Addr: srv.Addr}
	pb.RegisterFirestoreServer(srv.Gsrv, mock)
	srv.Start()

	return mock, nil
}

// Reset returns the MockServer to an empty state.
func (s *MockServer) Reset() {
	s.reqItems = nil
	s.resps = nil
}

// AddRPC adds a (request, response) pair to the server's list of expected
// interactions. The server will compare the incoming request with wantReq
// using proto.Equal. The response can be a message or an error.
//
// For the Listen RPC, resp should be a []interface{}, where each element
// is either ListenResponse or an error.
//
// Passing nil for wantReq disables the request check.
func (s *MockServer) AddRPC(wantReq proto.Message, resp interface{}) {
	s.AddRPCAdjust(wantReq, resp, nil)
}

// AddRPCAdjust is like AddRPC, but accepts a function that can be used
// to tweak the requests before comparison, for example to adjust for
// randomness.
func (s *MockServer) AddRPCAdjust(wantReq proto.Message, resp interface{}, adjust func(gotReq proto.Message)) {
	s.reqItems = append(s.reqItems, reqItem{wantReq, adjust})
	s.resps = append(s.resps, resp)
}

// popRPC compares the request with the next expected (request, response) pair.
// It returns the response, or an error if the request doesn't match what
// was expected or there are no expected rpcs.
func (s *MockServer) popRPC(gotReq proto.Message) (interface{}, error) {
	if len(s.reqItems) == 0 || len(s.resps) == 0 {
		panic("mockfs.popRPC: Out of RPCs.")
	}

	requestItems := s.reqItems[0]
	resp := s.resps[0]
	s.reqItems = s.reqItems[1:]
	s.resps = s.resps[1:]

	if requestItems.wantReq != nil {
		if requestItems.adjust != nil {
			requestItems.adjust(gotReq)
		}

		// Sort FieldTransforms by FieldPath, since slice order is undefined and proto.Equal
		// is strict about order.
		//nolint:gocritic
		switch gotReqTyped := gotReq.(type) {
		case *pb.CommitRequest:
			for _, w := range gotReqTyped.Writes {
				switch opTyped := w.Operation.(type) {
				case *pb.Write_Transform:
					sort.Sort(byFieldPath(opTyped.Transform.FieldTransforms))
				}
			}
		}

		if !proto.Equal(gotReq, requestItems.wantReq) {
			//nolint:goerr113
			return nil, fmt.Errorf("mockfs.popRPC: Bad request\ngot:  %T\n%s\nwant: %T\n%s",
				gotReq, protoimpl.X.MessageStringOf(gotReq),
				requestItems.wantReq, protoimpl.X.MessageStringOf(requestItems.wantReq))
		}
	}

	if err, ok := resp.(error); ok {
		return nil, err
	}

	return resp, nil
}

type byFieldPath []*pb.DocumentTransform_FieldTransform

func (a byFieldPath) Len() int           { return len(a) }
func (a byFieldPath) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byFieldPath) Less(i, j int) bool { return a[i].FieldPath < a[j].FieldPath }
