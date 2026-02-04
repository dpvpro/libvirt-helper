package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"libvirt.org/go/libvirt"
)

type VirtualMachineStatus string

const (
	VirtStatePending     = VirtualMachineStatus("pending")     // VM was just created and there is no state yet
	VirtStateRunning     = VirtualMachineStatus("running")     // VM is running
	VirtStateBlocked     = VirtualMachineStatus("blocked")     // VM Blocked on resource
	VirtStatePaused      = VirtualMachineStatus("paused")      // VM is paused
	VirtStateShutdown    = VirtualMachineStatus("shutdown")    // VM is being shut down
	VirtStateShutoff     = VirtualMachineStatus("shutoff")     // VM is shut off
	VirtStateCrashed     = VirtualMachineStatus("crashed")     // Most likely VM crashed on startup cause something is missing.
	VirtStateHybernating = VirtualMachineStatus("hybernating") // VM is hybernating usually due to guest machine request
)

type VirtualMachineStateInfo struct {
	State          VirtualMachineStatus
	MaxMemoryBytes uint64
	MemoryBytes    uint64
	CpuTime        uint64
	CpuCount       uint
}

// Versions - originally created for testing purposes, not actually something we would need.
// var libvirtVersion = *pflag.Bool("libvirt-version", false, "Returns result with version of libvirt populated")
// var virshVersion = *pflag.Bool("virsh-version", false, "Returns result with version of virsh populated")
// var tarsvirtVersion = *pflag.Bool("tarsvirt-version", false, "Returns result with version of tarsvirt populated")

// VirtualMachine commands
var virtualMachineState = pflag.Bool("state", false, "Returns result with a current machine state")
var virtualMachineSoftReboot = pflag.Bool("soft-reboot", false, "reboots a machine gracefully, as chosen by hypervisor. Returns result with a current machine state")
var virtualMachineHardReboot = pflag.Bool("hard-reboot", false, "sends a VM into hard-reset mode. This is damaging to all ongoing file operations. Returns result with a current machine state")
var virtualMachineShutdown = pflag.Bool("shutdown", false, "gracefully shuts down the VM. Returns result with a current machine state")
var virtualMachineShutoff = pflag.Bool("shutoff", false, "kills running VM. Equivalent to pulling a plug out of a computer. Returns result with a current machine state")
var virtualMachineStart = pflag.Bool("start", false, "starts up a VM. Returns result with a current machine state")
var virtualMachinePause = pflag.Bool("pause", false, "stops the execution of the VM. CPU is not used, but memory is still occupied. Returns result with a current machine state")
var virtualMachineResume = pflag.Bool("resume", false, "called after Pause, to resume the invocation of the VM. Returns result with a current machine state")
var virtualMachineCreate = pflag.Bool("create", false, "creates a new machine. Requires --xml-template parameter. Returns result with a current machine state")
var virtualMachineDelete = pflag.Bool("delete", false, "deletes an existing machine.")
var virtualMachinesIps = pflag.Bool("ips", false, "show ip addresses vm on host.")
var virtualMachinesStateAll = pflag.Bool("show-all", false, "show status all vms on host.")

var vm = pflag.String("vm", "", "vm of the machine to work with")
var xmlTemplate = pflag.String("xml-template", "", "path to an xml template file that describes a machine. See qemu docs on xml templates.")

var libvirtInstance *libvirt.Connect

// TODO: cool things you can do with Domain, but do not know how to:
// virDomainInterfaceAddresses - gets data about an IP addresses on a current interfaces. Mega-tool.
// virDomainGetGuestInfo - full data about a config of the guest OS
// virDomainGetState - provides the data about an actual domain state. Why is it shutoff or hybernating. Requires copious amount of magic fuckery to find out the actual reason with multiplication and matrix transforms, but can be translated into a redable form.
func main() {

	pflag.Parse()

	LibvirtInit()
	defer libvirtInstance.Close()

	switch {
	case *virtualMachineState:
		VirtualMachineState(*vm)
	case *virtualMachineSoftReboot:
		VirtualMachineSoftReboot(*vm)
	case *virtualMachineHardReboot:
		VirtualMachineHardReboot(*vm)
	case *virtualMachineShutdown:
		VirtualMachineShutdown(*vm)
	case *virtualMachineShutoff:
		VirtualMachineShutoff(*vm)
	case *virtualMachineStart:
		VirtualMachineStart(*vm)
	case *virtualMachinePause:
		VirtualMachinePause(*vm)
	case *virtualMachineResume:
		VirtualMachineResume(*vm)
	case *virtualMachineCreate:
		VirtualMachineCreate(*xmlTemplate)
	case *virtualMachineDelete:
		VirtualMachineDelete(*vm)
	case *virtualMachinesIps:
		VirtualMachinesIps()
	case *virtualMachinesStateAll:
		VirtualMachinesStateAll()
	}
}

