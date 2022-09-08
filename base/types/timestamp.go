package types

import (
	"encoding/json"
	"time"
)

// Go datetime parser does not like slightly incorrect RFC 3339 which we are using (missing Z )
const Rfc3339NoTz = "2006-01-02T15:04:05-07:00"

// timestamp format coming from vmaas /dbchange
const Rfc3339NoT = "2006-01-02 15:04:05.000000+00"

type Rfc3339Timestamp time.Time
type Rfc3339TimestampWithZ time.Time
type Rfc3339TimestampNoT time.Time

func unmarshalTimestamp(data []byte, format string) (time.Time, error) {
	var jd string
	var err error
	var t time.Time
	if err = json.Unmarshal(data, &jd); err != nil {
		return t, err
	}
	t, err = time.Parse(format, jd)
	return t, err
}

func (d Rfc3339Timestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Time().Format(Rfc3339NoTz))
}

func (d *Rfc3339Timestamp) UnmarshalJSON(data []byte) error {
	t, err := unmarshalTimestamp(data, Rfc3339NoTz)
	*d = Rfc3339Timestamp(t)
	return err
}

func (d *Rfc3339Timestamp) Time() *time.Time {
	if d == nil {
		return nil
	}
	return (*time.Time)(d)
}

func (d Rfc3339TimestampWithZ) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Time().Format(time.RFC3339))
}

func (d *Rfc3339TimestampWithZ) UnmarshalJSON(data []byte) error {
	t, err := unmarshalTimestamp(data, time.RFC3339)
	*d = Rfc3339TimestampWithZ(t)
	return err
}

func (d *Rfc3339TimestampWithZ) Time() *time.Time {
	if d == nil {
		return nil
	}
	return (*time.Time)(d)
}

func (d Rfc3339TimestampNoT) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Time().Format(Rfc3339NoT))
}

func (d *Rfc3339TimestampNoT) UnmarshalJSON(data []byte) error {
	t, err := unmarshalTimestamp(data, Rfc3339NoT)
	*d = Rfc3339TimestampNoT(t)
	return err
}

func (d *Rfc3339TimestampNoT) Time() *time.Time {
	if d == nil {
		return nil
	}
	return (*time.Time)(d)
}
