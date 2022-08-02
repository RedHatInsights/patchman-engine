package utils

import (
	"app/base/vmaas"
)

// Merge update data from vmaasDataB into vmaasDataA without duplicating.
// Requires sorted update input and returns sorted output.
func MergeVMaaSResponses(vmaasDataA *vmaas.UpdatesV2Response,
	vmaasDataB *vmaas.UpdatesV2Response) (*vmaas.UpdatesV2Response, error) {
	if vmaasDataA == nil {
		return vmaasDataB, nil
	}
	if vmaasDataB == nil {
		return vmaasDataA, nil
	}
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
	if err := RemoveNonLatestPackages(vmaasDataA); err != nil {
		return nil, err
	}
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

// Keep only updates for the latest package in update list
func RemoveNonLatestPackages(updates *vmaas.UpdatesV2Response) error {
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
			return err
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
	return nil
}
