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
	mockingester "movieexample.com/gen/mock/rating/ingester"
	mockrepository "movieexample.com/gen/mock/rating/repository"
	"movieexample.com/rating/internal/controller/rating"
	"movieexample.com/rating/internal/repository"
	"movieexample.com/rating/pkg/model"
)

func TestGetAggregatedRating(t *testing.T) {
	tests := []struct {
		name       string
		req        *gen.GetAggregatedRatingRequest
		expRepoRes []model.Rating
		expRepoErr error
		wantRes    float64
		wantErr    error
		wantCode   codes.Code
	}{
		{
			name:     "nil request",
			req:      nil,
			wantErr:  status.Errorf(codes.InvalidArgument, "nil req or empty id/type"),
			wantCode: codes.InvalidArgument,
		},
		{
			name: "empty record ID",
			req: &gen.GetAggregatedRatingRequest{
				RecordType: "movie",
			},
			wantErr:  status.Errorf(codes.InvalidArgument, "nil req or empty id/type"),
			wantCode: codes.InvalidArgument,
		},
		{
			name: "empty record type",
			req: &gen.GetAggregatedRatingRequest{
				RecordId: "the-movie",
			},
			wantErr:  status.Errorf(codes.InvalidArgument, "nil req or empty id/type"),
			wantCode: codes.InvalidArgument,
		},
		{
			name: "not found",
			req: &gen.GetAggregatedRatingRequest{
				RecordId:   "the-movie",
				RecordType: "movie",
			},
			expRepoRes: nil,
			expRepoErr: repository.ErrNotFound,
			wantRes:    0,
			wantErr:    status.Error(codes.NotFound, rating.ErrNotFound.Error()),
			wantCode:   codes.NotFound,
		},
		{
			name: "internal error",
			req: &gen.GetAggregatedRatingRequest{
				RecordId:   "the-movie",
				RecordType: "movie",
			},
			expRepoRes: nil,
			expRepoErr: errors.New("something unexpected happened"),
			wantRes:    0,
			wantErr:    status.Error(codes.Internal, "something unexpected happened"),
			wantCode:   codes.Internal,
		},
		{
			name: "success",
			req: &gen.GetAggregatedRatingRequest{
				RecordId:   "the-movie",
				RecordType: "movie",
			},
			expRepoRes: []model.Rating{
				{
					RecordID:   "the-movie",
					RecordType: model.RecordTypeMovie,
					UserID:     "alex_1998",
					Value:      5,
				},
				{
					RecordID:   "the-movie",
					RecordType: model.RecordTypeMovie,
					UserID:     "john_doe",
					Value:      10,
				},
				{
					RecordID:   "the-movie",
					RecordType: model.RecordTypeMovie,
					UserID:     "jack_doe",
					Value:      7,
				},
			},
			expRepoErr: nil,
			wantRes:    float64(5+10+7) / float64(3),
			wantErr:    nil,
			wantCode:   codes.OK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockrepo := mockrepository.NewMockratingRepository(ctrl)
			mockIngester := mockingester.NewMockratingIngester(ctrl)

			ctx := context.Background()

			if tt.req != nil && tt.req.RecordId != "" && tt.req.RecordType != "" {
				mockrepo.EXPECT().Get(ctx, model.RecordID(tt.req.RecordId), model.RecordType(tt.req.RecordType)).Return(tt.expRepoRes, tt.expRepoErr)
			}

			c := rating.New(mockrepo, mockIngester)

			handler := New(c)

			res, err := handler.GetAggregatedRating(ctx, tt.req)

			s, _ := status.FromError(err)

			assert.Equal(t, tt.wantCode, s.Code(), tt.name)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error(), tt.name)
			} else {
				require.NoError(t, err, tt.name)
				assert.Equal(t, tt.wantRes, res.RatingValue, tt.name)
			}
		})
	}
}
