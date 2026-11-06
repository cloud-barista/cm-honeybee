package controller

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	"github.com/cloud-barista/cm-honeybee/server/lib/spider"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
)

// fakeNetwork stands in for a driver. getVPC/getSG answer by SystemId only, as
// the aws, alibaba and ncp drivers' GetVPC/GetSecurity do; leaving them nil
// makes Get fail so the listing fallback is exercised.
type fakeNetwork struct {
	vpcs     []spider.VPCInfo
	sgs      []spider.SecurityGroupInfo
	getWorks bool
	listErr  error
	vpcGets  int
	vpcLists int
	sgGets   int
	sgLists  int
}

func (f *fakeNetwork) lookup() cspNetworkLookup {
	return cspNetworkLookup{
		GetVPC: func(iid spider.IID) (*spider.VPCInfo, error) {
			f.vpcGets++
			if !f.getWorks {
				return nil, errors.New("get not supported")
			}
			for i := range f.vpcs {
				if iid.SystemId != "" && f.vpcs[i].IId.SystemId == iid.SystemId {
					return &f.vpcs[i], nil
				}
			}
			return nil, errors.New("vpc not found")
		},
		ListVPC: func() ([]spider.VPCInfo, error) {
			f.vpcLists++
			return f.vpcs, f.listErr
		},
		GetSecurityGroup: func(iid spider.IID) (*spider.SecurityGroupInfo, error) {
			f.sgGets++
			if !f.getWorks {
				return nil, errors.New("get not supported")
			}
			for i := range f.sgs {
				if iid.SystemId != "" && f.sgs[i].IId.SystemId == iid.SystemId {
					return &f.sgs[i], nil
				}
			}
			return nil, errors.New("sg not found")
		},
		ListSecurityGroup: func() ([]spider.SecurityGroupInfo, error) {
			f.sgLists++
			return f.sgs, f.listErr
		},
	}
}

var testRule = spider.SecurityRuleInfo{Direction: "inbound", IPProtocol: "tcp", FromPort: "22", ToPort: "22", CIDR: "0.0.0.0/0"}

func systemOnlyVPC() spider.VPCInfo {
	return spider.VPCInfo{
		IId:       spider.IID{SystemId: "vpc-08adcb09d79c4ed82"},
		IPv4_CIDR: "198.51.100.0/24",
		SubnetInfoList: []spider.SubnetInfo{
			{IId: spider.IID{SystemId: "subnet-other"}, IPv4_CIDR: "198.51.100.128/26", Zone: "ap-northeast-2c"},
			{IId: spider.IID{SystemId: "subnet-0aa"}, IPv4_CIDR: "198.51.100.0/26", Zone: "ap-northeast-2a"},
		},
	}
}

// systemOnlyVM is a VM as the aws/alibaba/ncp drivers report it: VPC, subnet,
// image, data disk and security group IIDs carry a SystemId and no NameId.
func systemOnlyVM() *spider.VMInfo {
	return &spider.VMInfo{
		IId:               spider.IID{NameId: "vm-1", SystemId: "i-0123"},
		ImageIId:          spider.IID{SystemId: "m-image"},
		VpcIID:            spider.IID{SystemId: "vpc-08adcb09d79c4ed82"},
		SubnetIID:         spider.IID{SystemId: "subnet-0aa"},
		SecurityGroupIIds: []spider.IID{{SystemId: "sg-1"}},
		DataDiskIIDs:      []spider.IID{{SystemId: "vol-0427e4e463804e007"}},
	}
}

func systemOnlySG() spider.SecurityGroupInfo {
	return spider.SecurityGroupInfo{IId: spider.IID{SystemId: "sg-1"}, SecurityRules: []spider.SecurityRuleInfo{testRule}}
}

var wantSystemOnlyNetwork = infra.CSPNetwork{
	VPC: infra.CSPVPC{
		Name: "vpc-08adcb09d79c4ed82",
		CIDR: "198.51.100.0/24",
		Subnets: []infra.CSPSubnet{
			{Name: "subnet-other", CIDR: "198.51.100.128/26", Zone: "ap-northeast-2c"},
			{Name: "subnet-0aa", CIDR: "198.51.100.0/26", Zone: "ap-northeast-2a"},
		},
	},
	Subnet: "subnet-0aa",
	SecurityGroups: []infra.CSPSecurityGroup{{
		Name:  "sg-1",
		Rules: []infra.CSPSecurityRule{{Direction: "inbound", Protocol: "tcp", FromPort: "22", ToPort: "22", CIDR: "0.0.0.0/0"}},
	}},
}

