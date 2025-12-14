package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	mockRepository "movieexample.com/gen/mock/metadata/repository"
	"movieexample.com/metadata/internal/controller/metadata"
	"movieexample.com/metadata/internal/repository"
	"movieexample.com/metadata/pkg/model"
)

func TestHandlerGetMetadata(t *testing.T) {
	tests := []struct {
		name        string
		reqId       string
		wantRepoRes *model.Metadata
		wantRepoErr error
		wantRes     *model.Metadata
		wantCode    int
	}{
		{
			name:     "empty id",
			reqId:    "",
			wantCode: http.StatusBadRequest,
		},
		{
			name:        "not found",
			reqId:       "the-movie",
			wantRepoRes: nil,
			wantRepoErr: repository.ErrNotFound,
			wantRes:     nil,
			wantCode:    http.StatusNotFound,
		},
		{
			name:        "internal error",
			reqId:       "the-movie",
			wantRepoRes: nil,
			wantRepoErr: errors.New("unexpected error happen"),
			wantRes:     nil,
			wantCode:    http.StatusInternalServerError,
		},
		{
			name:  "success",
			reqId: "the-movie",
			wantRepoRes: &model.Metadata{
				ID:          "the-movie",
				Title:       "The Movie",
				Description: "The one and only Movie",
				Director:    "The Director",
			},
			wantRepoErr: nil,
			wantRes: &model.Metadata{
				ID:          "the-movie",
				Title:       "The Movie",
				Description: "The one and only Movie",
				Director:    "The Director",
			},
			wantCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repoMock := mockRepository.NewMockmetadataRepository(ctrl)

			ctx := context.Background()

			if tt.reqId != "" {
				repoMock.EXPECT().Get(ctx, tt.reqId).Return(tt.wantRepoRes, tt.wantRepoErr)
			}

			c := metadata.New(repoMock)
			handler := New(c)

			url := fmt.Sprintf("localhost:8081?id=%s", tt.reqId)
			req, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err, tt.name)

			recorder := httptest.NewRecorder()

			handler.GetMetadata(recorder, req)

			assert.Equal(t, tt.wantCode, recorder.Code)
			if tt.wantRes != nil {
				var metadata model.Metadata
				err := json.NewDecoder(recorder.Body).Decode(&metadata)
				require.NoError(t, err, tt.name)
				assert.Equal(t, tt.wantRes, &metadata)
			}
		})
	}
}

func TestHandlerPutMetadata(t *testing.T) {
	tests := []struct {
		name        string
		reqMetadata *model.Metadata
		wantRepoErr error
		wantCode    int
	}{
		{
			name:        "bad request",
			reqMetadata: nil,
			wantCode:    http.StatusBadRequest,
		},
		{
			name: "internal error",
			reqMetadata: &model.Metadata{
				ID:          "the-movie",
				Title:       "The Movie",
				Description: "The one and only Movie",
				Director:    "The Director",
			},
			wantRepoErr: errors.New("something unexpected happened"),
			wantCode:    http.StatusInternalServerError,
		},
		{
			name: "success",
			reqMetadata: &model.Metadata{
				ID:          "the-movie",
				Title:       "The Movie",
				Description: "The one and only Movie",
				Director:    "The Director",
			},
			wantRepoErr: nil,
			wantCode:    http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ctrl := gomock.NewController(t)
			repoMock := mockRepository.NewMockmetadataRepository(ctrl)

			ctx := context.Background()

			if tt.reqMetadata != nil {
				repoMock.EXPECT().Put(ctx, tt.reqMetadata.ID, tt.reqMetadata).Return(tt.wantRepoErr).Times(1)
			}

			c := metadata.New(repoMock)
			handler := New(c)

			byteBody, err := json.Marshal(tt.reqMetadata)
			require.NoError(t, err, tt.name)
			body := bytes.NewBuffer(byteBody)

			req, err := http.NewRequest(http.MethodGet, "localhost:8081", body)
			require.NoError(t, err, tt.name)

			recorder := httptest.NewRecorder()

			handler.PutMetadata(recorder, req)

			assert.Equal(t, tt.wantCode, recorder.Code)
		})
	}
}
