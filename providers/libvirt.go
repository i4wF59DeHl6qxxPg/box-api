package providers

import (
	"bytes"
	"fmt"

	"box-api/models"

	log "github.com/Sirupsen/logrus"
)

type LibvirtConn struct {
	conn *libvirt.Connect
}

func CreateDomain() {
	storagePool, err := conn.LookupStoragePoolByName(store.storagePool)
	if err != nil {
		return fmt.Errorf("failed to lookup vm storage pool: %s", err.(libvirt.Error).Message)
	}

	imagePool, err := conn.LookupStoragePoolByName(image.PoolName)
	if err != nil {
		return fmt.Errorf("failed to lookup image storage pool: %s", err.(libvirt.Error).Message)
	}

	network, err := conn.LookupNetworkByName(store.network)
	if err != nil {
		return err
	}

	if _, err := conn.LookupDomainByName(machine.Name); err == nil {
		return fmt.Errorf("domain with name '%s' already exists", machine.Name)
	}

	var volumeXML bytes.Buffer
	voltplContext := struct {
		Machine *models.VirtualMachine
		Image   *models.Image
		Plan    *models.Plan
	}{machine, image, plan}
	if err := store.voltpl.Execute(&volumeXML, voltplContext); err != nil {
		return fmt.Errorf("failed to create volume xml from template: %s", err)
	}
	imageVolume, err := imagePool.LookupStorageVolByName(image.Id)
	if err != nil {
		return fmt.Errorf("failed to lookup image volume: %s", err)
	}

	log.WithField("xml", volumeXML.String()).Debug("defining volume from xml")
	rootVolume, err := storagePool.StorageVolCreateXMLFrom(volumeXML.String(), imageVolume, 0)
	if err != nil {
		return fmt.Errorf("failed to clone image: %s", err)
	}
	rootVolumePath, err := rootVolume.GetPath()
	if err != nil {
		return fmt.Errorf("failed to get machine volume path: %s", err)
	}

	machine.OS = image.OS
	machine.Arch = image.Arch
	machine.Memory = plan.Memory
	machine.Cpus = plan.Cpus
	machine.ImageId = image.Id
	machine.Plan = plan.Name

	var domainCreationXml bytes.Buffer
	vmtplContext := struct {
		Machine    *models.VirtualMachine
		Image      *models.Image
		Plan       *models.Plan
		VolumePath string
		Network    string
		Metadata   string
	}{machine, image, plan, rootVolumePath, store.network, store.renderMetadata(machine)}
	if err := store.vmtpl.Execute(&domainCreationXml, vmtplContext); err != nil {
		return err
	}
	log.WithField("xml", domainCreationXml.String()).Debug("defining domain from xml")
	domain, err := store.conn.DomainDefineXML(domainCreationXml.String())
	if err != nil {
		return err
	}
	if err := store.fillVm(machine, domain, network); err != nil {
		return err
	}
	if err := store.assignIP(machine); err != nil {
		return err
	}

	log.Debug("creating config drive")
	configDriveVolume, err := store.createConfigDrive(machine, storagePool)
	if err != nil {
		return fmt.Errorf("failed to create config drive: %s", err)
	}
	configDrivePath, err := configDriveVolume.GetPath()
	if err != nil {
		return err
	}

	atttachConfigDriveXML := fmt.Sprintf(`
    <disk type='file' device='cdrom'>
      <source file="%s" />
      <target dev='hdc' bus='ide'/>
      <readonly />
    </disk>
	`, configDrivePath)
	if err := domain.UpdateDeviceFlags(atttachConfigDriveXML, libvirt.DOMAIN_DEVICE_MODIFY_CONFIG); err != nil {
		return fmt.Errorf("failed to attach config drive: %s", err)
	}

	if machine.RootDisk.Type == "qcow2" {
		if err := rootVolume.Resize(uint64(plan.DiskSize), 0); err != nil {
			configDriveVolume.Delete(0)
			return fmt.Errorf("failed to resize root volume: %s", err)
		}
	}
	return fillVm(machine, domain, network)
}

func releaseIP(vm *models.VirtualMachine) error {
	network, err := store.conn.LookupNetworkByName(store.network)
	if err != nil {
		return err
	}
	if vm.Ip == nil {
		log.WithField("machine", vm.Name).Warn("no ip to release")
		return nil
	}
	return network.Update(
		libvirt.NETWORK_UPDATE_COMMAND_DELETE,
		libvirt.NETWORK_SECTION_IP_DHCP_HOST,
		-1,
		fmt.Sprintf(
			`<host mac="%s" name="%s" ip="%s" />`,
			vm.HWAddr, vm.Name, vm.Ip.Address,
		),
		libvirt.NETWORK_UPDATE_AFFECT_LIVE|libvirt.NETWORK_UPDATE_AFFECT_CONFIG,
	)
}

func (store *LibvirtMachinerep) List(machines *models.VirtualMachineList) error {
	domains, err := store.conn.ListAllDomains(0)
	if err != nil {
		return err
	}
	network, err := store.conn.LookupNetworkByName(store.network)
	if err != nil {
		return err
	}

	for _, domain := range domains {
		domainName, err := domain.GetName()
		if err != nil {
			panic(err)
		}
		if store.isIgnored(domainName) {
			continue
		}
		vm := &models.VirtualMachine{}
		if err := store.fillVm(vm, &domain, network); err != nil {
			return err
		}
		machines.Add(vm)
	}
	return nil
}
