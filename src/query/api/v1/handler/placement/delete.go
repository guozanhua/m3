// Copyright (c) 2018 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package placement

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/m3db/m3/src/cmd/services/m3query/config"
	"github.com/m3db/m3/src/query/api/v1/handler"
	"github.com/m3db/m3/src/query/generated/proto/admin"
	"github.com/m3db/m3/src/query/util/logging"
	clusterclient "github.com/m3db/m3cluster/client"
	"github.com/m3db/m3cluster/placement"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

const (
	placementIDVar    = "id"
	placementForceVar = "force"

	// DeleteHTTPMethod is the HTTP method used with this resource.
	DeleteHTTPMethod = http.MethodDelete
)

var (
	// OldM3DBDeleteURL is the old url for the placement delete handler, maintained
	// for backwards compatibility.
	OldM3DBDeleteURL = fmt.Sprintf("%s/placement/{%s}", handler.RoutePrefixV1, placementIDVar)

	// M3DBDeleteURL is the url for the placement delete handler for the M3DB service.
	M3DBDeleteURL = fmt.Sprintf("%s/services/m3db/placement/{%s}", handler.RoutePrefixV1, placementIDVar)

	// M3AggDeleteURL is the url for the placement delete handler for the M3Agg service.
	M3AggDeleteURL = fmt.Sprintf("%s/services/m3agg/placement/{%s}", handler.RoutePrefixV1, placementIDVar)

	errEmptyID = errors.New("must specify placement ID to delete")
)

// DeleteHandler is the handler for placement deletes.
type DeleteHandler Handler

// NewDeleteHandler returns a new instance of DeleteHandler.
func NewDeleteHandler(client clusterclient.Client, cfg config.Configuration) *DeleteHandler {
	return &DeleteHandler{client: client, cfg: cfg}
}

func (h *DeleteHandler) ServeHTTP(serviceName string, w http.ResponseWriter, r *http.Request) {
	var (
		ctx    = r.Context()
		logger = logging.WithContext(ctx)
		id     = mux.Vars(r)[placementIDVar]
	)

	if id == "" {
		logger.Error("no placement ID provided to delete", zap.Any("error", errEmptyID))
		handler.Error(w, errEmptyID, http.StatusBadRequest)
		return
	}

	var (
		force = r.FormValue(placementForceVar) == "true"
		opts  = NewServiceOptionsFromHeaders(serviceName, r.Header)
	)

	if serviceName == M3AggServiceName {
		// Use default M3Agg values because we're deleting the placement
		// so the specific values don't matter.
		opts = NewServiceOptionsWithDefaultM3AggValues(r.Header)
	}

	service, algo, err := ServiceWithAlgo(h.client, opts)
	if err != nil {
		handler.Error(w, err, http.StatusInternalServerError)
		return
	}

	toRemove := []string{id}

	var newPlacement placement.Placement
	if force {
		newPlacement, err = service.RemoveInstances(toRemove)
		if err != nil {
			logger.Error("unable to delete instances", zap.Any("error", err))
			handler.Error(w, err, http.StatusNotFound)
			return
		}
	} else {
		curPlacement, version, err := service.Placement()
		if err != nil {
			logger.Error("unable to fetch placement", zap.Error(err))
			handler.Error(w, err, http.StatusInternalServerError)
			return
		}

		if err := validateAllAvailable(curPlacement); err != nil {
			logger.Info("unable to remove instance, some shards not available", zap.Error(err), zap.String("instance", id))
			handler.Error(w, err, http.StatusBadRequest)
			return
		}

		newPlacement, err = algo.RemoveInstances(curPlacement, toRemove)
		if err != nil {
			logger.Info("unable to generate placement with instances removed", zap.String("instance", id), zap.Error(err))
			handler.Error(w, err, http.StatusBadRequest)
			return
		}

		if err := service.CheckAndSet(newPlacement, version); err != nil {
			logger.Info("unable to remove instance from placement", zap.String("instance", id), zap.Error(err))
			handler.Error(w, err, http.StatusBadRequest)
			return
		}

		newPlacement = newPlacement.SetVersion(version + 1)
	}

	placementProto, err := newPlacement.Proto()
	if err != nil {
		logger.Error("unable to get placement protobuf", zap.Any("error", err))
		handler.Error(w, err, http.StatusInternalServerError)
		return
	}

	resp := &admin.PlacementGetResponse{
		Placement: placementProto,
		Version:   int32(newPlacement.GetVersion()),
	}

	handler.WriteProtoMsgJSONResponse(w, resp, logger)
}
