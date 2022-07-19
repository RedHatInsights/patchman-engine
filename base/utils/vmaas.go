package utils

import (
	"app/base/vmaas"
)

// Merge update data from vmaasDataB into vmaasDataA without duplicating.
// Requires sorted update input and returns sorted output.
func MergeVMaaSResponses(vmaasDataA *vmaas.UpdatesV2Response,
	vmaasDataB *vmaas.UpdatesV2Response) (*vmaas.UpdatesV2Response, error) {
	mergedList := vmaasDataA.GetUpdateList()
	for nevraB, updateListB := range vmaasDataB.GetUpdateList() {
		if updateListA, ok := mergedList[nevraB]; ok {
			merged, err := mergeUpdates(updateListA, updateListB)
			if err != nil {
				return nil, err
			}
			mergedList[nevraB] = *merged
		} else {
			mergedList[nevraB] = updateListB
		}
	}

	vmaasDataA.UpdateList = &mergedList

	return vmaasDataA, nil
}

func mergeUpdates(listA, listB vmaas.UpdatesV2ResponseUpdateList) (*vmaas.UpdatesV2ResponseUpdateList, error) {
	updatesA, updatesB := listA.GetAvailableUpdates(), listB.GetAvailableUpdates()
	newUpdates := make([]vmaas.UpdatesV2ResponseAvailableUpdates, 0)
	a, b := 0, 0
	for a < len(updatesA) && b < len(updatesB) {
		nevraA, err := ParseNevra(updatesA[a].GetPackage())
		if err != nil {
			return nil, err
		}
		nevraB, err := ParseNevra(updatesB[b].GetPackage())
		if err != nil {
			return nil, err
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
	return &vmaas.UpdatesV2ResponseUpdateList{
		AvailableUpdates: &newUpdates,
	}, nil
}
