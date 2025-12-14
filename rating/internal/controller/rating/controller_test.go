package rating

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"movieexample.com/gen/mock/rating/ingester"
	gen "movieexample.com/gen/mock/rating/repository"
	"movieexample.com/rating/internal/repository"
	"movieexample.com/rating/pkg/model"
)

func TestGetAggregatedRating(t *testing.T) {
	tests := []struct {
		name                 string
		recordID             model.RecordID
		recordType           model.RecordType
		expRepoRes           []model.Rating
		expRepoErr           error
		wantAggregatedRating float64
		wantErr              error
	}{
		{
			name:                 "not found",
			recordID:             "the-movie",
			recordType:           "movie",
			expRepoRes:           nil,
			expRepoErr:           repository.ErrNotFound,
			wantAggregatedRating: 0,
			wantErr:              ErrNotFound,
		},
		{
			name:                 "internal error",
			recordID:             "the-movie",
			recordType:           "movie",
			expRepoRes:           nil,
			expRepoErr:           errors.New("something unexpected happen"),
			wantAggregatedRating: 0,
			wantErr:              errors.New("something unexpected happen"),
		},
		{
			name:       "success",
			recordID:   "the-movie",
			recordType: "movie",
			expRepoRes: []model.Rating{
				{
					RecordID:   "the-movie",
					RecordType: "movie",
					UserID:     "alex_20",
					Value:      7,
				},
				{
					RecordID:   "the-movie",
					RecordType: "movie",
					UserID:     "john_doe",
					Value:      4,
				},
				{
					RecordID:   "the-movie",
					RecordType: "movie",
					UserID:     "jackDoe",
					Value:      6,
				},
			},
			expRepoErr:           nil,
			wantAggregatedRating: float64(17) / float64(3),
			wantErr:              nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repoMock := gen.NewMockratingRepository(ctrl)
			ingesterMock := ingester.NewMockratingIngester(ctrl)

			ctx := context.Background()
			repoMock.EXPECT().Get(ctx, tt.recordID, tt.recordType).Return(tt.expRepoRes, tt.expRepoErr).Times(1)

			c := New(repoMock, ingesterMock)

			aggRating, err := c.GetAggregatedRating(ctx, tt.recordID, tt.recordType)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error(), tt.name)
			} else {
				require.NoError(t, err, tt.name)
			}

			assert.Equal(t, tt.wantAggregatedRating, aggRating, tt.name)
		})
	}
}

func TestPutRating(t *testing.T) {
	tests := []struct {
		name       string
		recordID   model.RecordID
		recordType model.RecordType
		rating     *model.Rating
		expRepoErr error
		wantErr    error
	}{
		{
			name:       "internal error",
			recordID:   "the-movie",
			recordType: "movie",
			rating: &model.Rating{
				RecordID:   "the-movie",
				RecordType: "movie",
				UserID:     "alex",
				Value:      5,
			},
			expRepoErr: errors.New("something unexpected happened"),
			wantErr:    errors.New("something unexpected happened"),
		},
		{
			name:       "success",
			recordID:   "the-movie",
			recordType: "movie",
			rating: &model.Rating{
				RecordID:   "the-movie",
				RecordType: "movie",
				UserID:     "alex",
				Value:      5,
			},
			expRepoErr: nil,
			wantErr:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repoMock := gen.NewMockratingRepository(ctrl)
			ingesterMock := ingester.NewMockratingIngester(ctrl)

			ctx := context.Background()
			repoMock.EXPECT().Put(ctx, tt.recordID, tt.recordType, tt.rating).Return(tt.expRepoErr).Times(1)

			c := New(repoMock, ingesterMock)

			err := c.PutRating(ctx, tt.recordID, tt.recordType, tt.rating)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error(), tt.name)
			} else {
				require.NoError(t, err, tt.name)
			}
		})
	}
}

func TestStartIngestion(t *testing.T) {
	tests := []struct {
		name   string
		events []struct {
			event      model.RatingEvent
			expRepoErr error
		}
		expIngErr error
		wantErr   error
	}{
		{
			name:      "ingester error",
			events:    nil,
			expIngErr: errors.New("something unexpected happened"),
			wantErr:   errors.New("something unexpected happened"),
		},
		{
			name: "repository error",
			events: []struct {
				event      model.RatingEvent
				expRepoErr error
			}{
				{
					event: model.RatingEvent{
						Rating: model.Rating{
							RecordID:   "the-movie",
							RecordType: "movie",
							UserID:     "alex_20",
							Value:      5,
						},
						ProviderID: "kafka",
						EventType:  model.RatingEventTypePut,
					},
					expRepoErr: nil,
				},
				{
					event: model.RatingEvent{
						Rating: model.Rating{
							RecordID:   "the-movie",
							RecordType: "movie",
							UserID:     "jackDoe",
							Value:      9,
						},
						ProviderID: "kafka",
						EventType:  model.RatingEventTypePut,
					},
					expRepoErr: errors.New("something unexpected happened"),
				},
			},
			expIngErr: nil,
			wantErr:   errors.New("something unexpected happened"),
		},
		{
			name: "success",
			events: []struct {
				event      model.RatingEvent
				expRepoErr error
			}{
				{
					event: model.RatingEvent{
						Rating: model.Rating{
							RecordID:   "the-movie",
							RecordType: "movie",
							UserID:     "alex_20",
							Value:      5,
						},
						ProviderID: "kafka",
						EventType:  model.RatingEventTypePut,
					},
					expRepoErr: nil,
				},
				{
					event: model.RatingEvent{
						Rating: model.Rating{
							RecordID:   "the-movie",
							RecordType: "movie",
							UserID:     "jackDoe",
							Value:      9,
						},
						ProviderID: "kafka",
						EventType:  model.RatingEventTypePut,
					},
					expRepoErr: nil,
				},
			},
			expIngErr: nil,
			wantErr:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repoMock := gen.NewMockratingRepository(ctrl)
			ingesterMock := ingester.NewMockratingIngester(ctrl)

			ctx := context.Background()

			ch := make(chan model.RatingEvent, 1)
			ingesterMock.EXPECT().Ingest(ctx).Return(ch, tt.expIngErr)

			if tt.events != nil {
				for _, e := range tt.events {
					repoMock.EXPECT().Put(ctx, e.event.RecordID, e.event.RecordType, &model.Rating{UserID: e.event.UserID, Value: e.event.Value}).Return(e.expRepoErr)
				}
			}

			go func() {
				if tt.events != nil {
					for _, e := range tt.events {
						ch <- e.event
					}
				}

				defer close(ch)
			}()

			c := New(repoMock, ingesterMock)

			err := c.StartIngestion(ctx)
			if tt.wantErr != nil {
				assert.EqualError(t, err, err.Error(), tt.name)
			} else {
				require.NoError(t, err, tt.name)
			}
		})
	}
}
