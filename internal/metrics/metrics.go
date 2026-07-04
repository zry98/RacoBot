package metrics

import (
	"sync/atomic"
	"time"
)

var (
	noticesSent  atomic.Int64
	fibAPIErrors atomic.Int64
	lastPushUnix atomic.Int64
	startTime    = time.Now()
)

// AddNoticesSent adds the given n to the total number of notices sent to users
func AddNoticesSent(n int64) {
	noticesSent.Add(n)
}

// IncFIBAPIErrors increments the count of errors returned by the FIB API
func IncFIBAPIErrors() {
	fibAPIErrors.Add(1)
}

// SetLastPush records the time of the last completed PushNewNotices run
func SetLastPush(t time.Time) {
	lastPushUnix.Store(t.Unix())
}

// Snapshot represents a point-in-time view of the runtime metrics
type Snapshot struct {
	UptimeSeconds int64 `json:"uptime_seconds"`
	NoticesSent   int64 `json:"notices_sent_total"`
	FIBAPIErrors  int64 `json:"fibapi_errors_total"`
	LastPushUnix  int64 `json:"last_push_unix,omitempty"`
}

// Get returns a snapshot of the current metrics
func Get() Snapshot {
	return Snapshot{
		UptimeSeconds: int64(time.Since(startTime).Seconds()),
		NoticesSent:   noticesSent.Load(),
		FIBAPIErrors:  fibAPIErrors.Load(),
		LastPushUnix:  lastPushUnix.Load(),
	}
}