// VirtualMachineState returns current state of a virtual machine.
func VirtualMachineState(vm string) {
	ret := GetVirtualMachineStateInfo(vm)
	hret(ret)
}

// VirtualMachineCreate creates a new VM from an xml template file
func VirtualMachineCreate(xmlTemplate string) {

	xml, err := os.ReadFile(xmlTemplate)
	herr(err)

	d, err := libvirtInstance.DomainDefineXML(string(xml))
	herr(err)

	hret(d)
}

// VirtualMachineDelete deletes a new VM from an xml template file
func VirtualMachineDelete(vm string) {
	d, err := libvirtInstance.LookupDomainByName(vm)
	herr(err)

	err = d.UndefineFlags(libvirt.DOMAIN_UNDEFINE_KEEP_NVRAM)
	herr(err)
	hok(fmt.Sprintf("%v was deleted", vm))
}

// VirtualMachineSoftReboot reboots a machine gracefully, as chosen by hypervisor.
func VirtualMachineSoftReboot(vm string) {
	d, err := libvirtInstance.LookupDomainByName(vm)
	herr(err)

	err = d.Reboot(libvirt.DOMAIN_REBOOT_DEFAULT)
	herr(err)

	hok(fmt.Sprintf("%v was soft-rebooted successfully", vm))
}

// VirtualMachineHardReboot sends a VM into hard-reset mode. This is damaging to all ongoing file operations.
func VirtualMachineHardReboot(vm string) {
	d, err := libvirtInstance.LookupDomainByName(vm)
	herr(err)

	err = d.Reset(0)
	herr(err)

	hok(fmt.Sprintf("%v was hard-rebooted successfully", vm))
}

// VirtualMachineShutdown gracefully shuts down the VM.
func VirtualMachineShutdown(vm string) {
	d, err := libvirtInstance.LookupDomainByName(vm)
	herr(err)

	err = d.Shutdown()
	herr(err)

	hok(fmt.Sprintf("%v was shutdown successfully", vm))
}

// VirtualMachineShutoff kills running VM. Equivalent to pulling a plug out of a computer.
func VirtualMachineShutoff(vm string) {
	d, err := libvirtInstance.LookupDomainByName(vm)
	herr(err)

	err = d.Destroy()
	herr(err)

	hok(fmt.Sprintf("%v was shutoff successfully", vm))
}

// VirtualMachineStart starts up a VM.
func VirtualMachineStart(vm string) {
	d, err := libvirtInstance.LookupDomainByName(vm)
	herr(err)

	//v.DomainRestore()
	//_, err = v.DomainCreateWithFlags(d, uint32(libvirt.DomainStartBypassCache))
	err = d.Create()
	herr(err)

	hok(fmt.Sprintf("%v was started", vm))
}

// VirtualMachinePause stops the execution of the VM. CPU is not used, but memory is still occupied.
func VirtualMachinePause(vm string) {
	d, err := libvirtInstance.LookupDomainByName(vm)
	herr(err)

	err = d.Suspend()
	herr(err)

	hok(fmt.Sprintf("%v is paused", vm))
}

// VirtualMachineResume can be called after Pause, to resume the invocation of the VM.
func VirtualMachineResume(vm string) {
	d, err := libvirtInstance.LookupDomainByName(vm)
	herr(err)

	err = d.Resume()
	herr(err)

	hok(fmt.Sprintf("%v was resumed", vm))
}

