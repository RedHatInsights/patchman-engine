package evaluator

import (
	"app/base"
	"app/base/mqueue"
	"fmt"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/pkg/errors"
	"os"
)

var remediationsPublisher mqueue.Writer

func init() {
	if topic, has := os.LookupEnv("REMEDIATIONS_UPDATE_TOPIC"); has {
		remediationsPublisher = mqueue.WriterFromEnv(topic)
	}
}

type RemediationsState struct {
	HostID string   `json:"host_id"`
	Issues []string `json:"issues"`
}

func publishRemediationsState(id string, response vmaas.UpdatesResponse) error {
	if remediationsPublisher == nil {
		return nil
	}
	advisories := getReportedAdvisories(response)
	var state RemediationsState
	state.HostID = id
	state.Issues = make([]string, 0, len(advisories))
	for a := range advisories {
		state.Issues = append(state.Issues, fmt.Sprintf("patch:%s", a))
	}
	msg, err := mqueue.MessageFromJSON(id, state)
	if err != nil {
		return errors.Wrap(err, "Formatting message")
	}
	return remediationsPublisher.WriteMessages(base.Context, msg)
}
