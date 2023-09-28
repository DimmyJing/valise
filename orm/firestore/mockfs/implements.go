package mockfs

import (
	"context"
	"fmt"

	pb "cloud.google.com/go/firestore/apiv1/firestorepb"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GetDocument overrides the FirestoreServer GetDocument method.
func (s *MockServer) GetDocument(ctx context.Context, req *pb.GetDocumentRequest) (*pb.Document, error) {
	res, err := s.popRPC(req)
	if err != nil {
		return nil, err
	}
	//nolint:forcetypeassert
	return res.(*pb.Document), nil
}

// Commit overrides the FirestoreServer Commit method.
func (s *MockServer) Commit(ctx context.Context, req *pb.CommitRequest) (*pb.CommitResponse, error) {
	res, err := s.popRPC(req)
	if err != nil {
		return nil, err
	}
	//nolint:forcetypeassert
	return res.(*pb.CommitResponse), nil
}

// BatchGetDocuments overrides the FirestoreServer BatchGetDocuments method.
func (s *MockServer) BatchGetDocuments(
	req *pb.BatchGetDocumentsRequest,
	bSrv pb.Firestore_BatchGetDocumentsServer,
) error {
	res, err := s.popRPC(req)
	if err != nil {
		return err
	}

	//nolint:forcetypeassert
	responses := res.([]any)
	for _, res := range responses {
		switch res := res.(type) {
		case *pb.BatchGetDocumentsResponse:
			if err := bSrv.Send(res); err != nil {
				return fmt.Errorf("error sending response: %w", err)
			}
		case error:
			return res
		default:
			panic(fmt.Sprintf("mockfs.BatchGetDocuments: Bad response type: %+v", res))
		}
	}

	return nil
}

// RunQuery overrides the FirestoreServer RunQuery method.
func (s *MockServer) RunQuery(req *pb.RunQueryRequest, qSrv pb.Firestore_RunQueryServer) error {
	res, err := s.popRPC(req)
	// fmt.Println(res, err)
	if err != nil {
		return err
	}

	//nolint:forcetypeassert
	responses := res.([]interface{})
	for _, res := range responses {
		switch res := res.(type) {
		case *pb.RunQueryResponse:
			if err := qSrv.Send(res); err != nil {
				return fmt.Errorf("error sending response: %w", err)
			}
		case error:
			return res
		default:
			panic(fmt.Sprintf("mockfs.RunQuery: Bad response type: %+v", res))
		}
	}

	return nil
}

// BeginTransaction overrides the FirestoreServer BeginTransaction method.
func (s *MockServer) BeginTransaction(
	ctx context.Context,
	req *pb.BeginTransactionRequest,
) (*pb.BeginTransactionResponse, error) {
	res, err := s.popRPC(req)
	if err != nil {
		return nil, err
	}
	//nolint:forcetypeassert
	return res.(*pb.BeginTransactionResponse), nil
}

// Rollback overrides the FirestoreServer Rollback method.
func (s *MockServer) Rollback(ctx context.Context, req *pb.RollbackRequest) (*emptypb.Empty, error) {
	res, err := s.popRPC(req)
	if err != nil {
		return nil, err
	}
	//nolint:forcetypeassert
	return res.(*emptypb.Empty), nil
}

func (s *MockServer) BatchWrite(ctx context.Context, req *pb.BatchWriteRequest) (*pb.BatchWriteResponse, error) {
	res, err := s.popRPC(req)
	if err != nil {
		return nil, err
	}
	//nolint:forcetypeassert
	return res.(*pb.BatchWriteResponse), nil
}

func (s *MockServer) CreateDocument(ctx context.Context, req *pb.CreateDocumentRequest) (*pb.Document, error) {
	res, err := s.popRPC(req)
	if err != nil {
		return nil, err
	}
	//nolint:forcetypeassert
	return res.(*pb.Document), nil
}

func (s *MockServer) DeleteDocument(ctx context.Context, req *pb.DeleteDocumentRequest) (*emptypb.Empty, error) {
	res, err := s.popRPC(req)
	if err != nil {
		return nil, err
	}
	//nolint:forcetypeassert
	return res.(*emptypb.Empty), nil
}

func (s *MockServer) ListCollectionIds(
	ctx context.Context,
	req *pb.ListCollectionIdsRequest,
) (*pb.ListCollectionIdsResponse, error) {
	res, err := s.popRPC(req)
	if err != nil {
		return nil, err
	}
	//nolint:forcetypeassert
	return res.(*pb.ListCollectionIdsResponse), nil
}

func (s *MockServer) ListDocuments(
	ctx context.Context,
	req *pb.ListDocumentsRequest,
) (*pb.ListDocumentsResponse, error) {
	res, err := s.popRPC(req)
	if err != nil {
		return nil, err
	}
	//nolint:forcetypeassert
	return res.(*pb.ListDocumentsResponse), nil
}

func (s *MockServer) PartitionQuery(
	ctx context.Context,
	req *pb.PartitionQueryRequest,
) (*pb.PartitionQueryResponse, error) {
	res, err := s.popRPC(req)
	if err != nil {
		return nil, err
	}
	//nolint:forcetypeassert
	return res.(*pb.PartitionQueryResponse), nil
}

func (s *MockServer) RunAggregationQuery(
	req *pb.RunAggregationQueryRequest,
	srv pb.Firestore_RunAggregationQueryServer,
) error {
	res, err := s.popRPC(req)
	// fmt.Println(res, err)
	if err != nil {
		return err
	}

	//nolint:forcetypeassert
	responses := res.([]interface{})
	for _, res := range responses {
		switch res := res.(type) {
		case *pb.RunAggregationQueryResponse:
			if err := srv.Send(res); err != nil {
				return fmt.Errorf("error sending response: %w", err)
			}
		case error:
			return res
		default:
			panic(fmt.Sprintf("mockfs.RunQuery: Bad response type: %+v", res))
		}
	}

	return nil
}

func (s *MockServer) UpdateDocument(context.Context, *pb.UpdateDocumentRequest) (*pb.Document, error) {
	res, err := s.popRPC(nil)
	if err != nil {
		return nil, err
	}
	//nolint:forcetypeassert
	return res.(*pb.Document), nil
}

func (s *MockServer) Write(srv pb.Firestore_WriteServer) error {
	panic("implement me")
}

// Listen overrides the FirestoreServer Listen method.
func (s *MockServer) Listen(stream pb.Firestore_ListenServer) error {
	req, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("error receiving request: %w", err)
	}

	responses, err := s.popRPC(req)
	if err != nil {
		if status.Code(err) == codes.Unknown {
			panic(err)
		}

		return err
	}
	//nolint:forcetypeassert
	for _, res := range responses.([]interface{}) {
		if err, ok := res.(error); ok {
			return err
		}

		if err := stream.Send(res.(*pb.ListenResponse)); err != nil {
			return fmt.Errorf("error sending response: %w", err)
		}
	}

	return nil
}
