package evaluator

import (
	"app/base"
	"app/base/models"
	"app/base/mqueue"
	"app/base/utils"
	"app/base/vmaas"
	"fmt"
	"sort"

	"github.com/pkg/errors"
)

var remediationsPublisher mqueue.Writer

func configureRemediations() {
	if topic := utils.Cfg.RemediationUpdateTopic; topic != "" {
		remediationsPublisher = mqueue.NewKafkaWriterFromEnv(topic)
	}
}

type RemediationsState struct {
	HostID string   `json:"host_id"`
	Issues []string `json:"issues"`
}

func createRemediationsStateMsg(id string, response *vmaas.UpdatesV3Response) *RemediationsState {
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

func getReportedAdvisories(vmaasData *vmaas.UpdatesV3Response) map[string]int {
	updateList := vmaasData.GetUpdateList()
	advisories := make(map[string]int, len(updateList))
	for _, updates := range updateList {
		for _, u := range updates.GetAvailableUpdates() {
			advisories[u.GetErratum()] = u.StatusID
		}
	}
	return advisories
}

func getReportedPackageUpdates(vmaasData *vmaas.UpdatesV3Response) map[string]bool {
	updateList := vmaasData.GetUpdateList()
	packages := make(map[string]bool, len(updateList))
	for _, updates := range updateList {
		for _, u := range updates.GetAvailableUpdates() {
			packages[u.GetPackage()] = true
		}
	}
	return packages
}

func publishRemediationsState(system *models.SystemPlatform, response *vmaas.UpdatesV3Response) error {
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
