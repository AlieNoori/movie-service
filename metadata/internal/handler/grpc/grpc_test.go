package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"movieexample.com/gen"
	mockRepository "movieexample.com/gen/mock/metadata/repository"
	"movieexample.com/metadata/internal/controller/metadata"
	"movieexample.com/metadata/pkg/model"
)

func TestHandlerGet(t *testing.T) {
	tests := []struct {
		name       string
		req        *gen.GetMetadataRequest
		expRepoRes *model.Metadata
		expRepoErr error
		wantRes    *model.Metadata
		wantErr    error
		wantCode   codes.Code
	}{
		{
			name:     "nil request",
			req:      nil,
			wantErr:  status.Errorf(codes.InvalidArgument, "nil req or empty id"),
			wantCode: codes.InvalidArgument,
		},
		{
			name: "empty id",
			req: &gen.GetMetadataRequest{
				MovieId: "",
			},
			wantErr:  status.Errorf(codes.InvalidArgument, "nil req or empty id"),
			wantCode: codes.InvalidArgument,
		},
		{
			name: "not found",
			req: &gen.GetMetadataRequest{
				MovieId: "the-moive",
			},
			expRepoRes: nil,
			expRepoErr: metadata.ErrNotFound,
			wantErr:    status.Error(codes.NotFound, metadata.ErrNotFound.Error()),
			wantRes:    nil,
			wantCode:   codes.NotFound,
		},
		{
			name: "internal error",
			req: &gen.GetMetadataRequest{
				MovieId: "the-moive",
			},
			expRepoRes: nil,
			expRepoErr: errors.New("something unexpected happen"),
			wantErr:    status.Error(codes.Internal, "something unexpected happen"),
			wantRes:    nil,
			wantCode:   codes.Internal,
		},
		{
			name: "success",
			req: &gen.GetMetadataRequest{
				MovieId: "the-moive",
			},
			expRepoRes: &model.Metadata{
				ID:          "the-moive",
				Title:       "The Movie",
				Description: "The one and only movie",
				Director:    "The Director",
			},
			expRepoErr: nil,
			wantRes: &model.Metadata{
				ID:          "the-moive",
				Title:       "The Movie",
				Description: "The one and only movie",
				Director:    "The Director",
			},
			wantErr:  nil,
			wantCode: codes.OK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockRepo := mockRepository.NewMockmetadataRepository(ctrl)

			ctx := context.Background()
			if tt.req != nil && tt.req.MovieId != "" {
				mockRepo.EXPECT().Get(ctx, tt.req.MovieId).Return(tt.expRepoRes, tt.expRepoErr)
			}

			c := metadata.New(mockRepo)
			handler := New(c)

			res, err := handler.GetMetadata(ctx, tt.req)
			status, _ := status.FromError(err)

			assert.Equal(t, tt.wantCode, status.Code(), tt.name)

			if tt.wantErr == nil {
				require.NoError(t, err, tt.name)
				assert.Equal(t, tt.wantRes, model.MetadataFromProto(res.Metadata))
			} else {
				assert.EqualError(t, err, tt.wantErr.Error(), tt.name)
			}
		})
	}
}

func TestHandlerPut(t *testing.T) {
	tests := []struct {
		name       string
		req        *gen.PutMetadataRequest
		expRepoErr error
		wantRes    *gen.PutMetadataResponse
		wantErr    error
		wantCode   codes.Code
	}{
		{
			name:     "nil request",
			wantErr:  status.Errorf(codes.InvalidArgument, "nil req or metadata or empty id"),
			wantRes:  nil,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "nil metadata",
			req: &gen.PutMetadataRequest{
				Metadata: nil,
			},
			wantErr:  status.Errorf(codes.InvalidArgument, "nil req or metadata or empty id"),
			wantRes:  nil,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "empty id",
			req: &gen.PutMetadataRequest{
				Metadata: &gen.Metadata{
					Title:       "The Movie",
					Description: "The one and only movie",
					Director:    "The Director",
				},
			},
			wantErr:  status.Errorf(codes.InvalidArgument, "nil req or metadata or empty id"),
			wantRes:  nil,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "internal error",
			req: &gen.PutMetadataRequest{
				Metadata: &gen.Metadata{
					Id:          "the-movie",
					Title:       "The Movie",
					Description: "The one and only movie",
					Director:    "The Director",
				},
			},
			expRepoErr: errors.New("something unexpected happen"),
			wantErr:    status.Error(codes.Internal, "something unexpected happen"),
			wantRes:    nil,
			wantCode:   codes.Internal,
		},
		{
			name: "success",
			req: &gen.PutMetadataRequest{
				Metadata: &gen.Metadata{
					Id:          "the-movie",
					Title:       "The Movie",
					Description: "The one and only movie",
					Director:    "The Director",
				},
			},
			expRepoErr: nil,
			wantRes:    &gen.PutMetadataResponse{},
			wantErr:    nil,
			wantCode:   codes.OK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockRepo := mockRepository.NewMockmetadataRepository(ctrl)

			ctx := context.Background()
			if tt.req != nil && tt.req.Metadata != nil && tt.req.Metadata.Id != "" {
				mockRepo.EXPECT().Put(ctx, tt.req.Metadata.Id, model.MetadataFromProto(tt.req.Metadata)).Return(tt.expRepoErr)
			}

			c := metadata.New(mockRepo)
			handler := New(c)

			res, err := handler.PutMetadata(ctx, tt.req)
			status, _ := status.FromError(err)

			assert.Equal(t, tt.wantCode, status.Code(), tt.name)

			if tt.wantErr == nil {
				require.NoError(t, err, tt.name)
				assert.Equal(t, tt.wantRes, res, tt.name)
			} else {
				assert.EqualError(t, err, tt.wantErr.Error(), tt.name)
			}
		})
	}
}
