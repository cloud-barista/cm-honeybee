package controller

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	"github.com/cloud-barista/cm-honeybee/server/lib/spider"
)

// detailVM is a VM as the aws driver reports it: the key pair carries a
// SystemId and NameId, data disks a SystemId only.
func detailVM() *spider.VMInfo {
	return &spider.VMInfo{
		IId:          spider.IID{NameId: "vm-1", SystemId: "i-0abc"},
		VMSpecName:   "t3a.small",
		ImageIId:     spider.IID{SystemId: "ami-0216"},
		KeyPairIId:   spider.IID{NameId: "tb4kkepltpgrcgleb48v", SystemId: "tb4kkepltpgrcgleb48v"},
		DataDiskIIDs: []spider.IID{{SystemId: "vol-1"}, {SystemId: "vol-2"}},
	}
}

func workingDetailLookup() cspDetailLookup {
	return cspDetailLookup{
		GetVMStatus: func(spider.IID) (string, error) { return "Running", nil },
		GetKeyPair: func(iid spider.IID) (*spider.KeyPairInfo, error) {
			return &spider.KeyPairInfo{IId: iid, Fingerprint: "3b:16:bf"}, nil
		},
		GetDisk: func(iid spider.IID) (*spider.DiskInfo, error) {
			return &spider.DiskInfo{IId: spider.IID{NameId: "name-" + iid.SystemId, SystemId: iid.SystemId}, DiskType: "gp2", DiskSize: "20", Status: "Attached"}, nil
		},
		GetVMSpec: func(name string) (*spider.VMSpecInfo, error) {
			return &spider.VMSpecInfo{
				Name:       name,
				VCpu:       spider.VCpuInfo{Count: "2", ClockGHz: "2.2"},
				MemSizeMiB: "2048",
				DiskSizeGB: "-1",
				Gpu:        []spider.GpuInfo{{Count: "1", Mfr: "NVIDIA", Model: "T4", MemSizeGB: "16"}},
				KeyValueList: []spider.KeyValue{
					{Key: "ProcessorInfo", Value: "{Manufacturer:AMD,SupportedArchitectures:[i386; x86_64],SustainedClockSpeedInGhz:2.2}"},
				},
			}, nil
		},
		GetImage: func(iid spider.IID) (*spider.ImageInfo, error) {
			return &spider.ImageInfo{
				IId: iid, Name: "ubuntu-jammy", OSArchitecture: "x86_64", OSPlatform: "Linux/UNIX",
				OSDistribution: "Ubuntu 22.04", OSDiskType: "NA", OSDiskSizeGB: "8", ImageStatus: "Available",
			}, nil
		},
	}
}

