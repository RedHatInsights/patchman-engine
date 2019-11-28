package vmaas_client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type VMaaSMock struct {
	Port int
	UpdatesResp string
	server *http.Server
}

type updatesReq struct {
	PackageList    []string  `json:"package_list"`
	RepositoryList []string  `json:"repository_list"`
	ModulesList    []string  `json:"modules_list"`
	Releasever     string    `json:"releasever"`
	Basearch       string    `json:"basearch"`
}


func (mock * VMaaSMock) Run() {
	http.HandleFunc("/api/v1/updates", handleUpdates(mock.UpdatesResp))

	mock.server = &http.Server{Addr: fmt.Sprintf(":%d", mock.Port), Handler: nil}
	err := mock.server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

func (mock * VMaaSMock) Stop() {
	if mock.server != nil {
		err := mock.server.Close()
		if err != nil {
			panic(err)
		}
	}
}

func handleUpdates(resp string) http.HandlerFunc {
	handl := func(w http.ResponseWriter, r *http.Request) {
		bts, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var parsedReq updatesReq

		err = json.Unmarshal(bts, &parsedReq)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
		}

		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(resp))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
	return handl
}
