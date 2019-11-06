package listener

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"gin-container/app/utils"
)

type Message struct {
	ID              int        `json:"id"`
	Arch            string     `json:"arch"`
	Packages        *[]string  `json:"packages"`
}

func (msg *Message) FilterPackages() {
	// filter packages in parallel using go-routines
	nPkgs := len(*msg.Packages)
	channel := make(chan int, nPkgs)
	for i, pkg := range *msg.Packages {
		go filterNevra(i, pkg, msg.Arch, channel)
	}

	// create mask of filtered packages
	packagesMask := make([]int, nPkgs)
	for i := 0; i < nPkgs; i++ {
		index, _ := <- channel
		if index != -1 {
			packagesMask[index] = 1
		}
	}

	// select packages according to mask
	filteredPackages := make([]string, 0, nPkgs)
	for i, pkgMask := range packagesMask {
		if pkgMask == 1 {
			filteredPackages = append(filteredPackages, (*msg.Packages)[i])
		}
	}
	msg.Packages = &filteredPackages
}

// parse nevra and check arch, send index to channel, or -1 to remove
func filterNevra(index int, pkg, arch string, channel chan int) {
	nevra, err := utils.ParseNevra(pkg)
	if err != nil {
		utils.Log("err", err.Error(), "nevra", pkg).Error("unable to parse nevra")
	}

	if nevra.Arch == arch {
		channel <- index
	} else {
		channel <- -1
	}
}

func (msg *Message) ToJSON() []byte {
	bytes, err := json.Marshal(*msg)
	if err != nil {
		utils.Log("err", err.Error()).Error("unable to jsonify message")
	}
	return bytes
}

func (msg *Message) JSONChecksum() string {
	js := msg.ToJSON()
	bytes := sha256.Sum256(js)
	return hex.EncodeToString(bytes[:])
}
