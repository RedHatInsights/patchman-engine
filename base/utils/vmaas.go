package utils

import (
	"app/base/vmaas"
	"github.com/pkg/errors"
)

// Merges VMaaS responses without duplicating updates. Requires sorted input and returns sorted output.
func MergeVMaaSResponses(vmaasDataA *vmaas.UpdatesV2Response,
	vmaasDataB *vmaas.UpdatesV2Response) (*vmaas.UpdatesV2Response, error) {
	if *vmaasDataA.Basearch != *vmaasDataB.Basearch {
		return nil, errors.New("unable to merge different archs")
	}

	if *vmaasDataA.Releasever != *vmaasDataB.Releasever {
		return nil, errors.New("release versions do not match")
	}

	if !RepoListsEqual(vmaasDataA.RepositoryList, vmaasDataB.RepositoryList) {
		return nil, errors.New("repository lists differ")
	}
	if !ModuleListsEqual(vmaasDataA.ModulesList, vmaasDataB.ModulesList) {
		return nil, errors.New("module lists differ")
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

	return vmaasDataA, nil
}

func mergeUpdates(listA, listB vmaas.UpdatesV2ResponseUpdateList) (*vmaas.UpdatesV2ResponseUpdateList, error) {
	updatesA, updatesB := listA.GetAvailableUpdates(), listB.GetAvailableUpdates()
	if len(updatesA) == 0 && len(updatesB) == 0 {
		return nil, nil
	}
	if len(updatesA) == 0 {
		return &listB, nil
	}
	if len(updatesB) == 0 {
		return &listA, nil
	}

	newUpdates, err := processMerge(updatesA, updatesB)
	if err != nil {
		return nil, err
	}

	return &vmaas.UpdatesV2ResponseUpdateList{
		AvailableUpdates: newUpdates,
	}, nil
}

func processMerge(updatesA,
	updatesB []vmaas.UpdatesV2ResponseAvailableUpdates) (*[]vmaas.UpdatesV2ResponseAvailableUpdates, error) {
	newUpdates := make([]vmaas.UpdatesV2ResponseAvailableUpdates, 0)
	processed := make(map[string]bool)
	offset, i := 0, 0
	for i < len(updatesA) {
		nevraA, err := ParseNevra(updatesA[i].GetPackage())
		if err != nil {
			return nil, err
		}
		if offset > len(updatesB)-1 {
			appendUpdateIfNotExists(&newUpdates, updatesA[i], nevraA.Version, processed)
		}
		for j := 0 + offset; j < len(updatesB); j++ {
			nevraB, err := ParseNevra(updatesB[j].GetPackage())
			if err != nil {
				return nil, err
			}
			AIsLess, err := nevraA.IsLessVersion(nevraB)
			if err != nil {
				return nil, err
			}
			BIsLess, err := nevraB.IsLessVersion(nevraA)
			if err != nil {
				return nil, err
			}
			if AIsLess {
				appendUpdateIfNotExists(&newUpdates, updatesA[i], nevraA.Version, processed)
				break
			}
			if BIsLess {
				appendUpdateIfNotExists(&newUpdates, updatesB[j], nevraB.Version, processed)
				offset++
				if offset == len(updatesB) {
					appendUpdateIfNotExists(&newUpdates, updatesA[i], nevraA.Version, processed)
				}
				continue
			}
			appendUpdateIfNotExists(&newUpdates, updatesA[i], nevraA.Version, processed)
			appendUpdateIfNotExists(&newUpdates, updatesB[j], nevraB.Version, processed)
			offset++
			break
		}
		i++
	}
	if offset <= len(updatesB)-1 {
		newUpdates = append(newUpdates, updatesB[offset:]...)
	}
	return &newUpdates, nil
}

func appendUpdateIfNotExists(list *[]vmaas.UpdatesV2ResponseAvailableUpdates,
	update vmaas.UpdatesV2ResponseAvailableUpdates, version string, inserted map[string]bool) {
	if !inserted[*update.Repository+version] {
		*list = append(*list, update)
		inserted[*update.Repository+version] = true
	}
}

func ModuleListsEqual(listA *[]vmaas.UpdatesV3RequestModulesList, listB *[]vmaas.UpdatesV3RequestModulesList) bool {
	if listA == nil || listB == nil {
		return true
	}

	for _, moduleA := range *listA {
		exists := false
		for _, moduleB := range *listB {
			if moduleA.ModuleName == moduleB.ModuleName {
				exists = true
			}
		}
		if !exists {
			return false
		}
	}

	return true
}

func RepoListsEqual(listA, listB *[]string) bool {
	if listA == nil || listB == nil {
		return true
	}

	for _, repoA := range *listA {
		exists := false
		for _, repoB := range *listB {
			if repoB == repoA {
				exists = true
			}
		}
		if !exists {
			return false
		}
	}

	return true
}
