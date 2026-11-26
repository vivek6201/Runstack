package test

import (
	"context"
	"testing"
	"time"

	"code-runner/shared/cache"
)

func TestCache_DynamicKeyGeneration(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		parts     []interface{}
		expected  string
	}{
		{
			name:      "simple key",
			namespace: "dashboard",
			parts:     []interface{}{"stats", "user-123"},
			expected:  "dashboard:stats:user-123",
		},
		{
			name:      "empty namespace",
			namespace: "",
			parts:     []interface{}{"apikey", "hash999"},
			expected:  "apikey:hash999",
		},
		{
			name:      "mixed types with integers and status",
			namespace: "dashboard",
			parts:     []interface{}{"submissions", "user-abc", "list", 1, 20, "COMPLETED", "python"},
			expected:  "dashboard:submissions:user-abc:list:1:20:COMPLETED:python",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cache.Key(tt.namespace, tt.parts...)
			if got != tt.expected {
				t.Errorf("cache.Key() = %v, want %v", got, tt.expected)
			}

			c := cache.New(nil)
			gotInstance := c.BuildKey(tt.namespace, tt.parts...)
			if gotInstance != tt.expected {
				t.Errorf("c.BuildKey() = %v, want %v", gotInstance, tt.expected)
			}
		})
	}
}

func TestCache_NilClientGracefulDegradation(t *testing.T) {
	ctx := context.Background()
	c := cache.New(nil)

	if c.Client() != nil {
		t.Errorf("expected c.Client() to be nil")
	}

	// 1. Get on nil cache returns false, nil
	var dest string
	found, err := c.Get(ctx, "test:key", &dest)
	if err != nil {
		t.Errorf("unexpected error on Get: %v", err)
	}
	if found {
		t.Errorf("expected found to be false on nil client")
	}

	// 2. Set on nil cache returns nil
	if err := c.Set(ctx, "test:key", "value", time.Minute); err != nil {
		t.Errorf("unexpected error on Set: %v", err)
	}

	// 3. GetOrSet on nil cache executes fallback fetch and populates target
	type TestPayload struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	var fetchedPayload TestPayload
	fetchCalled := false
	err = c.GetOrSet(ctx, "test:payload", &fetchedPayload, time.Minute, func() (interface{}, error) {
		fetchCalled = true
		return TestPayload{Name: "coderunner", Count: 42}, nil
	})
	if err != nil {
		t.Fatalf("unexpected error on GetOrSet: %v", err)
	}
	if !fetchCalled {
		t.Errorf("expected fetch function to be called on cache miss")
	}
	if fetchedPayload.Name != "coderunner" || fetchedPayload.Count != 42 {
		t.Errorf("unexpected target data: %+v", fetchedPayload)
	}

	// 4. Update on nil cache returns nil
	err = c.Update(ctx, "test:key", time.Minute, func(existing []byte) (interface{}, error) {
		return "updated", nil
	})
	if err != nil {
		t.Errorf("unexpected error on Update: %v", err)
	}

	// 5. Revoke and RevokePattern on nil cache return nil
	if err := c.Revoke(ctx, "k1", "k2"); err != nil {
		t.Errorf("unexpected error on Revoke: %v", err)
	}
	if err := c.RevokePattern(ctx, "k*"); err != nil {
		t.Errorf("unexpected error on RevokePattern: %v", err)
	}

	// 6. Incr and Decr on nil cache return defaults
	val, err := c.Incr(ctx, "counter", time.Minute)
	if err != nil || val != 1 {
		t.Errorf("unexpected Incr result: val=%d, err=%v", val, err)
	}
	decVal, err := c.Decr(ctx, "counter")
	if err != nil || decVal != 0 {
		t.Errorf("unexpected Decr result: val=%d, err=%v", decVal, err)
	}

	// 7. Exists on nil cache returns false, nil
	exists, err := c.Exists(ctx, "test:key")
	if err != nil || exists {
		t.Errorf("unexpected Exists result: exists=%v, err=%v", exists, err)
	}
}

func TestCache_FetchWrapper(t *testing.T) {
	ctx := context.Background()
	c := cache.New(nil) // gracefully operates even when Redis is offline

	type UserData struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}

	callCount := 0
	mockDBCall := func() (*UserData, error) {
		callCount++
		return &UserData{ID: "u123", Email: "test@coderunner.com"}, nil
	}

	key := cache.Key("users", "u123")
	user, err := cache.Fetch(ctx, c, key, 5*time.Minute, mockDBCall)
	if err != nil {
		t.Fatalf("unexpected error from Fetch: %v", err)
	}

	if user == nil || user.ID != "u123" || user.Email != "test@coderunner.com" {
		t.Errorf("unexpected user data returned: %+v", user)
	}
	if callCount != 1 {
		t.Errorf("expected dbCall to be invoked once, got %d", callCount)
	}
}

func TestCache_FetchPaginatedWrapper(t *testing.T) {
	ctx := context.Background()
	c := cache.New(nil)

	type Item struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	callCount := 0
	mockDBList := func() ([]Item, int64, error) {
		callCount++
		return []Item{
			{ID: "1", Name: "Item 1"},
			{ID: "2", Name: "Item 2"},
		}, 100, nil
	}

	key := cache.Key("items", "list", 1, 2)
	items, total, err := cache.FetchPaginated(ctx, c, key, 1*time.Minute, mockDBList)
	if err != nil {
		t.Fatalf("unexpected error from FetchPaginated: %v", err)
	}

	if len(items) != 2 || total != 100 {
		t.Errorf("unexpected paginated result: items=%d, total=%d", len(items), total)
	}
	if callCount != 1 {
		t.Errorf("expected dbList to be called once, got %d", callCount)
	}
}

func TestCache_KeySanitization(t *testing.T) {
	got := cache.Key("dashboard", "submissions", "user-1", "list", 1, 20, "COMPLETED:*:injected", "py?*[0]\n")
	want := "dashboard:submissions:user-1:list:1:20:COMPLETED___injected:py___0__"
	if got != want {
		t.Errorf("cache.Key() sanitization mismatch: got %q, want %q", got, want)
	}
}

func TestTypedCache_NilSafetyAndFetch(t *testing.T) {
	type Sample struct {
		Name string `json:"name"`
	}
	tc := cache.NewTypedCache[Sample](nil)
	res, err := tc.Fetch(context.Background(), "sample:1", time.Minute, func() (*Sample, error) {
		return &Sample{Name: "runstack"}, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil || res.Name != "runstack" {
		t.Errorf("unexpected TypedCache result: %+v", res)
	}
	// Async invalidation on nil client must not panic
	tc.InvalidateAsync("sample:1")
	tc.InvalidatePatternAsync("sample:*")
}
