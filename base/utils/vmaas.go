package utils

import (
	"app/base/models"
	"app/base/vmaas"
	"encoding/json"
)

// Merge update data from vmaasDataB into vmaasDataA without duplicating.
// Requires sorted update input and returns sorted output.
func MergeVMaaSResponses(vmaasDataA, vmaasDataB *vmaas.UpdatesV3Response) *vmaas.UpdatesV3Response {
	if vmaasDataA == nil {
		return vmaasDataB
	}
	if vmaasDataB == nil {
		return vmaasDataA
	}
	mergedList := vmaasDataA.GetUpdateList()
	if mergedList == nil {
		return vmaasDataB
	}
	for nevraB, updateListB := range vmaasDataB.GetUpdateList() {
		if updateListA, ok := mergedList[nevraB]; ok {
			merged := mergeUpdates(updateListA, updateListB)
			mergedList[nevraB] = merged
		} else {
			mergedList[nevraB] = updateListB
		}
	}

	vmaasDataA.UpdateList = &mergedList
	RemoveNonLatestPackages(vmaasDataA)
	return vmaasDataA
}

func mergeUpdates(listA, listB *vmaas.UpdatesV3ResponseUpdateList) *vmaas.UpdatesV3ResponseUpdateList {
	updatesA, updatesB := listA.GetAvailableUpdates(), listB.GetAvailableUpdates()
	newUpdates := make([]vmaas.UpdatesV3ResponseAvailableUpdates, 0)
	a, b := 0, 0
	for a < len(updatesA) && b < len(updatesB) {
		nevraA, err := ParseNevra(updatesA[a].GetPackage())
		if err != nil {
			LogWarn("nevra", nevraA, "Skipping package in mergeUpdates")
			continue
		}
		nevraB, err := ParseNevra(updatesB[b].GetPackage())
		if err != nil {
			LogWarn("nevra", nevraB, "Skipping package in mergeUpdates")
			continue
		}

		cmp := nevraA.Cmp(nevraB)
		if cmp == 0 {
			// NEVRA is the same, check if rest of update is the same
			cmp = updatesA[a].Cmp(&updatesB[b])
		}

		switch {
		case cmp < 0:
			newUpdates = append(newUpdates, updatesA[a])
			a++
		case cmp > 0:
			newUpdates = append(newUpdates, updatesB[b])
			b++
		default: // cmp == 0
			newUpdates = append(newUpdates, updatesA[a])
			a++
			b++
		}
	}
	if a <= len(updatesA)-1 {
		newUpdates = append(newUpdates, updatesA[a:]...)
	}
	if b <= len(updatesB)-1 {
		newUpdates = append(newUpdates, updatesB[b:]...)
	}
	return &vmaas.UpdatesV3ResponseUpdateList{
		AvailableUpdates: &newUpdates,
	}
}

// Keep only updates for the latest package in update list
func RemoveNonLatestPackages(updates *vmaas.UpdatesV3Response) {
	var toDel []string
	type nevraStruct struct {
		nameString string
		nevra      *Nevra
	}
	nameMap := make(map[string]nevraStruct)
	updateList := updates.GetUpdateList()
	for k := range updateList {
		nevra, err := ParseNevra(k)
		if err != nil {
			LogWarn("err", err.Error(), "Removing package because nevra is malformed")
			toDel = append(toDel, k)
			continue
		}
		if _, has := nameMap[nevra.Name]; has {
			// mark older pkg for deletion
			switch cmp := nameMap[nevra.Name].nevra.Cmp(nevra); cmp {
			case -1:
				// nevra is newer
				toDel = append(toDel, nameMap[nevra.Name].nameString)
				// put latest to nameMap for future comparison
				nameMap[nevra.Name] = nevraStruct{k, nevra}
			case 1:
				// nameMap[nevra.Name] is newer
				toDel = append(toDel, k)
			default:
				// should not happen after `mergeUpdates`
				// but we don't need to fail because of that
				continue
			}
		} else {
			nameMap[nevra.Name] = nevraStruct{k, nevra}
		}
	}
	for _, k := range toDel {
		delete(updateList, k)
	}
	updates.UpdateList = &updateList
}

func ParseVmaasJSON(system *models.SystemPlatform) (vmaas.UpdatesV3Request, error) {
	var updatesReq vmaas.UpdatesV3Request
	err := json.Unmarshal([]byte(*system.VmaasJSON), &updatesReq)
	return updatesReq, err
}