func VirtualMachinesIps() {

	var OutputString strings.Builder

	AllDomains, err := libvirtInstance.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_RUNNING)
	herr(err)
	// fmt.Println(AllDomains)
	fmt.Fprintf(&OutputString, "There are %d running domains:\n", len(AllDomains))

	for _, domain := range AllDomains {
		DomainName, err := domain.GetName()
		fmt.Fprintf(&OutputString, "Domain - %s:\n", DomainName)
		herr(err)

		AllDomainInterfaces, err := domain.ListAllInterfaceAddresses(libvirt.DOMAIN_INTERFACE_ADDRESSES_SRC_AGENT)
		herr(err)
		// fmt.Printf("All interfaces for domain %s - %v, Type - %T\n", DomainName, AllDomainInterfaces, AllDomainInterfaces)
		for _, DomainInterfaceEntry := range AllDomainInterfaces {
		    fmt.Fprintf(&OutputString, "interface - %v, address - ", DomainInterfaceEntry.Name)
		    for _, val := range DomainInterfaceEntry.Addrs {
				fmt.Fprintf(&OutputString, val.Addr)
				fmt.Fprintf(&OutputString, " ")
			}
			fmt.Fprintf(&OutputString,"\n")
			// OutputString := fmt.Sprintf("interface - %v, address - %v", DomainInterfaceEntry.Name, AllAddrs.String())
			// fmt.Printf("Domain - %s, interface - %v, ip address - %v\n",
			//   DomainName, DomainInterfaceEntry.Name, DomainInterfaceEntry.Addrs[0].Addr)
		}
		domain.Free()
	}
	fmt.Print(OutputString.String())
}

func VirtualMachinesStateAll() {
	AllDomainsActive, err := libvirtInstance.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_ACTIVE)
	herr(err)
	AllDomainsInactiv, err := libvirtInstance.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_INACTIVE)
	herr(err)

	OutputString := fmt.Sprintf("There are %d domains: %d active and %d inactive",
		len(AllDomainsActive)+len(AllDomainsInactiv), len(AllDomainsActive), len(AllDomainsInactiv))
	fmt.Println(OutputString)
	PrintVirtualMachinesStatus(AllDomainsActive)
	PrintVirtualMachinesStatus(AllDomainsInactiv)

}

func PrintVirtualMachinesStatus(domains []libvirt.Domain) {
	for _, domain := range domains {
		DomainName, err := domain.GetName()
		herr(err)
		VmState := GetVirtualMachineStateInfo(DomainName)
		fmt.Printf("%-30v %-15v\n", DomainName, VmState.State)
	}
}

func GetVirtualMachineStateInfo(vm string) (info VirtualMachineStateInfo) {

	var VmStateInfo VirtualMachineStateInfo

	d, err := libvirtInstance.LookupDomainByName(vm)
	herr(err)

	dominfo, err := d.GetInfo()
	herr(err)

	VmStateInfo.CpuCount = dominfo.NrVirtCpu
	VmStateInfo.CpuTime = dominfo.CpuTime
	// god only knows why they return memory in kilobytes.
	VmStateInfo.MemoryBytes = dominfo.Memory * 1024
	VmStateInfo.MaxMemoryBytes = dominfo.MaxMem * 1024

	switch dominfo.State {
	case libvirt.DOMAIN_NOSTATE:
		VmStateInfo.State = VirtStatePending
	case libvirt.DOMAIN_RUNNING:
		VmStateInfo.State = VirtStateRunning
	case libvirt.DOMAIN_BLOCKED:
		VmStateInfo.State = VirtStateBlocked
	case libvirt.DOMAIN_PAUSED:
		VmStateInfo.State = VirtStatePaused
	case libvirt.DOMAIN_SHUTDOWN:
		VmStateInfo.State = VirtStateShutdown
	case libvirt.DOMAIN_SHUTOFF:
		VmStateInfo.State = VirtStateShutoff
	case libvirt.DOMAIN_CRASHED:
		VmStateInfo.State = VirtStateCrashed
	case libvirt.DOMAIN_PMSUSPENDED:
		VmStateInfo.State = VirtStateHybernating
	}

	return VmStateInfo
}

func LibvirtInit() {
	var err error
	libvirtInstance, err = libvirt.NewConnect("qemu:///system")
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
}

func herr(e error) {
	if e != nil {
		fmt.Printf("%v\n", strings.ReplaceAll(e.Error(), "\"", ""))
		// os.Exit(1)
	}
}

func hok(message string) {
	fmt.Printf(`{"ok":"%v"}`, strings.ReplaceAll(message, "\"", ""))
	os.Exit(0)
}

func hret(i any) {
	ret, err := json.Marshal(i)
	herr(err)
	fmt.Print(string(ret))
	os.Exit(0)
}
