package evaluator

import (
	"app/base"
	"app/base/models"
	"app/base/mqueue"
	"fmt"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/pkg/errors"
	"os"
	"sort"
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

func createRemediationsStateMsg(id string, response *vmaas.UpdatesV2Response) *RemediationsState {
	advisories := getReportedAdvisories(response)
	packages := getReportedPackageUpdates(response)
	var state RemediationsState
	state.HostID = id
	state.Issues = make([]string, 0, len(advisories)+len(packages))
	for a := range advisories {
		state.Issues = append(state.Issues, fmt.Sprintf("patch:%s", a))
	}
	for p := range packages {
		state.Issues = append(state.Issues, fmt.Sprintf("patch:%s", p))
	}
	sort.Strings(state.Issues)
	return &state
}

func getReportedAdvisories(vmaasData *vmaas.UpdatesV2Response) map[string]bool {
	advisories := map[string]bool{}
	for _, updates := range vmaasData.UpdateList {
		for _, u := range updates.AvailableUpdates {
			advisories[u.Erratum] = true
		}
	}
	return advisories
}

func getReportedPackageUpdates(vmaasData *vmaas.UpdatesV2Response) map[string]bool {
	packages := map[string]bool{}
	for _, updates := range vmaasData.UpdateList {
		for _, u := range updates.AvailableUpdates {
			packages[u.Package] = true
		}
	}
	return packages
}

func publishRemediationsState(system *models.SystemPlatform, response *vmaas.UpdatesV2Response) error {
	if remediationsPublisher == nil {
		return nil
	}

	if response == nil {
		return nil
	}

	if system == nil {
		return nil
	}

	state := createRemediationsStateMsg(system.InventoryID, response)
	msg, err := mqueue.MessageFromJSON(system.InventoryID, state)
	if err != nil {
		return errors.Wrap(err, "Formatting message")
	}
	return remediationsPublisher.WriteMessages(base.Context, msg)
}
