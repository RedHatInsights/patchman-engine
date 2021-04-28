package base

import (
	"app/base/utils"
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const VMaaSAPIPrefix = "/api/v3"
const RBACApiPrefix = "/api/rbac/v1"

// Go datetime parser does not like slightly incorrect RFC 3339 which we are using (missing Z )
// Also, we're receiving multiple formats from inventory
var timeFormats = []string{
	"2006-01-02T15:04:05.999999-07:00",
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02T15:04:05-07:00",
	"2006-01-02T15:04:05",
}

var Context context.Context
var CancelContext context.CancelFunc

func init() {
	Context, CancelContext = context.WithCancel(context.Background())
}

func HandleSignals() {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		CancelContext()
		utils.Log().Info("SIGTERM/SIGINT handled")
	}()
}

func remove(r rune) rune {
	if r == 0 {
		return -1
	}
	return r
}

// Removes characters, which are not accepted by postgresql driver
// in parameter values
func RemoveInvalidChars(s string) string {
	return strings.Map(remove, s)
}

type Rfc3339Timestamp time.Time

func ParseTime(s string) (time.Time, error) {
	var t time.Time
	var err error
	for _, f := range timeFormats {
		t, err = time.Parse(f, s)
		if err == nil {
			return t, nil
		}
	}
	return t, err
}
func (d Rfc3339Timestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Time().Format(timeFormats[0]))
}

func (d *Rfc3339Timestamp) UnmarshalJSON(data []byte) error {
	var jd string
	var err error
	if err := json.Unmarshal(data, &jd); err != nil {
		return err
	}
	t, err := ParseTime(jd)
	*d = Rfc3339Timestamp(t)
	return err
}

func (d *Rfc3339Timestamp) Time() *time.Time {
	if d == nil {
		return nil
	}
	return (*time.Time)(d)
}
