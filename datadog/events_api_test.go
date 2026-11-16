package datadog

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	datadogV2 "github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/stretchr/testify/assert"
)

type fakeEventsAPI struct {
	mu       sync.Mutex
	calls    int
	lastBody *datadogV2.EventsListRequest
	response datadogV2.EventsListResponse
	err      error
}

func (f *fakeEventsAPI) SearchEvents(ctx context.Context, o ...datadogV2.SearchEventsOptionalParameters) (datadogV2.EventsListResponse, *http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if len(o) > 0 {
		f.lastBody = o[0].Body
	}
	return f.response, nil, f.err
}

func eventWithStatus(status *datadogV2.EventStatusType) datadogV2.EventResponse {
	return datadogV2.EventResponse{
		Attributes: &datadogV2.EventResponseAttributes{
			Attributes: &datadogV2.EventAttributes{Status: status},
		},
	}
}

func TestIsRevisionTriggered(t *testing.T) {
	errStatus := datadogV2.EVENTSTATUSTYPE_ERROR
	okStatus := datadogV2.EVENTSTATUSTYPE_SUCCESS

	for _, tc := range []struct {
		name      string
		events    []datadogV2.EventResponse
		triggered bool
	}{
		{"no events", nil, false},
		{"only ok events", []datadogV2.EventResponse{eventWithStatus(&okStatus)}, false},
		{"error event", []datadogV2.EventResponse{eventWithStatus(&errStatus)}, true},
		{"ok then error", []datadogV2.EventResponse{eventWithStatus(&okStatus), eventWithStatus(&errStatus)}, true},
		// regression: these used to panic on a nil dereference
		{"event without status", []datadogV2.EventResponse{eventWithStatus(nil)}, false},
		{"event without inner attributes", []datadogV2.EventResponse{{Attributes: &datadogV2.EventResponseAttributes{}}}, false},
		{"event without attributes", []datadogV2.EventResponse{{}}, false},
		{"malformed event before error", []datadogV2.EventResponse{{}, eventWithStatus(&errStatus)}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := InjectEventsAPI(&fakeEventsAPI{response: datadogV2.EventsListResponse{Data: tc.events}}, time.Minute)
			triggered, attrs, err := api.IsRevisionTriggered(context.Background(), "echo", "main-123")
			assert.NoError(t, err)
			assert.Equal(t, tc.triggered, triggered)
			if tc.triggered {
				assert.NotNil(t, attrs)
			} else {
				assert.Nil(t, attrs)
			}
		})
	}
}

func TestIsRevisionTriggeredQuery(t *testing.T) {
	fake := &fakeEventsAPI{}
	api := InjectEventsAPI(fake, time.Minute)

	_, _, err := api.IsRevisionTriggered(context.Background(), "echo", "main-123")
	assert.NoError(t, err)

	if assert.NotNil(t, fake.lastBody) && assert.NotNil(t, fake.lastBody.Filter) {
		assert.Equal(t, "echo-canary-monitor AND status:error AND version:main-123", *fake.lastBody.Filter.Query)

		from, err := time.Parse(time.RFC3339, *fake.lastBody.Filter.From)
		assert.NoError(t, err)
		to, err := time.Parse(time.RFC3339, *fake.lastBody.Filter.To)
		assert.NoError(t, err)
		assert.InDelta(t, 5*time.Minute.Seconds(), to.Sub(from).Seconds(), 1, "searches the last five minutes")
	}
}

func TestIsRevisionTriggeredError(t *testing.T) {
	apiErr := errors.New("boom")
	api := InjectEventsAPI(&fakeEventsAPI{err: apiErr}, time.Minute)

	triggered, attrs, err := api.IsRevisionTriggered(context.Background(), "echo", "main-123")
	assert.ErrorIs(t, err, apiErr)
	assert.False(t, triggered)
	assert.Nil(t, attrs)
}

func TestIsRevisionTriggeredCache(t *testing.T) {
	fake := &fakeEventsAPI{}
	api := InjectEventsAPI(fake, time.Hour)

	for i := 0; i < 3; i++ {
		_, _, err := api.IsRevisionTriggered(context.Background(), "echo", "main-123")
		assert.NoError(t, err)
	}
	assert.Equal(t, 1, fake.calls, "repeat queries within the ttl are served from cache")

	// a different tag is a different query
	_, _, err := api.IsRevisionTriggered(context.Background(), "echo", "main-124")
	assert.NoError(t, err)
	assert.Equal(t, 2, fake.calls)
}

func TestIsRevisionTriggeredCacheExpires(t *testing.T) {
	fake := &fakeEventsAPI{}
	api := InjectEventsAPI(fake, time.Nanosecond)

	_, _, err := api.IsRevisionTriggered(context.Background(), "echo", "main-123")
	assert.NoError(t, err)
	time.Sleep(5 * time.Millisecond)
	_, _, err = api.IsRevisionTriggered(context.Background(), "echo", "main-123")
	assert.NoError(t, err)
	assert.Equal(t, 2, fake.calls, "expired entries are re-queried")
}

func TestIsRevisionTriggeredErrorsAreNotCached(t *testing.T) {
	fake := &fakeEventsAPI{err: errors.New("boom")}
	api := InjectEventsAPI(fake, time.Hour)

	_, _, err := api.IsRevisionTriggered(context.Background(), "echo", "main-123")
	assert.Error(t, err)

	fake.err = nil
	_, _, err = api.IsRevisionTriggered(context.Background(), "echo", "main-123")
	assert.NoError(t, err)
	assert.Equal(t, 2, fake.calls)
}
