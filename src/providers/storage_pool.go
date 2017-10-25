package providers

import (
	libvirt "github.com/libvirt/libvirt-go"
	)
type storage_pool struct {
	conn *libvirt.Connect
	name string
}

func (sp *storage_pool) CreatePool() {
	sp.conn.StoragePoolCreateXML()
}