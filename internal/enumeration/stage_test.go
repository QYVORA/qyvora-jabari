package enumeration

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QYVORA/qyvora-jabari/internal/core"
	"github.com/QYVORA/qyvora-jabari/pkg/models"
)

// fakeTransport implements transport.Transport for stage tests. It records
// the maximum number of in-flight Execute calls so tests can prove the
// detail pass actually runs concurrently, and it tracks call order so tests
// can prove results are written back in package order.
type fakeTransport struct {
	maxInFlight int64
	inFlight    int64
	calls       []string
	mu          sync.Mutex
}

func (f *fakeTransport) Connect(_ context.Context) error { return nil }
func (f *fakeTransport) Disconnect() error                 { return nil }
func (f *fakeTransport) String() string                    { return "fake" }
func (f *fakeTransport) Info(_ context.Context) (*models.DeviceInfo, error) {
	return &models.DeviceInfo{}, nil
}

func (f *fakeTransport) Execute(_ context.Context, req models.Request) (models.Response, error) {
	f.mu.Lock()
	f.calls = append(f.calls, req.Args[len(req.Args)-1])
	f.mu.Unlock()

	cur := atomic.AddInt64(&f.inFlight, 1)
	for {
		max := atomic.LoadInt64(&f.maxInFlight)
		if cur <= max || atomic.CompareAndSwapInt64(&f.maxInFlight, max, cur) {
			break
		}
	}
	defer atomic.AddInt64(&f.inFlight, -1)

	// A minimal "pm list packages" or per-package dumpsys reply. The delay on
	// per-package calls makes concurrent overlap observable in the test.
	switch req.Args[0] {
	case "pm":
		return models.Response{Stdout: []byte("package:com.a\npackage:com.b\npackage:com.c\npackage:com.d\n")}, nil
	default:
		time.Sleep(20 * time.Millisecond)
		return models.Response{Stdout: []byte("versionCode=10\nversionName=1.0\n")}, nil
	}
}

func TestStageRunConcurrent(t *testing.T) {
	ft := &fakeTransport{}
	env := &core.Env{
		Transport: ft,
		Session:   models.NewSession(),
	}
	s := &Stage{DetailLimit: 10, DetailWorkers: 4}

	if err := s.Run(context.Background(), env); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(env.Apps) != 4 {
		t.Fatalf("got %d apps, want 4", len(env.Apps))
	}
	for i, want := range []string{"com.a", "com.b", "com.c", "com.d"} {
		if env.Apps[i].PackageName != want {
			t.Errorf("apps[%d] = %q, want %q (order must be preserved)", i, env.Apps[i].PackageName, want)
		}
	}
	if got := atomic.LoadInt64(&ft.maxInFlight); got < 2 {
		t.Errorf("max in-flight detail calls = %d, want > 1 (detail pass must run concurrently)", got)
	}
}