func TestAddCSPDetailFillsEveryField(t *testing.T) {
	var got infra.CSPInfo
	addCSPDetail(workingDetailLookup(), detailVM(), &got)

	want := infra.CSPInfo{
		Status:  "Running",
		KeyPair: &infra.CSPKeyPair{Name: "tb4kkepltpgrcgleb48v", ID: "tb4kkepltpgrcgleb48v", Fingerprint: "3b:16:bf"},
		DataDiskDetails: []infra.CSPDataDisk{
			{Name: "name-vol-1", ID: "vol-1", Size: 20, Type: "gp2", Status: "Attached"},
			{Name: "name-vol-2", ID: "vol-2", Size: 20, Type: "gp2", Status: "Attached"},
		},
		VMSpecDetail: &infra.CSPVMSpec{
			Name: "t3a.small", VCPUs: 2, MemoryMiB: 2048, DiskGB: 0, Architecture: "x86_64",
			GPUs: []infra.CSPGPU{{Count: 1, Vendor: "NVIDIA", Model: "T4", MemoryGB: 16}},
		},
		ImageDetail: &infra.CSPImage{
			Name: "ubuntu-jammy", ID: "ami-0216", OSDistribution: "Ubuntu 22.04", OSArchitecture: "x86_64",
			OSPlatform: "Linux/UNIX", OSDiskSizeGB: 8, Status: "Available",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got  %+v\nwant %+v", got, want)
	}
}

// A failing handler leaves only its own field empty; the key pair keeps the
// name the VM reported.
func TestAddCSPDetailHandlerFailures(t *testing.T) {
	failed := errors.New("boom")
	lookup := workingDetailLookup()
	lookup.GetKeyPair = func(spider.IID) (*spider.KeyPairInfo, error) { return nil, failed }
	lookup.GetDisk = func(iid spider.IID) (*spider.DiskInfo, error) {
		if iid.SystemId == "vol-1" {
			return nil, failed
		}
		return &spider.DiskInfo{IId: iid, DiskType: "gp2", DiskSize: "default", Status: "Attached"}, nil
	}
	lookup.GetImage = func(spider.IID) (*spider.ImageInfo, error) { return nil, failed }

	var got infra.CSPInfo
	addCSPDetail(lookup, detailVM(), &got)

	if got.Status != "Running" {
		t.Errorf("status = %q", got.Status)
	}
	if got.KeyPair == nil || got.KeyPair.Name != "tb4kkepltpgrcgleb48v" || got.KeyPair.Fingerprint != "" {
		t.Errorf("keypair = %+v", got.KeyPair)
	}
	wantDisks := []infra.CSPDataDisk{{Name: "vol-2", ID: "vol-2", Size: 0, Type: "gp2", Status: "Attached"}}
	if !reflect.DeepEqual(got.DataDiskDetails, wantDisks) {
		t.Errorf("disks = %+v", got.DataDiskDetails)
	}
	if got.VMSpecDetail == nil || got.VMSpecDetail.VCPUs != 2 {
		t.Errorf("vm spec = %+v", got.VMSpecDetail)
	}
	if got.ImageDetail != nil {
		t.Errorf("image = %+v, want nil", got.ImageDetail)
	}

	lookup = workingDetailLookup()
	lookup.GetVMStatus = func(spider.IID) (string, error) { return "", failed }
	lookup.GetVMSpec = func(string) (*spider.VMSpecInfo, error) { return nil, failed }
	got = infra.CSPInfo{}
	addCSPDetail(lookup, detailVM(), &got)
	if got.Status != "" || got.VMSpecDetail != nil {
		t.Errorf("status = %q, vm spec = %+v, want both empty", got.Status, got.VMSpecDetail)
	}
	if got.KeyPair == nil || got.KeyPair.Fingerprint == "" || len(got.DataDiskDetails) != 2 || got.ImageDetail == nil {
		t.Errorf("other fields lost: %+v", got)
	}
}

// A driver without a handler (nil func) is not asked.
func TestAddCSPDetailNoCapability(t *testing.T) {
	var got infra.CSPInfo
	addCSPDetail(cspDetailLookup{}, detailVM(), &got)

	want := infra.CSPInfo{KeyPair: &infra.CSPKeyPair{Name: "tb4kkepltpgrcgleb48v", ID: "tb4kkepltpgrcgleb48v"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestSpecArchitecture(t *testing.T) {
	cases := []struct {
		kvs  []spider.KeyValue
		want string
	}{
		{[]spider.KeyValue{{Key: "ProcessorInfo", Value: "{SupportedArchitectures:[arm64],SustainedClockSpeedInGhz:2.5}"}}, "arm64"},
		{[]spider.KeyValue{{Key: "Architecture", Value: "X86_64"}}, "x86_64"},
		{[]spider.KeyValue{{Key: "CpuArchitecture", Value: "X86"}}, "x86"},
		{[]spider.KeyValue{{Key: "VcpuArchitecture", Value: "{default:amd64,type:fixed,value:amd64}"}}, "amd64"},
		{[]spider.KeyValue{{Key: "Name", Value: "x86_64"}}, ""},
		{nil, ""},
	}
	for _, c := range cases {
		if got := specArchitecture(c.kvs); got != c.want {
			t.Errorf("specArchitecture(%v) = %q, want %q", c.kvs, got, c.want)
		}
	}
}
