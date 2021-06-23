/*
Copyright 2020 GramLabs, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/thestormforge/optimize-go/pkg/api"
)

const endpointExperiments = "/experiments/"

// NewAPI returns a new API implementation for the specified client.
func NewAPI(c api.Client) API {
	return &httpAPI{client: c}
}

type httpAPI struct {
	client api.Client
}

var _ API = &httpAPI{}

func (h *httpAPI) Options(ctx context.Context) (Server, error) {
	u := h.client.URL(endpointExperiments).String()
	s := Server{}

	req, err := http.NewRequest(http.MethodOptions, u, nil)
	if err != nil {
		return s, err
	}

	// We actually want to do OPTIONS for the whole server, now that the host:port has been captured, overwrite the RequestURL
	// TODO This isn't working because of backend configuration issues
	// req.URL.Opaque = "*"

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return s, err
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent,
		// TODO Current behavior is to return 404 or 405 instead of 204 (or 200?)
		http.StatusNotFound, http.StatusMethodNotAllowed:
		s.Metadata = api.Metadata(resp.Header)
		return s, nil
	default:
		return s, api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) GetAllExperiments(ctx context.Context, q ExperimentListQuery) (ExperimentList, error) {
	u := h.client.URL(endpointExperiments)
	u.RawQuery = url.Values(q.IndexQuery).Encode()

	return h.GetAllExperimentsByPage(ctx, u.String())
}

func (h *httpAPI) GetAllExperimentsByPage(ctx context.Context, u string) (ExperimentList, error) {
	lst := ExperimentList{}

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return lst, err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return lst, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		lst.Metadata = api.Metadata(resp.Header)
		err = json.Unmarshal(body, &lst)
		return lst, err
	default:
		return lst, api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) GetExperimentByName(ctx context.Context, n ExperimentName) (Experiment, error) {
	u := h.client.URL(endpointExperiments + n.Name()).String()
	exp, err := h.GetExperiment(ctx, u)

	// Improve the "not found" error message using the name
	if eerr, ok := err.(*api.Error); ok && eerr.Type == ErrExperimentNotFound {
		eerr.Message = fmt.Sprintf(`experiment "%s" not found`, n.Name())
	}

	return exp, err
}

func (h *httpAPI) GetExperiment(ctx context.Context, u string) (Experiment, error) {
	e := Experiment{}

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return e, err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return e, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		e.Metadata = api.Metadata(resp.Header)
		err = json.Unmarshal(body, &e)
		return e, err
	case http.StatusNotFound:
		return e, api.NewError(ErrExperimentNotFound, resp, body)
	default:
		return e, api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) CreateExperimentByName(ctx context.Context, n ExperimentName, exp Experiment) (Experiment, error) {
	u := h.client.URL(endpointExperiments + n.Name()).String()
	return h.CreateExperiment(ctx, u, exp)
}

func (h *httpAPI) CreateExperiment(ctx context.Context, u string, exp Experiment) (Experiment, error) {
	e := Experiment{}

	req, err := httpNewJSONRequest(http.MethodPut, u, exp)
	if err != nil {
		return e, err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return e, err
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated:
		e.Metadata = api.Metadata(resp.Header)
		err = json.Unmarshal(body, &e)
		return e, err
	case http.StatusBadRequest:
		return e, api.NewError(ErrExperimentNameInvalid, resp, body)
	case http.StatusConflict:
		return e, api.NewError(ErrExperimentNameConflict, resp, body)
	case http.StatusUnprocessableEntity:
		return e, api.NewError(ErrExperimentInvalid, resp, body)
	default:
		return e, api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) DeleteExperiment(ctx context.Context, u string) error {
	req, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusNoContent:
		return nil
	case http.StatusNotFound:
		return api.NewError(ErrExperimentNotFound, resp, body)
	default:
		return api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) GetAllTrials(ctx context.Context, u string, q TrialListQuery) (TrialList, error) {
	lst := TrialList{}

	rawQuery := url.Values(q.IndexQuery).Encode()
	if rawQuery != "" {
		if uu, err := url.Parse(u); err == nil {
			// TODO Should we be merging the query in this case?
			uu.RawQuery = rawQuery
			u = uu.String()
		}
	}

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return lst, err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return lst, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		err = json.Unmarshal(body, &lst)
		return lst, err
	default:
		return lst, api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) CreateTrial(ctx context.Context, u string, asm TrialAssignments) (TrialAssignments, error) {
	ta := TrialAssignments{}

	req, err := httpNewJSONRequest(http.MethodPost, u, asm)
	if err != nil {
		return ta, err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return ta, err
	}

	switch resp.StatusCode {
	case http.StatusCreated, http.StatusAccepted:
		ta.Metadata = api.Metadata(resp.Header)
		err = json.Unmarshal(body, &ta)
		return ta, nil // TODO Stop ignoring this when the server starts sending a response body
	case http.StatusConflict:
		return ta, api.NewError(ErrExperimentStopped, resp, body)
	case http.StatusUnprocessableEntity:
		return ta, api.NewError(ErrTrialInvalid, resp, body)
	default:
		return ta, api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) NextTrial(ctx context.Context, u string) (TrialAssignments, error) {
	asm := TrialAssignments{}

	req, err := http.NewRequest(http.MethodPost, u, nil)
	if err != nil {
		return asm, err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return asm, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		asm.Metadata = api.Metadata(resp.Header)
		err = json.Unmarshal(body, &asm)
		return asm, err
	case http.StatusGone:
		return asm, api.NewError(ErrExperimentStopped, resp, body)
	case http.StatusServiceUnavailable:
		return asm, api.NewError(ErrTrialUnavailable, resp, body)
	default:
		return asm, api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) ReportTrial(ctx context.Context, u string, vls TrialValues) error {
	if vls.Failed {
		vls.Values = nil
	}

	if vls.StartTime != nil && vls.CompletionTime != nil {
		*vls.StartTime = vls.StartTime.Round(time.Millisecond).UTC()
		*vls.CompletionTime = vls.CompletionTime.Round(time.Millisecond).UTC()
	} else {
		vls.StartTime = nil
		vls.CompletionTime = nil
	}

	req, err := httpNewJSONRequest(http.MethodPost, u, vls)
	if err != nil {
		return err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusCreated:
		return nil
	case http.StatusNotFound:
		return api.NewError(ErrTrialNotFound, resp, body)
	case http.StatusConflict:
		return api.NewError(ErrTrialAlreadyReported, resp, body)
	case http.StatusUnprocessableEntity:
		return api.NewError(ErrTrialInvalid, resp, body)
	default:
		return api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) AbandonRunningTrial(ctx context.Context, u string) error {
	req, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusNoContent:
		return nil
	case http.StatusNotFound:
		return api.NewError(ErrTrialNotFound, resp, body)
	default:
		return api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) LabelExperiment(ctx context.Context, u string, lbl ExperimentLabels) error {
	req, err := httpNewJSONRequest(http.MethodPost, u, lbl)
	if err != nil {
		return err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusCreated:
		return nil
	case http.StatusNotFound:
		return api.NewError(ErrTrialNotFound, resp, body)
	case http.StatusUnprocessableEntity:
		return api.NewError(ErrTrialInvalid, resp, body)
	default:
		return api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) LabelTrial(ctx context.Context, u string, lbl TrialLabels) error {
	req, err := httpNewJSONRequest(http.MethodPost, u, lbl)
	if err != nil {
		return err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusCreated:
		return nil
	case http.StatusNotFound:
		return api.NewError(ErrTrialNotFound, resp, body)
	case http.StatusUnprocessableEntity:
		return api.NewError(ErrTrialInvalid, resp, body)
	default:
		return api.NewUnexpectedError(resp, body)
	}
}

// httpNewJSONRequest returns a new HTTP request with a JSON payload
func httpNewJSONRequest(method, u string, body interface{}) (*http.Request, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, u, bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	return req, err
}