func TestBuildCSPInfoSystemIdOnly(t *testing.T) {
	sg := &model.SourceGroup{ProviderName: "AWS"}

	t.Run("driver Get by IID, no listing", func(t *testing.T) {
		f := &fakeNetwork{vpcs: []spider.VPCInfo{systemOnlyVPC()}, sgs: []spider.SecurityGroupInfo{systemOnlySG()}, getWorks: true}
		got := buildCSPInfoWith(f.lookup(), sg, systemOnlyVM())

		if !reflect.DeepEqual(got.Network, wantSystemOnlyNetwork) {
			t.Errorf("network =\n%+v\nwant\n%+v", got.Network, wantSystemOnlyNetwork)
		}
		if got.Image != "m-image" {
			t.Errorf("image = %q, want m-image", got.Image)
		}
		if !reflect.DeepEqual(got.DataDisks, []string{"vol-0427e4e463804e007"}) {
			t.Errorf("data disks = %q", got.DataDisks)
		}
		if f.vpcLists != 0 || f.sgLists != 0 {
			t.Errorf("listed vpc %d / sg %d times, want 0 when Get answers", f.vpcLists, f.sgLists)
		}
	})

	t.Run("Get fails, listing matched by SystemId", func(t *testing.T) {
		f := &fakeNetwork{vpcs: []spider.VPCInfo{systemOnlyVPC()}, sgs: []spider.SecurityGroupInfo{systemOnlySG()}}
		got := buildCSPInfoWith(f.lookup(), sg, systemOnlyVM())

		if !reflect.DeepEqual(got.Network, wantSystemOnlyNetwork) {
			t.Errorf("network =\n%+v\nwant\n%+v", got.Network, wantSystemOnlyNetwork)
		}
	})

	t.Run("Get and listing both fail, identifiers kept", func(t *testing.T) {
		f := &fakeNetwork{listErr: errors.New("driver call panicked")}
		got := buildCSPInfoWith(f.lookup(), sg, systemOnlyVM())

		want := infra.CSPNetwork{
			VPC:            infra.CSPVPC{Name: "vpc-08adcb09d79c4ed82"},
			Subnet:         "subnet-0aa",
			SecurityGroups: []infra.CSPSecurityGroup{{Name: "sg-1"}},
		}
		if !reflect.DeepEqual(got.Network, want) {
			t.Errorf("network =\n%+v\nwant\n%+v", got.Network, want)
		}
	})
}

// A VM whose IIDs carry a NameId keeps reporting that name, as before.
func TestBuildCSPInfoNameIdKept(t *testing.T) {
	vpc := spider.VPCInfo{
		IId:            spider.IID{NameId: "my-vpc", SystemId: "vpc-1"},
		IPv4_CIDR:      "203.0.113.0/24",
		SubnetInfoList: []spider.SubnetInfo{{IId: spider.IID{NameId: "my-subnet", SystemId: "subnet-1"}, IPv4_CIDR: "203.0.113.0/26"}},
	}
	sgInfo := spider.SecurityGroupInfo{IId: spider.IID{NameId: "my-sg", SystemId: "sg-1"}, SecurityRules: []spider.SecurityRuleInfo{testRule}}
	vm := &spider.VMInfo{
		ImageIId:          spider.IID{NameId: "ubuntu", SystemId: "img-1"},
		VpcIID:            spider.IID{NameId: "my-vpc", SystemId: "vpc-1"},
		SubnetIID:         spider.IID{NameId: "my-subnet", SystemId: "subnet-1"},
		SecurityGroupIIds: []spider.IID{{NameId: "my-sg", SystemId: "sg-1"}},
		DataDiskIIDs:      []spider.IID{{NameId: "disk-a", SystemId: "vol-a"}},
	}

	for _, getWorks := range []bool{true, false} {
		f := &fakeNetwork{vpcs: []spider.VPCInfo{vpc}, sgs: []spider.SecurityGroupInfo{sgInfo}, getWorks: getWorks}
		got := buildCSPInfoWith(f.lookup(), &model.SourceGroup{ProviderName: "NCP"}, vm)

		if got.Network.VPC.Name != "my-vpc" || got.Network.VPC.CIDR != "203.0.113.0/24" || got.Network.Subnet != "my-subnet" {
			t.Errorf("getWorks=%v: vpc/subnet = %+v / %q", getWorks, got.Network.VPC, got.Network.Subnet)
		}
		if len(got.Network.SecurityGroups) != 1 || got.Network.SecurityGroups[0].Name != "my-sg" || len(got.Network.SecurityGroups[0].Rules) != 1 {
			t.Errorf("getWorks=%v: security groups = %+v", getWorks, got.Network.SecurityGroups)
		}
		if got.Image != "ubuntu" || !reflect.DeepEqual(got.DataDisks, []string{"disk-a"}) {
			t.Errorf("getWorks=%v: image %q, disks %q", getWorks, got.Image, got.DataDisks)
		}
	}
}
