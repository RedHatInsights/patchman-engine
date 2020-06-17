package evaluator

import (
	"app/base"
	"app/base/mqueue"
	"github.com/RedHatInsights/patchman-clients/vmaas"
)

var remediationsPublisher = mqueue.WriterFromEnv("REMEDIATIONS_UPDATE_TOPIC")

type RemediationsState struct {
	HostID string   `json:"host_id"`
	Issues []string `json:"issues"`
}

func publishRemediationsState(id string, response vmaas.UpdatesV2Response) error {
	var state RemediationsState
	state.HostID = id
	state.Issues = make([]string, 0)
	for a := range getReportedAdvisories(response) {
		state.Issues = append(state.Issues, a)
	}
	msg, err := mqueue.MessageFromJSON(id, state)
	if err != nil {
		return err
	}
	return remediationsPublisher.WriteMessages(base.Context, msg)
}
