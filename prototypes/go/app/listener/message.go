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
	packages2 := make([]string, 0, len(*msg.Packages))
	for _, pkg := range *msg.Packages {
		nevra, err := utils.ParseRpmName(pkg)
		if err != nil {
			utils.Log("err", err.Error(), "nevra", pkg).Error("unable to parse nevra")
		}
		if nevra.Arch == msg.Arch {
			packages2 = append(packages2, pkg)
		}
	}
	msg.Packages = &packages2
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
