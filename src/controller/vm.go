package controller

import (
	"net/http"
	"fmt"
)

func CreateVM(w http.ResponseWriter, r *http.Request) {
	//
}

func ListVMs(w http.ResponseWriter, r *http.Request) {
	doms, err := conn.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_ACTIVE)
	if err != nil {
		fmt.Fprintln(w, err)
	}
	for _, dom := range doms {
		name, err := dom.GetName()
		if err == nil {
			fmt.Fprintln(w, "VM:" + string(name) + "/n")
		}
		dom.Free()
	}
}