/*
Copyright 2021 GramLabs, Inc.

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

package v2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/thestormforge/optimize-go/pkg/api"
)

func NewAPI(client api.Client) API {
	endpoint := os.Getenv("STORMFORGE_APPLICATIONS_ENDPOINT")
	if endpoint == "" {
		endpoint = "v2/applications/"
	} else {
		endpoint = strings.TrimRight(endpoint, "/") + "/"
	}
	return &httpAPI{client: client, endpoint: endpoint}
}

type httpAPI struct {
	client   api.Client
	endpoint string
}

var _ API = &httpAPI{}

func (h *httpAPI) CheckEndpoint(ctx context.Context) (api.Metadata, error) {
	result := api.Metadata{}

	req, err := http.NewRequest(http.MethodHead, h.client.URL(h.endpoint).String(), nil)
	if err != nil {
		return nil, err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		api.UnmarshalMetadata(resp, &result)
		return result, nil
	default:
		return nil, api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) ListApplications(ctx context.Context, q ApplicationListQuery) (ApplicationList, error) {
	u := h.client.URL(h.endpoint)
	u.RawQuery = url.Values(q.IndexQuery).Encode()

	return h.ListApplicationsByPage(ctx, u.String())
}

func (h *httpAPI) ListApplicationsByPage(ctx context.Context, u string) (ApplicationList, error) {
	result := ApplicationList{}

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return result, err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return result, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		api.UnmarshalMetadata(resp, &result.Metadata)
		err = json.Unmarshal(body, &result)
		return result, err
	default:
		return result, api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) CreateApplication(ctx context.Context, app Application) (api.Metadata, error) {
	result := api.Metadata{}
	u := h.client.URL(h.endpoint).String()

	req, err := httpNewJSONRequest(http.MethodPost, u, app)
	if err != nil {
		return nil, err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated:
		api.UnmarshalMetadata(resp, &result)
		return result, nil
	case http.StatusBadRequest:
		return nil, api.NewError(ErrApplicationInvalid, resp, body)
	case http.StatusUnprocessableEntity:
		return nil, api.NewError(ErrApplicationInvalid, resp, body)
	default:
		return nil, api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) CreateApplicationByName(ctx context.Context, n ApplicationName, app Application) (api.Metadata, error) {
	u := h.client.URL(h.endpoint)
	u.Path = path.Join(u.Path, n.String())
	result := api.Metadata{}

	req, err := httpNewJSONRequest(http.MethodPut, u.String(), app)
	if err != nil {
		return nil, err
	}

	req.Header.Set("If-None-Match", "*")

	// TODO Fake support for conditional PUT
	if _, err := h.GetApplication(ctx, u.String()); err == nil {
		msg := fmt.Sprintf("application %q already exists", n)
		return nil, &api.Error{Type: ErrApplicationExists, Message: msg, Location: u.String()}
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusAccepted:
		api.UnmarshalMetadata(resp, &result)
		return result, nil
	case http.StatusBadRequest:
		return nil, api.NewError(ErrApplicationInvalid, resp, body)
	case http.StatusPreconditionFailed:
		return nil, api.NewError(ErrApplicationExists, resp, body)
	case http.StatusUnprocessableEntity:
		return nil, api.NewError(ErrApplicationInvalid, resp, body)
	default:
		return nil, api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) GetApplication(ctx context.Context, u string) (Application, error) {
	result := Application{}

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return result, err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return result, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		api.UnmarshalMetadata(resp, &result.Metadata)
		err = json.Unmarshal(body, &result)
		return result, err
	case http.StatusNotFound:
		return result, api.NewError(ErrApplicationNotFound, resp, body)
	default:
		return result, api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) GetApplicationByName(ctx context.Context, n ApplicationName) (Application, error) {
	u := h.client.URL(h.endpoint)
	u.Path = path.Join(u.Path, n.String())
	result, err := h.GetApplication(ctx, u.String())

	// Improve the "not found" error message using the name
	if eerr, ok := err.(*api.Error); ok && eerr.Type == ErrApplicationNotFound {
		eerr.Message = fmt.Sprintf(`application "%s" not found`, n)
	}

	return result, err
}

func (h *httpAPI) UpdateApplication(ctx context.Context, u string, app Application) (api.Metadata, error) {
	result := api.Metadata{}

	req, err := httpNewJSONRequest(http.MethodPut, u, app)
	if err != nil {
		return nil, err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusAccepted:
		api.UnmarshalMetadata(resp, &result)
		return result, nil
	case http.StatusBadRequest:
		return nil, api.NewError(ErrApplicationInvalid, resp, body)
	case http.StatusUnprocessableEntity:
		return nil, api.NewError(ErrApplicationInvalid, resp, body)
	default:
		return nil, api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) UpdateApplicationByName(ctx context.Context, n ApplicationName, app Application) (api.Metadata, error) {
	u := h.client.URL(h.endpoint)
	u.Path = path.Join(u.Path, n.String())
	return h.UpdateApplication(ctx, u.String(), app)
}

func (h *httpAPI) DeleteApplication(ctx context.Context, u string) error {
	req, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent:
		return nil
	case http.StatusNotFound:
		return api.NewError(ErrApplicationNotFound, resp, body)
	default:
		return api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) ListScenarios(ctx context.Context, u string, q ScenarioListQuery) (ScenarioList, error) {
	u = applyQuery(u, url.Values(q.IndexQuery))
	result := ScenarioList{}

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return result, err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return result, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		err = json.Unmarshal(body, &result)
		return result, err
	default:
		return result, api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) CreateScenario(ctx context.Context, u string, scn Scenario) (api.Metadata, error) {
	result := api.Metadata{}

	req, err := httpNewJSONRequest(http.MethodPost, u, scn)
	if err != nil {
		return nil, err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated:
		api.UnmarshalMetadata(resp, &result)
		return result, nil
	case http.StatusBadRequest:
		return nil, api.NewError(ErrScenarioInvalid, resp, body)
	case http.StatusUnprocessableEntity:
		return nil, api.NewError(ErrScenarioInvalid, resp, body)
	default:
		return nil, api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) CreateScenarioByName(ctx context.Context, u string, n ScenarioName, scn Scenario) (Scenario, error) {
	uu, err := url.Parse(u)
	if err != nil {
		return Scenario{}, err
	}
	uu.Path = path.Join(uu.Path, n.String())
	result := Scenario{}

	req, err := httpNewJSONRequest(http.MethodPut, uu.String(), scn)
	if err != nil {
		return result, err
	}

	req.Header.Set("If-None-Match", "*")

	// TODO Fake support for conditional PUT
	if _, err := h.GetScenario(ctx, uu.String()); err == nil {
		msg := fmt.Sprintf("scenario %q already exists", n)
		return result, &api.Error{Type: ErrScenarioExists, Message: msg, Location: uu.String()}
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return result, err
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusAccepted, http.StatusCreated:
		api.UnmarshalMetadata(resp, &result.Metadata)
		err = json.Unmarshal(body, &result)
		return result, err
	case http.StatusBadRequest:
		return result, api.NewError(ErrScenarioInvalid, resp, body)
	case http.StatusPreconditionFailed:
		return result, api.NewError(ErrScenarioExists, resp, body)
	case http.StatusUnprocessableEntity:
		return result, api.NewError(ErrScenarioInvalid, resp, body)
	default:
		return result, api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) GetScenario(ctx context.Context, u string) (Scenario, error) {
	result := Scenario{}

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return result, err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return result, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		api.UnmarshalMetadata(resp, &result.Metadata)
		err = json.Unmarshal(body, &result)
		return result, err
	case http.StatusNotFound:
		return result, api.NewError(ErrScenarioNotFound, resp, body)
	default:
		return result, api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) GetScenarioByName(ctx context.Context, u string, n ScenarioName) (Scenario, error) {
	uu, err := url.Parse(u)
	if err != nil {
		return Scenario{}, err
	}
	uu.Path = path.Join(uu.Path, n.String())
	return h.GetScenario(ctx, uu.String())
}

func (h *httpAPI) UpdateScenario(ctx context.Context, u string, scn Scenario) (Scenario, error) {
	result := Scenario{}

	req, err := httpNewJSONRequest(http.MethodPut, u, scn)
	if err != nil {
		return result, err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return result, err
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusAccepted, http.StatusCreated:
		api.UnmarshalMetadata(resp, &result.Metadata)
		err = json.Unmarshal(body, &result)
		return result, err
	case http.StatusBadRequest:
		return result, api.NewError(ErrScenarioInvalid, resp, body)
	case http.StatusUnprocessableEntity:
		return result, api.NewError(ErrScenarioInvalid, resp, body)
	default:
		return result, api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) UpdateScenarioByName(ctx context.Context, u string, n ScenarioName, scn Scenario) (Scenario, error) {
	uu, err := url.Parse(u)
	if err != nil {
		return Scenario{}, err
	}
	uu.Path = path.Join(uu.Path, n.String())
	return h.UpdateScenario(ctx, uu.String(), scn)
}

func (h *httpAPI) DeleteScenario(ctx context.Context, u string) error {
	req, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent:
		return nil
	default:
		return api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) PatchScenario(ctx context.Context, u string, scn Scenario) error {
	req, err := httpNewJSONRequest(http.MethodPatch, u, scn)
	if err != nil {
		return err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent:
		return nil
	case http.StatusBadRequest:
		return api.NewError(ErrScenarioInvalid, resp, body)
	case http.StatusUnprocessableEntity:
		return api.NewError(ErrScenarioInvalid, resp, body)
	default:
		return api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) GetTemplate(ctx context.Context, u string) (Template, error) {
	result := Template{}

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return result, err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return result, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		err = json.Unmarshal(body, &result)
		return result, err
	default:
		return result, api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) UpdateTemplate(ctx context.Context, u string, t Template) error {
	req, err := httpNewJSONRequest(http.MethodPut, u, t)
	if err != nil {
		return err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusAccepted:
		return nil
	case http.StatusBadRequest:
		return api.NewError(ErrScanInvalid, resp, body)
	case http.StatusUnprocessableEntity:
		return api.NewError(ErrScanInvalid, resp, body)
	default:
		return api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) PatchTemplate(ctx context.Context, u string, t Template) error {
	req, err := httpNewJSONRequest(http.MethodPatch, u, t)
	if err != nil {
		return err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent:
		return nil
	case http.StatusBadRequest:
		return api.NewError(ErrScanInvalid, resp, body)
	case http.StatusUnprocessableEntity:
		return api.NewError(ErrScanInvalid, resp, body)
	default:
		return api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) ListActivity(ctx context.Context, u string, q ActivityFeedQuery) (ActivityFeed, error) {
	u = applyQuery(u, q.Query)
	result := ActivityFeed{}

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return result, err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return result, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		err = json.Unmarshal(body, &result)
		result.SetBaseURL(u)
		return result, err
	default:
		return result, api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) CreateActivity(ctx context.Context, u string, a Activity) error {
	req, err := httpNewJSONRequest(http.MethodPost, u, a)
	if err != nil {
		return err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent, http.StatusCreated:
		return nil
	case http.StatusBadRequest:
		return api.NewError(ErrActivityInvalid, resp, body)
	case http.StatusUnprocessableEntity:
		return api.NewError(ErrActivityInvalid, resp, body)
	default:
		return api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) DeleteActivity(ctx context.Context, u string) error {
	req, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent:
		return nil
	default:
		return api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) PatchApplicationActivity(ctx context.Context, u string, a ActivityFailure) error {
	req, err := httpNewJSONRequest(http.MethodPatch, u, a)
	if err != nil {
		return err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	default:
		return api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) SubscribeActivity(ctx context.Context, q ActivityFeedQuery) (Subscriber, error) {
	md, err := h.CheckEndpoint(ctx)
	if err != nil {
		return nil, err
	}

	// TODO Also filter on `type=application/feed+json`
	u := md.Link(api.RelationAlternate)
	if u == "" {
		return nil, fmt.Errorf("missing activity feed URL")
	}

	feed, err := h.ListActivity(ctx, u, q)
	if err != nil {
		return nil, err
	}

	return newSubscriber(h, feed), nil
}

func (h *httpAPI) CreateRecommendation(ctx context.Context, u string) (api.Metadata, error) {
	result := api.Metadata{}

	req, err := httpNewJSONRequest(http.MethodPost, h.client.URL(h.endpoint).String(), nil)
	if err != nil {
		return nil, err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated:
		api.UnmarshalMetadata(resp, &result)
		return result, nil
	case http.StatusBadRequest:
		return nil, api.NewError(ErrApplicationInvalid, resp, body)
	case http.StatusUnprocessableEntity:
		return nil, api.NewError(ErrApplicationInvalid, resp, body)
	default:
		return nil, api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) GetRecommendation(ctx context.Context, u string) (Recommendation, error) {
	result := Recommendation{}

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return result, err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return result, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		api.UnmarshalMetadata(resp, &result.Metadata)
		// TODO This should be `err = json.Unmarshal(body, &result)` but the recommendations API isn't setting headers...
		err = api.UnmarshalJSON(body, &result)
		return result, err
	case http.StatusNotFound:
		return result, api.NewError(ErrRecommendationNotFound, resp, body)
	default:
		return result, api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) ListRecommendations(ctx context.Context, u string) (RecommendationList, error) {
	result := RecommendationList{}

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return result, err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return result, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		api.UnmarshalMetadata(resp, &result.Metadata)
		err = json.Unmarshal(body, &result)
		return result, err
	default:
		return result, api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) PatchRecommendations(ctx context.Context, u string, details RecommendationList) error {
	req, err := httpNewJSONRequest(http.MethodPatch, u, details)
	if err != nil {
		return err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent:
		return nil
	case http.StatusBadRequest:
		return api.NewError(ErrRecommendationInvalid, resp, body)
	case http.StatusUnprocessableEntity:
		return api.NewError(ErrRecommendationInvalid, resp, body)
	default:
		return api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) GetClusterByName(ctx context.Context, n ClusterName) (Cluster, error) {
	u := h.client.URL(h.endpoint)
	// TODO This is less then ideal
	u.Path = path.Join(u.Path, "..", "clusters", n.String())
	result, err := h.GetCluster(ctx, u.String())

	// Improve the "not found" error message using the name
	if eerr, ok := err.(*api.Error); ok && eerr.Type == ErrClusterNotFound {
		eerr.Message = fmt.Sprintf(`cluster "%s" not found`, n)
	}

	return result, err
}

func (h *httpAPI) GetCluster(ctx context.Context, u string) (Cluster, error) {
	result := Cluster{}

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return result, err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return result, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		api.UnmarshalMetadata(resp, &result.Metadata)
		// TODO This should be `err = json.Unmarshal(body, &result)` but the clusters API isn't setting headers...
		err = api.UnmarshalJSON(body, &result)
		return result, err
	case http.StatusNotFound:
		return result, api.NewError(ErrClusterNotFound, resp, body)
	default:
		return result, api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) ListClusters(ctx context.Context, q ClusterListQuery) (ClusterList, error) {
	// TODO This is less then ideal
	u := h.client.URL(h.endpoint + "../clusters")
	u.RawQuery = url.Values(q.IndexQuery).Encode()

	result := ClusterList{}

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return result, err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return result, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		api.UnmarshalMetadata(resp, &result.Metadata)
		err = json.Unmarshal(body, &result)
		return result, err
	default:
		return result, api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) PatchCluster(ctx context.Context, u string, c ClusterTitle) error {
	req, err := httpNewJSONRequest(http.MethodPatch, u, c)
	if err != nil {
		return err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	default:
		return api.NewUnexpectedError(resp, body)
	}
}

func (h *httpAPI) DeleteCluster(ctx context.Context, u string) error {
	req, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return err
	}

	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent:
		return nil
	case http.StatusNotFound:
		return api.NewError(ErrClusterNotFound, resp, body)
	default:
		return api.NewUnexpectedError(resp, body)
	}
}

// httpNewJSONRequest returns a new HTTP request with a JSON payload.
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

// applyQuery adds the query values to the supplied URL.
func applyQuery(u string, q url.Values) string {
	if len(q) == 0 {
		return u
	}

	uu, err := url.Parse(u)
	if err != nil {
		return u + "?" + q.Encode()
	}

	qq := uu.Query()
	for k, v := range q {
		// TODO Do we need to be smart about merging with "," strings instead?
		qq[k] = append(qq[k], v...)
	}

	uu.RawQuery = qq.Encode()
	return uu.String()
}
