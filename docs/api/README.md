# API 가이드


## API 작성 규칙 
[cm-beetle Useful samples to add new APIs 참고](https://github.com/cloud-barista/cm-beetle/blob/main/docs/useful-samples-to-add-new-apis.md)

## Using APIs

### Get infra information
```
http://127.0.0.1:8082/infra
```

<details>
<summary>Reply example</summary>

```
{
    "compute": {
        "os": {
            "os": {
                "name": "linux",
                "vendor": "ubuntu",
                "version": "23.10 (Mantic Minotaur)",
                "release": "23.10",
                "architecture": "x86_64"
            },
            "kernel": {
                "release": "6.5.0-14-generic",
                "version": "#14-Ubuntu SMP PREEMPT_DYNAMIC Tue Nov 14 14:59:49 UTC 2023",
                "architecture": "x86_64"
            },
            "node": {
                "hostname": "ish",
                "hypervisor": "",
                "machineid": "032e02b4-0499-054d-2506-190700080009",
                "timezone": "KST"
            }
        },
        "compute_resource": {
            "cpu": {
                "vendor": "GenuineIntel",
                "model": "Intel(R) Core(TM) i7-9700K CPU @ 3.60GHz",
                "speed": 4900,
                "cache": 12288,
                "cpus": 1,
                "cores": 8,
                "threads": 8
            },
            "memory": {
                "type": "DDR4",
                "speed": 3200,
                "size": 32768
            },
            "storage": [
                {
                    "name": "nvme0n1",
                    "driver": "",
                    "vendor": "",
                    "model": "WDC WDS250G1B0C-00S6U0",
                    "serial": "190704642503",
                    "size": 250
                },
                {
                    "name": "sda",
                    "driver": "sd",
                    "vendor": "ATA",
                    "model": "SanDisk SD8SBAT1",
                    "serial": "160561411700",
                    "size": 128
                },
                {
                    "name": "sdb",
                    "driver": "sd",
                    "vendor": "ATA",
                    "model": "ST1000DM003-1SB1",
                    "serial": "Z9A16CG3",
                    "size": 1000
                }
            ]
        }
    },
    "network": {
        "network_subsystem": {
            "network_interfaces": [
                {
                    "interface": "lo",
                    "address": [
                        "127.0.0.1/8",
                        "::1/128"
                    ],
                    "gateway": null,
                    "route": null,
                    "mac": "",
                    "mtu": 65536
                },
                {
                    "interface": "enp7s0",
                    "address": [
                        "192.168.110.14/24",
                        "fe80::d338:92b5:df08:778/64"
                    ],
                    "gateway": [
                        "192.168.110.254"
                    ],
                    "route": [
                        {
                            "destination": "0.0.0.0",
                            "netmask": "0.0.0.0",
                            "next_hop": "192.168.110.254"
                        },
                        {
                            "destination": "169.254.0.0",
                            "netmask": "255.255.0.0",
                            "next_hop": "on-link"
                        },
                        {
                            "destination": "192.168.110.0",
                            "netmask": "255.255.255.0",
                            "next_hop": "on-link"
                        }
                    ],
                    "mac": "b4:2e:99:4d:25:19",
                    "mtu": 1500
                },
                {
                    "interface": "virbr0",
                    "address": [
                        "192.168.122.254/24"
                    ],
                    "gateway": null,
                    "route": [
                        {
                            "destination": "192.168.122.0",
                            "netmask": "255.255.255.0",
                            "next_hop": "on-link"
                        }
                    ],
                    "mac": "52:54:00:95:b0:d2",
                    "mtu": 1500
                },
                {
                    "interface": "br-9cc4698f0d3c",
                    "address": [
                        "192.168.64.1/20",
                        "fe80::42:75ff:fe6e:ca0/64"
                    ],
                    "gateway": null,
                    "route": [
                        {
                            "destination": "192.168.64.0",
                            "netmask": "255.255.240.0",
                            "next_hop": "on-link"
                        }
                    ],
                    "mac": "02:42:75:6e:0c:a0",
                    "mtu": 1500
                },
                {
                    "interface": "docker0",
                    "address": [
                        "172.17.0.1/16",
                        "2001:db8:1::1/64",
                        "fe80::1/64"
                    ],
                    "gateway": null,
                    "route": [
                        {
                            "destination": "172.17.0.0",
                            "netmask": "255.255.0.0",
                            "next_hop": "on-link"
                        }
                    ],
                    "mac": "02:42:f3:6b:91:3c",
                    "mtu": 1500
                },
                {
                    "interface": "br-b54c7f32f3db",
                    "address": [
                        "172.23.0.1/16"
                    ],
                    "gateway": null,
                    "route": [
                        {
                            "destination": "172.23.0.0",
                            "netmask": "255.255.0.0",
                            "next_hop": "on-link"
                        }
                    ],
                    "mac": "02:42:6a:96:d6:d0",
                    "mtu": 1500
                },
                {
                    "interface": "br-bdfda54c6284",
                    "address": [
                        "172.22.0.1/16"
                    ],
                    "gateway": null,
                    "route": [
                        {
                            "destination": "172.22.0.0",
                            "netmask": "255.255.0.0",
                            "next_hop": "on-link"
                        }
                    ],
                    "mac": "02:42:91:b9:e3:c8",
                    "mtu": 1500
                },
                {
                    "interface": "br-451ab35505ac",
                    "address": [
                        "172.20.0.1/16"
                    ],
                    "gateway": null,
                    "route": [
                        {
                            "destination": "172.20.0.0",
                            "netmask": "255.255.0.0",
                            "next_hop": "on-link"
                        }
                    ],
                    "mac": "02:42:d8:a4:c6:10",
                    "mtu": 1500
                },
                {
                    "interface": "br-4d1d043f596f",
                    "address": [
                        "172.18.0.1/16",
                        "fc00:f853:ccd:e793::1/64",
                        "fe80::1/64"
                    ],
                    "gateway": null,
                    "route": [
                        {
                            "destination": "172.18.0.0",
                            "netmask": "255.255.0.0",
                            "next_hop": "on-link"
                        }
                    ],
                    "mac": "02:42:c4:0d:82:89",
                    "mtu": 1500
                },
                {
                    "interface": "br-7e13d39099f0",
                    "address": [
                        "172.25.0.1/16"
                    ],
                    "gateway": null,
                    "route": [
                        {
                            "destination": "172.25.0.0",
                            "netmask": "255.255.0.0",
                            "next_hop": "on-link"
                        }
                    ],
                    "mac": "02:42:c7:10:4e:da",
                    "mtu": 1500
                },
                {
                    "interface": "br-81fb56def3b3",
                    "address": [
                        "172.24.0.1/16"
                    ],
                    "gateway": null,
                    "route": [
                        {
                            "destination": "172.24.0.0",
                            "netmask": "255.255.0.0",
                            "next_hop": "on-link"
                        }
                    ],
                    "mac": "02:42:30:71:f3:f9",
                    "mtu": 1500
                },
                {
                    "interface": "veth9dd8a4f",
                    "address": [
                        "fe80::cb4:51ff:fe4d:688c/64"
                    ],
                    "gateway": null,
                    "route": null,
                    "mac": "0e:b4:51:4d:68:8c",
                    "mtu": 1500
                },
                {
                    "interface": "vethebba855",
                    "address": [
                        "fe80::b42b:d9ff:fe94:a663/64"
                    ],
                    "gateway": null,
                    "route": null,
                    "mac": "b6:2b:d9:94:a6:63",
                    "mtu": 1500
                },
                {
                    "interface": "vethe80df84",
                    "address": [
                        "fe80::dc0d:19ff:fe11:fce9/64"
                    ],
                    "gateway": null,
                    "route": null,
                    "mac": "de:0d:19:11:fc:e9",
                    "mtu": 1500
                },
                {
                    "interface": "veth8e77476",
                    "address": [
                        "fe80::f889:6fff:feb5:76ef/64"
                    ],
                    "gateway": null,
                    "route": null,
                    "mac": "fa:89:6f:b5:76:ef",
                    "mtu": 1500
                },
                {
                    "interface": "veth2e9cf65",
                    "address": [
                        "fe80::3cba:81ff:feaa:18b8/64"
                    ],
                    "gateway": null,
                    "route": null,
                    "mac": "3e:ba:81:aa:18:b8",
                    "mtu": 1500
                },
                {
                    "interface": "vethac154dc",
                    "address": [
                        "fe80::5444:adff:fe35:289b/64"
                    ],
                    "gateway": null,
                    "route": null,
                    "mac": "56:44:ad:35:28:9b",
                    "mtu": 1500
                }
            ],
            "routes": [
                {
                    "destination": "0.0.0.0",
                    "netmask": "0.0.0.0",
                    "next_hop": "192.168.110.254"
                },
                {
                    "destination": "169.254.0.0",
                    "netmask": "255.255.0.0",
                    "next_hop": "enp7s0"
                },
                {
                    "destination": "172.17.0.0",
                    "netmask": "255.255.0.0",
                    "next_hop": "docker0"
                },
                {
                    "destination": "172.18.0.0",
                    "netmask": "255.255.0.0",
                    "next_hop": "br-4d1d043f596f"
                },
                {
                    "destination": "172.20.0.0",
                    "netmask": "255.255.0.0",
                    "next_hop": "br-451ab35505ac"
                },
                {
                    "destination": "172.22.0.0",
                    "netmask": "255.255.0.0",
                    "next_hop": "br-bdfda54c6284"
                },
                {
                    "destination": "172.23.0.0",
                    "netmask": "255.255.0.0",
                    "next_hop": "br-b54c7f32f3db"
                },
                {
                    "destination": "172.24.0.0",
                    "netmask": "255.255.0.0",
                    "next_hop": "br-81fb56def3b3"
                },
                {
                    "destination": "172.25.0.0",
                    "netmask": "255.255.0.0",
                    "next_hop": "br-7e13d39099f0"
                },
                {
                    "destination": "192.168.64.0",
                    "netmask": "255.255.240.0",
                    "next_hop": "br-9cc4698f0d3c"
                },
                {
                    "destination": "192.168.110.0",
                    "netmask": "255.255.255.0",
                    "next_hop": "enp7s0"
                },
                {
                    "destination": "192.168.122.0",
                    "netmask": "255.255.255.0",
                    "next_hop": "virbr0"
                }
            ],
            "netfilter": {
                "ipv4_tables": [
                    {
                        "table_name": "filter",
                        "chains": [
                            {
                                "chain_name": "INPUT",
                                "rules": [
                                    "-P INPUT ACCEPT",
                                    "-A INPUT -j LIBVIRT_INP"
                                ]
                            },
                            {
                                "chain_name": "FORWARD",
                                "rules": [
                                    "-P FORWARD ACCEPT",
                                    "-A FORWARD -j DOCKER-USER",
                                    "-A FORWARD -j DOCKER-ISOLATION-STAGE-1",
                                    "-A FORWARD -o docker0 -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT",
                                    "-A FORWARD -o docker0 -j DOCKER",
                                    "-A FORWARD -i docker0 ! -o docker0 -j ACCEPT",
                                    "-A FORWARD -i docker0 -o docker0 -j ACCEPT",
                                    "-A FORWARD -o br-81fb56def3b3 -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT",
                                    "-A FORWARD -o br-81fb56def3b3 -j DOCKER",
                                    "-A FORWARD -i br-81fb56def3b3 ! -o br-81fb56def3b3 -j ACCEPT",
                                    "-A FORWARD -i br-81fb56def3b3 -o br-81fb56def3b3 -j ACCEPT",
                                    "-A FORWARD -o br-7e13d39099f0 -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT",
                                    "-A FORWARD -o br-7e13d39099f0 -j DOCKER",
                                    "-A FORWARD -i br-7e13d39099f0 ! -o br-7e13d39099f0 -j ACCEPT",
                                    "-A FORWARD -i br-7e13d39099f0 -o br-7e13d39099f0 -j ACCEPT",
                                    "-A FORWARD -o br-4d1d043f596f -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT",
                                    "-A FORWARD -o br-4d1d043f596f -j DOCKER",
                                    "-A FORWARD -i br-4d1d043f596f ! -o br-4d1d043f596f -j ACCEPT",
                                    "-A FORWARD -i br-4d1d043f596f -o br-4d1d043f596f -j ACCEPT",
                                    "-A FORWARD -o br-451ab35505ac -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT",
                                    "-A FORWARD -o br-451ab35505ac -j DOCKER",
                                    "-A FORWARD -i br-451ab35505ac ! -o br-451ab35505ac -j ACCEPT",
                                    "-A FORWARD -i br-451ab35505ac -o br-451ab35505ac -j ACCEPT",
                                    "-A FORWARD -o br-bdfda54c6284 -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT",
                                    "-A FORWARD -o br-bdfda54c6284 -j DOCKER",
                                    "-A FORWARD -i br-bdfda54c6284 ! -o br-bdfda54c6284 -j ACCEPT",
                                    "-A FORWARD -i br-bdfda54c6284 -o br-bdfda54c6284 -j ACCEPT",
                                    "-A FORWARD -o br-b54c7f32f3db -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT",
                                    "-A FORWARD -o br-b54c7f32f3db -j DOCKER",
                                    "-A FORWARD -i br-b54c7f32f3db ! -o br-b54c7f32f3db -j ACCEPT",
                                    "-A FORWARD -i br-b54c7f32f3db -o br-b54c7f32f3db -j ACCEPT",
                                    "-A FORWARD -o br-9cc4698f0d3c -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT",
                                    "-A FORWARD -o br-9cc4698f0d3c -j DOCKER",
                                    "-A FORWARD -i br-9cc4698f0d3c ! -o br-9cc4698f0d3c -j ACCEPT",
                                    "-A FORWARD -i br-9cc4698f0d3c -o br-9cc4698f0d3c -j ACCEPT",
                                    "-A FORWARD -j LIBVIRT_FWX",
                                    "-A FORWARD -j LIBVIRT_FWI",
                                    "-A FORWARD -j LIBVIRT_FWO"
                                ]
                            },
                            {
                                "chain_name": "OUTPUT",
                                "rules": [
                                    "-P OUTPUT ACCEPT",
                                    "-A OUTPUT -j LIBVIRT_OUT"
                                ]
                            },
                            {
                                "chain_name": "DOCKER",
                                "rules": [
                                    "-N DOCKER",
                                    "-A DOCKER -d 192.168.64.5/32 ! -i br-9cc4698f0d3c -o br-9cc4698f0d3c -p tcp -m tcp --dport 8080 -j ACCEPT"
                                ]
                            },
                            {
                                "chain_name": "DOCKER-ISOLATION-STAGE-1",
                                "rules": [
                                    "-N DOCKER-ISOLATION-STAGE-1",
                                    "-A DOCKER-ISOLATION-STAGE-1 -i docker0 ! -o docker0 -j DOCKER-ISOLATION-STAGE-2",
                                    "-A DOCKER-ISOLATION-STAGE-1 -i br-81fb56def3b3 ! -o br-81fb56def3b3 -j DOCKER-ISOLATION-STAGE-2",
                                    "-A DOCKER-ISOLATION-STAGE-1 -i br-7e13d39099f0 ! -o br-7e13d39099f0 -j DOCKER-ISOLATION-STAGE-2",
                                    "-A DOCKER-ISOLATION-STAGE-1 -i br-4d1d043f596f ! -o br-4d1d043f596f -j DOCKER-ISOLATION-STAGE-2",
                                    "-A DOCKER-ISOLATION-STAGE-1 -i br-451ab35505ac ! -o br-451ab35505ac -j DOCKER-ISOLATION-STAGE-2",
                                    "-A DOCKER-ISOLATION-STAGE-1 -i br-bdfda54c6284 ! -o br-bdfda54c6284 -j DOCKER-ISOLATION-STAGE-2",
                                    "-A DOCKER-ISOLATION-STAGE-1 -i br-b54c7f32f3db ! -o br-b54c7f32f3db -j DOCKER-ISOLATION-STAGE-2",
                                    "-A DOCKER-ISOLATION-STAGE-1 -i br-9cc4698f0d3c ! -o br-9cc4698f0d3c -j DOCKER-ISOLATION-STAGE-2",
                                    "-A DOCKER-ISOLATION-STAGE-1 -j RETURN"
                                ]
                            },
                            {
                                "chain_name": "DOCKER-ISOLATION-STAGE-2",
                                "rules": [
                                    "-N DOCKER-ISOLATION-STAGE-2",
                                    "-A DOCKER-ISOLATION-STAGE-2 -o docker0 -j DROP",
                                    "-A DOCKER-ISOLATION-STAGE-2 -o br-81fb56def3b3 -j DROP",
                                    "-A DOCKER-ISOLATION-STAGE-2 -o br-7e13d39099f0 -j DROP",
                                    "-A DOCKER-ISOLATION-STAGE-2 -o br-4d1d043f596f -j DROP",
                                    "-A DOCKER-ISOLATION-STAGE-2 -o br-451ab35505ac -j DROP",
                                    "-A DOCKER-ISOLATION-STAGE-2 -o br-bdfda54c6284 -j DROP",
                                    "-A DOCKER-ISOLATION-STAGE-2 -o br-b54c7f32f3db -j DROP",
                                    "-A DOCKER-ISOLATION-STAGE-2 -o br-9cc4698f0d3c -j DROP",
                                    "-A DOCKER-ISOLATION-STAGE-2 -j RETURN"
                                ]
                            },
                            {
                                "chain_name": "DOCKER-USER",
                                "rules": [
                                    "-N DOCKER-USER",
                                    "-A DOCKER-USER -j RETURN"
                                ]
                            },
                            {
                                "chain_name": "LIBVIRT_FWI",
                                "rules": [
                                    "-N LIBVIRT_FWI",
                                    "-A LIBVIRT_FWI -d 192.168.122.0/24 -o virbr0 -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT",
                                    "-A LIBVIRT_FWI -o virbr0 -j REJECT --reject-with icmp-port-unreachable"
                                ]
                            },
                            {
                                "chain_name": "LIBVIRT_FWO",
                                "rules": [
                                    "-N LIBVIRT_FWO",
                                    "-A LIBVIRT_FWO -s 192.168.122.0/24 -i virbr0 -j ACCEPT",
                                    "-A LIBVIRT_FWO -i virbr0 -j REJECT --reject-with icmp-port-unreachable"
                                ]
                            },
                            {
                                "chain_name": "LIBVIRT_FWX",
                                "rules": [
                                    "-N LIBVIRT_FWX",
                                    "-A LIBVIRT_FWX -i virbr0 -o virbr0 -j ACCEPT"
                                ]
                            },
                            {
                                "chain_name": "LIBVIRT_INP",
                                "rules": [
                                    "-N LIBVIRT_INP",
                                    "-A LIBVIRT_INP -i virbr0 -p udp -m udp --dport 53 -j ACCEPT",
                                    "-A LIBVIRT_INP -i virbr0 -p tcp -m tcp --dport 53 -j ACCEPT",
                                    "-A LIBVIRT_INP -i virbr0 -p udp -m udp --dport 67 -j ACCEPT",
                                    "-A LIBVIRT_INP -i virbr0 -p tcp -m tcp --dport 67 -j ACCEPT"
                                ]
                            },
                            {
                                "chain_name": "LIBVIRT_OUT",
                                "rules": [
                                    "-N LIBVIRT_OUT",
                                    "-A LIBVIRT_OUT -o virbr0 -p udp -m udp --dport 53 -j ACCEPT",
                                    "-A LIBVIRT_OUT -o virbr0 -p tcp -m tcp --dport 53 -j ACCEPT",
                                    "-A LIBVIRT_OUT -o virbr0 -p udp -m udp --dport 68 -j ACCEPT",
                                    "-A LIBVIRT_OUT -o virbr0 -p tcp -m tcp --dport 68 -j ACCEPT"
                                ]
                            }
                        ]
                    },
                    {
                        "table_name": "nat",
                        "chains": [
                            {
                                "chain_name": "PREROUTING",
                                "rules": [
                                    "-P PREROUTING ACCEPT",
                                    "-A PREROUTING -m addrtype --dst-type LOCAL -j DOCKER"
                                ]
                            },
                            {
                                "chain_name": "INPUT",
                                "rules": [
                                    "-P INPUT ACCEPT"
                                ]
                            },
                            {
                                "chain_name": "OUTPUT",
                                "rules": [
                                    "-P OUTPUT ACCEPT",
                                    "-A OUTPUT ! -d 127.0.0.0/8 -m addrtype --dst-type LOCAL -j DOCKER"
                                ]
                            },
                            {
                                "chain_name": "POSTROUTING",
                                "rules": [
                                    "-P POSTROUTING ACCEPT",
                                    "-A POSTROUTING -s 172.17.0.0/16 ! -o docker0 -j MASQUERADE",
                                    "-A POSTROUTING -s 172.24.0.0/16 ! -o br-81fb56def3b3 -j MASQUERADE",
                                    "-A POSTROUTING -s 172.25.0.0/16 ! -o br-7e13d39099f0 -j MASQUERADE",
                                    "-A POSTROUTING -s 172.18.0.0/16 ! -o br-4d1d043f596f -j MASQUERADE",
                                    "-A POSTROUTING -s 172.20.0.0/16 ! -o br-451ab35505ac -j MASQUERADE",
                                    "-A POSTROUTING -s 172.22.0.0/16 ! -o br-bdfda54c6284 -j MASQUERADE",
                                    "-A POSTROUTING -s 172.23.0.0/16 ! -o br-b54c7f32f3db -j MASQUERADE",
                                    "-A POSTROUTING -s 192.168.64.0/20 ! -o br-9cc4698f0d3c -j MASQUERADE",
                                    "-A POSTROUTING -j LIBVIRT_PRT",
                                    "-A POSTROUTING -s 192.168.64.5/32 -d 192.168.64.5/32 -p tcp -m tcp --dport 8080 -j MASQUERADE"
                                ]
                            },
                            {
                                "chain_name": "DOCKER",
                                "rules": [
                                    "-N DOCKER",
                                    "-A DOCKER -i docker0 -j RETURN",
                                    "-A DOCKER -i br-81fb56def3b3 -j RETURN",
                                    "-A DOCKER -i br-7e13d39099f0 -j RETURN",
                                    "-A DOCKER -i br-4d1d043f596f -j RETURN",
                                    "-A DOCKER -i br-451ab35505ac -j RETURN",
                                    "-A DOCKER -i br-bdfda54c6284 -j RETURN",
                                    "-A DOCKER -i br-b54c7f32f3db -j RETURN",
                                    "-A DOCKER -i br-9cc4698f0d3c -j RETURN",
                                    "-A DOCKER ! -i br-9cc4698f0d3c -p tcp -m tcp --dport 8080 -j DNAT --to-destination 192.168.64.5:8080"
                                ]
                            },
                            {
                                "chain_name": "LIBVIRT_PRT",
                                "rules": [
                                    "-N LIBVIRT_PRT",
                                    "-A LIBVIRT_PRT -s 192.168.122.0/24 -d 224.0.0.0/24 -j RETURN",
                                    "-A LIBVIRT_PRT -s 192.168.122.0/24 -d 255.255.255.255/32 -j RETURN",
                                    "-A LIBVIRT_PRT -s 192.168.122.0/24 ! -d 192.168.122.0/24 -p tcp -j MASQUERADE --to-ports 1024-65535",
                                    "-A LIBVIRT_PRT -s 192.168.122.0/24 ! -d 192.168.122.0/24 -p udp -j MASQUERADE --to-ports 1024-65535",
                                    "-A LIBVIRT_PRT -s 192.168.122.0/24 ! -d 192.168.122.0/24 -j MASQUERADE"
                                ]
                            }
                        ]
                    },
                    {
                        "table_name": "mangle",
                        "chains": [
                            {
                                "chain_name": "PREROUTING",
                                "rules": [
                                    "-P PREROUTING ACCEPT"
                                ]
                            },
                            {
                                "chain_name": "INPUT",
                                "rules": [
                                    "-P INPUT ACCEPT"
                                ]
                            },
                            {
                                "chain_name": "FORWARD",
                                "rules": [
                                    "-P FORWARD ACCEPT"
                                ]
                            },
                            {
                                "chain_name": "OUTPUT",
                                "rules": [
                                    "-P OUTPUT ACCEPT"
                                ]
                            },
                            {
                                "chain_name": "POSTROUTING",
                                "rules": [
                                    "-P POSTROUTING ACCEPT",
                                    "-A POSTROUTING -j LIBVIRT_PRT"
                                ]
                            },
                            {
                                "chain_name": "LIBVIRT_PRT",
                                "rules": [
                                    "-N LIBVIRT_PRT",
                                    "-A LIBVIRT_PRT -o virbr0 -p udp -m udp --dport 68 -j CHECKSUM --checksum-fill"
                                ]
                            }
                        ]
                    },
                    {
                        "table_name": "raw",
                        "chains": [
                            {
                                "chain_name": "PREROUTING",
                                "rules": [
                                    "-P PREROUTING ACCEPT"
                                ]
                            },
                            {
                                "chain_name": "OUTPUT",
                                "rules": [
                                    "-P OUTPUT ACCEPT"
                                ]
                            }
                        ]
                    },
                    {
                        "table_name": "security",
                        "chains": [
                            {
                                "chain_name": "INPUT",
                                "rules": [
                                    "-P INPUT ACCEPT"
                                ]
                            },
                            {
                                "chain_name": "FORWARD",
                                "rules": [
                                    "-P FORWARD ACCEPT"
                                ]
                            },
                            {
                                "chain_name": "OUTPUT",
                                "rules": [
                                    "-P OUTPUT ACCEPT"
                                ]
                            }
                        ]
                    }
                ],
                "ipv6_tables": [
                    {
                        "table_name": "filter",
                        "chains": [
                            {
                                "chain_name": "INPUT",
                                "rules": [
                                    "-P INPUT ACCEPT",
                                    "-A INPUT -j LIBVIRT_INP"
                                ]
                            },
                            {
                                "chain_name": "FORWARD",
                                "rules": [
                                    "-P FORWARD ACCEPT",
                                    "-A FORWARD -j LIBVIRT_FWX",
                                    "-A FORWARD -j LIBVIRT_FWI",
                                    "-A FORWARD -j LIBVIRT_FWO"
                                ]
                            },
                            {
                                "chain_name": "OUTPUT",
                                "rules": [
                                    "-P OUTPUT ACCEPT",
                                    "-A OUTPUT -j LIBVIRT_OUT"
                                ]
                            },
                            {
                                "chain_name": "LIBVIRT_FWI",
                                "rules": [
                                    "-N LIBVIRT_FWI"
                                ]
                            },
                            {
                                "chain_name": "LIBVIRT_FWO",
                                "rules": [
                                    "-N LIBVIRT_FWO"
                                ]
                            },
                            {
                                "chain_name": "LIBVIRT_FWX",
                                "rules": [
                                    "-N LIBVIRT_FWX"
                                ]
                            },
                            {
                                "chain_name": "LIBVIRT_INP",
                                "rules": [
                                    "-N LIBVIRT_INP"
                                ]
                            },
                            {
                                "chain_name": "LIBVIRT_OUT",
                                "rules": [
                                    "-N LIBVIRT_OUT"
                                ]
                            }
                        ]
                    },
                    {
                        "table_name": "nat",
                        "chains": [
                            {
                                "chain_name": "PREROUTING",
                                "rules": [
                                    "-P PREROUTING ACCEPT"
                                ]
                            },
                            {
                                "chain_name": "INPUT",
                                "rules": [
                                    "-P INPUT ACCEPT"
                                ]
                            },
                            {
                                "chain_name": "OUTPUT",
                                "rules": [
                                    "-P OUTPUT ACCEPT"
                                ]
                            },
                            {
                                "chain_name": "POSTROUTING",
                                "rules": [
                                    "-P POSTROUTING ACCEPT",
                                    "-A POSTROUTING -j LIBVIRT_PRT"
                                ]
                            },
                            {
                                "chain_name": "LIBVIRT_PRT",
                                "rules": [
                                    "-N LIBVIRT_PRT"
                                ]
                            }
                        ]
                    },
                    {
                        "table_name": "mangle",
                        "chains": [
                            {
                                "chain_name": "PREROUTING",
                                "rules": [
                                    "-P PREROUTING ACCEPT"
                                ]
                            },
                            {
                                "chain_name": "INPUT",
                                "rules": [
                                    "-P INPUT ACCEPT"
                                ]
                            },
                            {
                                "chain_name": "FORWARD",
                                "rules": [
                                    "-P FORWARD ACCEPT"
                                ]
                            },
                            {
                                "chain_name": "OUTPUT",
                                "rules": [
                                    "-P OUTPUT ACCEPT"
                                ]
                            },
                            {
                                "chain_name": "POSTROUTING",
                                "rules": [
                                    "-P POSTROUTING ACCEPT",
                                    "-A POSTROUTING -j LIBVIRT_PRT"
                                ]
                            },
                            {
                                "chain_name": "LIBVIRT_PRT",
                                "rules": [
                                    "-N LIBVIRT_PRT"
                                ]
                            }
                        ]
                    },
                    {
                        "table_name": "raw",
                        "chains": [
                            {
                                "chain_name": "PREROUTING",
                                "rules": [
                                    "-P PREROUTING ACCEPT"
                                ]
                            },
                            {
                                "chain_name": "OUTPUT",
                                "rules": [
                                    "-P OUTPUT ACCEPT"
                                ]
                            }
                        ]
                    },
                    {
                        "table_name": "security",
                        "chains": [
                            {
                                "chain_name": "INPUT",
                                "rules": [
                                    "-P INPUT ACCEPT"
                                ]
                            },
                            {
                                "chain_name": "FORWARD",
                                "rules": [
                                    "-P FORWARD ACCEPT"
                                ]
                            },
                            {
                                "chain_name": "OUTPUT",
                                "rules": [
                                    "-P OUTPUT ACCEPT"
                                ]
                            }
                        ]
                    }
                ]
            },
            "bonding": null
        },
        "virtual_network": {
            "ovs": null,
            "libvirt_net": {
                "domains": [
                    {
                        "domain_name": "efi_test",
                        "domain_uuid": "a34a6e76ee374f36939b72096ee589ad",
                        "interfaces": [
                            {
                                "XMLName": {
                                    "Space": "",
                                    "Local": "interface"
                                },
                                "Managed": "",
                                "TrustGuestRXFilters": "",
                                "MAC": {
                                    "Address": "52:54:00:c9:64:4c",
                                    "Type": "",
                                    "Check": ""
                                },
                                "Source": {
                                    "User": null,
                                    "Ethernet": null,
                                    "VHostUser": null,
                                    "Server": null,
                                    "Client": null,
                                    "MCast": null,
                                    "Network": {
                                        "Network": "default",
                                        "PortGroup": "",
                                        "Bridge": "",
                                        "PortID": ""
                                    },
                                    "Bridge": null,
                                    "Internal": null,
                                    "Direct": null,
                                    "Hostdev": null,
                                    "UDP": null,
                                    "VDPA": null
                                },
                                "Boot": null,
                                "VLan": null,
                                "VirtualPort": null,
                                "IP": null,
                                "Route": null,
                                "Script": null,
                                "DownScript": null,
                                "BackendDomain": null,
                                "Target": null,
                                "Guest": null,
                                "Model": {
                                    "Type": "e1000"
                                },
                                "Driver": null,
                                "Backend": null,
                                "FilterRef": null,
                                "Tune": null,
                                "Teaming": null,
                                "Link": null,
                                "MTU": null,
                                "Bandwidth": null,
                                "PortOptions": null,
                                "Coalesce": null,
                                "ROM": null,
                                "ACPI": null,
                                "Alias": null,
                                "Address": {
                                    "PCI": {
                                        "Domain": 0,
                                        "Bus": 0,
                                        "Slot": 3,
                                        "Function": 0,
                                        "MultiFunction": "",
                                        "ZPCI": null
                                    },
                                    "Drive": null,
                                    "VirtioSerial": null,
                                    "CCID": null,
                                    "USB": null,
                                    "SpaprVIO": null,
                                    "VirtioS390": null,
                                    "CCW": null,
                                    "VirtioMMIO": null,
                                    "ISA": null,
                                    "DIMM": null,
                                    "Unassigned": null
                                }
                            }
                        ]
                    },
                    {
                        "domain_name": "ubuntu22.04",
                        "domain_uuid": "d60e084ee5b248548e42b76afd8327d9",
                        "interfaces": [
                            {
                                "XMLName": {
                                    "Space": "",
                                    "Local": "interface"
                                },
                                "Managed": "",
                                "TrustGuestRXFilters": "",
                                "MAC": {
                                    "Address": "52:54:00:93:1d:71",
                                    "Type": "",
                                    "Check": ""
                                },
                                "Source": {
                                    "User": null,
                                    "Ethernet": null,
                                    "VHostUser": null,
                                    "Server": null,
                                    "Client": null,
                                    "MCast": null,
                                    "Network": {
                                        "Network": "default",
                                        "PortGroup": "",
                                        "Bridge": "",
                                        "PortID": ""
                                    },
                                    "Bridge": null,
                                    "Internal": null,
                                    "Direct": null,
                                    "Hostdev": null,
                                    "UDP": null,
                                    "VDPA": null
                                },
                                "Boot": null,
                                "VLan": null,
                                "VirtualPort": null,
                                "IP": null,
                                "Route": null,
                                "Script": null,
                                "DownScript": null,
                                "BackendDomain": null,
                                "Target": null,
                                "Guest": null,
                                "Model": {
                                    "Type": "virtio"
                                },
                                "Driver": null,
                                "Backend": null,
                                "FilterRef": null,
                                "Tune": null,
                                "Teaming": null,
                                "Link": null,
                                "MTU": null,
                                "Bandwidth": null,
                                "PortOptions": null,
                                "Coalesce": null,
                                "ROM": null,
                                "ACPI": null,
                                "Alias": null,
                                "Address": {
                                    "PCI": {
                                        "Domain": 0,
                                        "Bus": 1,
                                        "Slot": 0,
                                        "Function": 0,
                                        "MultiFunction": "",
                                        "ZPCI": null
                                    },
                                    "Drive": null,
                                    "VirtioSerial": null,
                                    "CCID": null,
                                    "USB": null,
                                    "SpaprVIO": null,
                                    "VirtioS390": null,
                                    "CCW": null,
                                    "VirtioMMIO": null,
                                    "ISA": null,
                                    "DIMM": null,
                                    "Unassigned": null
                                }
                            }
                        ]
                    }
                ]
            }
        }
    },
    "gpu": {
        "nvidia": [
            {
                "device_attributes": {
                    "gpu_uuid": "GPU-05548171-05c7-229a-e00e-59703ed40eb0",
                    "driver_version": "535.129.03",
                    "cuda_version": "12.2",
                    "product_name": "NVIDIA GeForce GTX 1660",
                    "product_brand": "GeForce",
                    "product_architecture": "Turing",
                    "serial_number": "N/A"
                },
                "performance": {
                    "gpu_usage": 34,
                    "fb_memory_used": 1091,
                    "fb_memory_total": 6144,
                    "fb_memory_usage": 17,
                    "bar1_memory_used": 18,
                    "bar1_memory_total": 256,
                    "bar1_memory_usage": 7
                }
            }
        ],
        "drm": [
            {
                "driver_name": "nvidia-drm",
                "driver_version": "0.0.0",
                "driver_date": "20160202",
                "driver_description": "NVIDIA DRM driver"
            }
        ]
    }
}
```
</details>

### Get software information
```
http://127.0.0.1:8082/software
```

<details>
<summary>Reply example</summary>

```
{
    "deb": [
        {
            "package": "accountsservice",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "23.13.9-2ubuntu2",
            "section": "admin",
            "installed_size": 516,
            "depends": "default-dbus-system-bus | dbus-system-bus, libaccountsservice0 (= 23.13.9-2ubuntu2), libc6 (>= 2.34), libglib2.0-0 (>= 2.75.3), libpolkit-gobject-1-0 (>= 0.99)",
            "conffiles": [],
            "pre_depends": "",
            "description": "query and manipulate user account information The AccountService project provides a set of D-Bus interfaces for querying and manipulating user account information and an implementation of these interfaces, based on the useradd, usermod and userdel commands.",
            "source": "",
            "homepage": "https://www.freedesktop.org/wiki/Software/AccountsService/"
        },
        {
            "package": "acl",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "2.3.1-3",
            "section": "utils",
            "installed_size": 192,
            "depends": "libacl1 (= 2.3.1-3), libc6 (>= 2.34)",
            "conffiles": [],
            "pre_depends": "",
            "description": "access control list - utilities This package contains the getfacl and setfacl utilities needed for manipulating access control lists. It also contains the chacl IRIX compatible utility.",
            "source": "",
            "homepage": "https://savannah.nongnu.org/projects/acl/"
        },
        {
            "package": "acpi-support",
            "status": "deinstall ok config-files",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Core developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "0.144",
            "section": "admin",
            "installed_size": 60,
            "depends": "acpid (>= 1.0.4-1ubuntu4)",
            "conffiles": [],
            "pre_depends": "",
            "description": "scripts for handling many ACPI events This package contains scripts to react to various ACPI events. It only includes scripts for events that can be supported with some level of safety cross platform. . It is able to:  * Detect loss and gain of AC power, lid closure, and the press of a    number of specific buttons (on Asus, IBM, Lenovo, Panasonic, Sony    and Toshiba laptops).  * Suspend, hibernate and resume the computer, with workarounds for    hardware that needs it.  * On some laptops, set screen brightness.",
            "source": "",
            "homepage": ""
        },
        {
            "package": "acpid",
            "status": "deinstall ok config-files",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "1:2.0.33-2ubuntu1",
            "section": "admin",
            "installed_size": 146,
            "depends": "libc6 (>= 2.34), lsb-base (>= 3.2-14), kmod",
            "conffiles": [],
            "pre_depends": "init-system-helpers (>= 1.54~) /etc/default/acpid 5b934527919a9bba89c7978d15e918b3 /etc/init.d/acpid 2ba41d3445b3052d9d2d170b7a9c30dc",
            "description": "Advanced Configuration and Power Interface event daemon Modern computers support the Advanced Configuration and Power Interface (ACPI) to allow intelligent power management on your system and to query battery and configuration status. . ACPID is a completely flexible, totally extensible daemon for delivering ACPI events. It listens on netlink interface (or on the deprecated file /proc/acpi/event), and when an event occurs, executes programs to handle the event. The programs it executes are configured through a set of configuration files, which can be dropped into place by packages or by the admin.",
            "source": "",
            "homepage": "http://sourceforge.net/projects/acpid2/"
        },
        {
            "package": "adduser",
            "status": "install ok installed",
            "priority": "important",
            "architecture": "all",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "3.137ubuntu1",
            "section": "admin",
            "installed_size": 452,
            "depends": "passwd",
            "conffiles": [
                "/etc/adduser.conf",
                "/etc/deluser.conf"
            ],
            "pre_depends": "",
            "description": "add and remove users and groups This package includes the 'adduser' and 'deluser' commands for creating and removing users. .  - 'adduser' creates new users and groups and adds existing users to    existing groups;  - 'deluser' removes users and groups and removes users from a given    group. . Adding users with 'adduser' is much easier than adding them manually. 'Adduser' will choose UID and GID values that conform to Debian policy, create a home directory, copy skeletal user configuration, and automate setting initial values for the user's password, real name and so on. . 'Deluser' can back up and remove users' home directories and mail spool or all the files they own on the system. . A custom script can be executed after each of the commands. . 'Adduser' and 'Deluser' are intended to be used by the local administrator in lieu of the tools from the 'useradd' suite, and they provide support for easy use from Debian package maintainer scripts, functioning as kind of a policy layer to make those scripts easier and more stable to write and maintain.",
            "source": "",
            "homepage": ""
        },
        {
            "package": "adwaita-icon-theme",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "all",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "41.0-1ubuntu1",
            "section": "gnome",
            "installed_size": 5234,
            "depends": "hicolor-icon-theme, gtk-update-icon-cache, ubuntu-mono | adwaita-icon-theme-full",
            "conffiles": [],
            "pre_depends": "",
            "description": "default icon theme of GNOME (small subset) This package contains the default icon theme used by the GNOME desktop. The icons are used in many of the official gnome applications like eog, evince, system monitor, and many more. . This package only contains a small subset of the original GNOME icons which are not provided by the Humanity icon theme, to avoid installing many duplicated icons. Please install adwaita-icon-theme-full if you want the full set.",
            "source": "",
            "homepage": ""
        },
        {
            "package": "aisleriot",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "1:3.22.23-1",
            "section": "games",
            "installed_size": 9156,
            "depends": "dconf-gsettings-backend | gsettings-backend, guile-3.0-libs, libatk1.0-0 (>= 1.12.4), libc6 (>= 2.34), libcairo2 (>= 1.10.0), libcanberra-gtk3-0 (>= 0.25), libcanberra0 (>= 0.2), libgdk-pixbuf-2.0-0 (>= 2.22.0), libglib2.0-0 (>= 2.37.3), libgtk-3-0 (>= 3.19.12), librsvg2-2 (>= 2.32.0)",
            "conffiles": [],
            "pre_depends": "",
            "description": "GNOME solitaire card game collection This is a collection of over eighty different solitaire card games, including popular variants such as spider, freecell, klondike, thirteen (pyramid), yukon, canfield and many more.",
            "source": "",
            "homepage": "https://wiki.gnome.org/Apps/Aisleriot"
        },
        {
            "package": "alsa-base",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "all",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Core Developers <ubuntu-devel@lists.ubuntu.com>",
            "version": "1.0.25+dfsg-0ubuntu7",
            "section": "sound",
            "installed_size": 464,
            "depends": "kmod (>= 17-1), linux-sound-base, udev",
            "conffiles": [
                "/etc/apm/scripts.d/alsa",
                "/etc/modprobe.d/alsa-base.conf",
                "/etc/modprobe.d/blacklist-modem.conf"
            ],
            "pre_depends": "",
            "description": "ALSA driver configuration files This package contains various configuration files for the ALSA drivers. . For ALSA to work on a system with a given sound card, there must be an ALSA driver for that card in the kernel. Linux 2.6 as shipped in linux-image packages contains ALSA drivers for all supported sound cards in the form of loadable modules. A custom alsa-modules package can be built from the sources in the alsa-source package using the m-a utility (included in the module-assistant package). Please read the README.Debian file for more information about loading and building modules. . ALSA is the Advanced Linux Sound Architecture.",
            "source": "alsa-driver",
            "homepage": "http://www.alsa-project.org/"
        },
        {
            "package": "alsa-topology-conf",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "all",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "1.2.5.1-2",
            "section": "libs",
            "installed_size": 420,
            "depends": "",
            "conffiles": [],
            "pre_depends": "",
            "description": "ALSA topology configuration files This package contains ALSA topology configuration files that can be used by libasound2 for specific audio hardware. . ALSA is the Advanced Linux Sound Architecture.",
            "source": "",
            "homepage": "https://www.alsa-project.org/"
        },
        {
            "package": "alsa-ucm-conf",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "all",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "1.2.9-1ubuntu3.1",
            "section": "libs",
            "installed_size": 814,
            "depends": "libasound2 (>= 1.2.7)",
            "conffiles": [],
            "pre_depends": "",
            "description": "ALSA Use Case Manager configuration files This package contains ALSA Use Case Manager configuration of audio input/output names and routing for specific audio hardware. They can be used with the alsaucm tool. . ALSA is the Advanced Linux Sound Architecture.",
            "source": "",
            "homepage": "https://www.alsa-project.org/"
        },
        {
            "package": "alsa-utils",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "1.2.9-1ubuntu1",
            "section": "sound",
            "installed_size": 2576,
            "depends": "kmod (>= 17-1~), libasound2 (>= 1.2.6.1), libatopology2 (>= 1.2.2), libc6 (>= 2.34), libfftw3-single3 (>= 3.3.10), libncursesw6 (>= 6), libsamplerate0 (>= 0.1.7), libtinfo6 (>= 6)",
            "conffiles": [
                "/etc/init.d/alsa-utils"
            ],
            "pre_depends": "",
            "description": "Utilities for configuring and using ALSA Included tools:  - alsactl: advanced controls for ALSA sound drivers  - alsaloop: create loopbacks between PCM capture and playback devices  - alsamixer: curses mixer  - alsaucm: alsa use case manager  - amixer: command line mixer  - amidi: read from and write to ALSA RawMIDI ports  - aplay, arecord: command line playback and recording  - aplaymidi, arecordmidi: command line MIDI playback and recording  - aconnect, aseqnet, aseqdump: command line MIDI sequencer control  - iecset: set or dump IEC958 status bits  - speaker-test: speaker test tone generator . ALSA is the Advanced Linux Sound Architecture.",
            "source": "",
            "homepage": "https://www.alsa-project.org/"
        },
        {
            "package": "amd64-microcode",
            "status": "install ok installed",
            "priority": "standard",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "3.20230808.1.1ubuntu1",
            "section": "non-free-firmware/admin",
            "installed_size": 275,
            "depends": "",
            "conffiles": [
                "/etc/default/amd64-microcode",
                "/etc/modprobe.d/amd64-microcode-blacklist.conf"
            ],
            "pre_depends": "",
            "description": "Processor microcode firmware for AMD CPUs This package contains microcode patches for all AMD AMD64 processors.  AMD releases microcode patches to correct processor behavior as documented in the respective processor revision guides.  This package includes both AMD CPU microcode patches and AMD SEV firmware updates. . For Intel processors, please refer to the intel-microcode package.",
            "source": "",
            "homepage": ""
        },
        {
            "package": "anacron",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "2.3-38ubuntu1",
            "section": "admin",
            "installed_size": 95,
            "depends": "lsb-base, libc6 (>= 2.34)",
            "conffiles": [
                "/etc/anacrontab",
                "/etc/cron.d/anacron",
                "/etc/cron.daily/0anacron",
                "/etc/cron.monthly/0anacron",
                "/etc/cron.weekly/0anacron",
                "/etc/default/anacron",
                "/etc/init.d/anacron"
            ],
            "pre_depends": "",
            "description": "cron-like program that doesn't go by time Anacron (like \"anac(h)ronistic\") is a periodic command scheduler.  It executes commands at intervals specified in days.  Unlike cron, it does not assume that the system is running continuously.  It can therefore be used to control the execution of daily, weekly, and monthly jobs (or anything with a period of n days), on systems that don't run 24 hours a day.  When installed and configured properly, Anacron will make sure that the commands are run at the specified intervals as closely as machine uptime permits. . This package is pre-configured to execute the daily jobs of the Debian system.  You should install this program if your system isn't powered on 24 hours a day to make sure the maintenance jobs of other Debian packages are executed each day.",
            "source": "",
            "homepage": "http://sourceforge.net/projects/anacron/"
        },
        {
            "package": "anydesk",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "AnyDesk Software GmbH <linux-support@anydesk.com>",
            "version": "6.3.0",
            "section": "net",
            "installed_size": 16818,
            "depends": "libc6 (>= 2.7), libgcc1 (>= 1:4.1.1), libglib2.0-0 (>= 2.16.0), libgtk2.0-0 (>= 2.20.1), libstdc++6 (>= 4.1.1), libx11-6, libxcb-shm0, libxcb1, libpango-1.0-0, libcairo2, libxrandr2 (>= 1.3), libx11-xcb1, libxtst6, libxfixes3, libxdamage1, libxkbfile1, libgtkglext1",
            "conffiles": [],
            "pre_depends": "",
            "description": "The fastest remote desktop software on the market.  It allows for new usage scenarios and applications that have not been possible with current remote desktop software.",
            "source": "",
            "homepage": "https://anydesk.com/"
        },
        {
            "package": "apg",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "2.2.3.dfsg.1-5build2",
            "section": "admin",
            "installed_size": 114,
            "depends": "libc6 (>= 2.34), libcrypt1 (>= 1:4.1.0) /etc/apg.conf 0ce80337a44589a92effbcde338f74e1",
            "conffiles": [
                "/etc/apg.conf"
            ],
            "pre_depends": "",
            "description": "Automated Password Generator - Standalone version APG (Automated Password Generator) is the tool set for random password generation. It generates some random words of required type and prints them to standard output. This binary package contains only the standalone version of apg. Advantages:  * Built-in ANSI X9.17 RNG (Random Number Generator)(CAST/SHA1)  * Built-in password quality checking system (now it has support for Bloom    filter for faster access)  * Two Password Generation Algorithms:     1. Pronounceable Password Generation Algorithm (according to NIST        FIPS 181)     2. Random Character Password Generation Algorithm with 35        configurable modes of operation  * Configurable password length parameters  * Configurable amount of generated passwords  * Ability to initialize RNG with user string  * Support for /dev/random  * Ability to crypt() generated passwords and print them as additional output.  * Special parameters to use APG in script  * Ability to log password generation requests for network version  * Ability to control APG service access using tcpd  * Ability to use password generation service from any type of box (Mac,    WinXX, etc.) that connected to network  * Ability to enforce remote users to use only allowed type of password    generation The client/server version of apg has been deliberately omitted. . Please note that there are security flaws in pronounceable password generation schemes (see Ganesan / Davis \"A New Attack on Random Pronounceable Password Generators\", in \"Proceedings of the 17th National Computer Security Conference (NCSC), Oct. 11-14, 1994 (Volume 1)\", http://csrc.nist.gov/publications/history/nissc/ 1994-17th-NCSC-proceedings-vol-1.pdf, pages 203-216) . Also note that the FIPS 181 standard from 1993 has been withdrawn by NIST in 2015 with no superseding publication. This means that the document is considered by its publicher as obsolete and not been updated to reference current or revised voluntary industry standards, federal specifications, or federal data standards. . apg has not seen upstream attention since 2003, upstream is not answering e-mail, and the upstream web page does not look like it is in good working order. The Debian maintainer plans to discontinue apg maintenance as soon as an actually maintained software with a compariable feature set becomes available.",
            "source": "",
            "homepage": "http://www.adel.nursat.kz/apg/"
        },
        {
            "package": "apparmor",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "4.0.0~alpha2-0ubuntu5",
            "section": "admin",
            "installed_size": 2880,
            "depends": "debconf, lsb-base, debconf (>= 0.5) | debconf-2.0, libc6 (>= 2.38)",
            "conffiles": [
                "/etc/apparmor.d/abi/3.0",
                "/etc/apparmor.d/abi/4.0",
                "/etc/apparmor.d/abi/kernel-5.4-outoftree-network",
                "/etc/apparmor.d/abi/kernel-5.4-vanilla",
                "/etc/apparmor.d/abstractions/X",
                "/etc/apparmor.d/abstractions/apache2-common",
                "/etc/apparmor.d/abstractions/apparmor_api/change_profile",
                "/etc/apparmor.d/abstractions/apparmor_api/examine",
                "/etc/apparmor.d/abstractions/apparmor_api/find_mountpoint",
                "/etc/apparmor.d/abstractions/apparmor_api/introspect",
                "/etc/apparmor.d/abstractions/apparmor_api/is_enabled",
                "/etc/apparmor.d/abstractions/aspell",
                "/etc/apparmor.d/abstractions/audio",
                "/etc/apparmor.d/abstractions/authentication",
                "/etc/apparmor.d/abstractions/base",
                "/etc/apparmor.d/abstractions/bash",
                "/etc/apparmor.d/abstractions/consoles",
                "/etc/apparmor.d/abstractions/crypto",
                "/etc/apparmor.d/abstractions/cups-client",
                "/etc/apparmor.d/abstractions/dbus",
                "/etc/apparmor.d/abstractions/dbus-accessibility",
                "/etc/apparmor.d/abstractions/dbus-accessibility-strict",
                "/etc/apparmor.d/abstractions/dbus-network-manager-strict",
                "/etc/apparmor.d/abstractions/dbus-session",
                "/etc/apparmor.d/abstractions/dbus-session-strict",
                "/etc/apparmor.d/abstractions/dbus-strict",
                "/etc/apparmor.d/abstractions/dconf",
                "/etc/apparmor.d/abstractions/dovecot-common",
                "/etc/apparmor.d/abstractions/dri-common",
                "/etc/apparmor.d/abstractions/dri-enumerate",
                "/etc/apparmor.d/abstractions/enchant",
                "/etc/apparmor.d/abstractions/exo-open",
                "/etc/apparmor.d/abstractions/fcitx",
                "/etc/apparmor.d/abstractions/fcitx-strict",
                "/etc/apparmor.d/abstractions/fonts",
                "/etc/apparmor.d/abstractions/freedesktop.org",
                "/etc/apparmor.d/abstractions/gio-open",
                "/etc/apparmor.d/abstractions/gnome",
                "/etc/apparmor.d/abstractions/gnupg",
                "/etc/apparmor.d/abstractions/groff",
                "/etc/apparmor.d/abstractions/gtk",
                "/etc/apparmor.d/abstractions/gvfs-open",
                "/etc/apparmor.d/abstractions/hosts_access",
                "/etc/apparmor.d/abstractions/ibus",
                "/etc/apparmor.d/abstractions/kde",
                "/etc/apparmor.d/abstractions/kde-globals-write",
                "/etc/apparmor.d/abstractions/kde-icon-cache-write",
                "/etc/apparmor.d/abstractions/kde-language-write",
                "/etc/apparmor.d/abstractions/kde-open5",
                "/etc/apparmor.d/abstractions/kerberosclient",
                "/etc/apparmor.d/abstractions/ldapclient",
                "/etc/apparmor.d/abstractions/libpam-systemd",
                "/etc/apparmor.d/abstractions/likewise",
                "/etc/apparmor.d/abstractions/mdns",
                "/etc/apparmor.d/abstractions/mesa",
                "/etc/apparmor.d/abstractions/mir",
                "/etc/apparmor.d/abstractions/mozc",
                "/etc/apparmor.d/abstractions/mysql",
                "/etc/apparmor.d/abstractions/nameservice",
                "/etc/apparmor.d/abstractions/nis",
                "/etc/apparmor.d/abstractions/nss-systemd",
                "/etc/apparmor.d/abstractions/nvidia",
                "/etc/apparmor.d/abstractions/opencl",
                "/etc/apparmor.d/abstractions/opencl-common",
                "/etc/apparmor.d/abstractions/opencl-intel",
                "/etc/apparmor.d/abstractions/opencl-mesa",
                "/etc/apparmor.d/abstractions/opencl-nvidia",
                "/etc/apparmor.d/abstractions/opencl-pocl",
                "/etc/apparmor.d/abstractions/openssl",
                "/etc/apparmor.d/abstractions/orbit2",
                "/etc/apparmor.d/abstractions/p11-kit",
                "/etc/apparmor.d/abstractions/perl",
                "/etc/apparmor.d/abstractions/php",
                "/etc/apparmor.d/abstractions/php-worker",
                "/etc/apparmor.d/abstractions/php5",
                "/etc/apparmor.d/abstractions/postfix-common",
                "/etc/apparmor.d/abstractions/private-files",
                "/etc/apparmor.d/abstractions/private-files-strict",
                "/etc/apparmor.d/abstractions/python",
                "/etc/apparmor.d/abstractions/qt5",
                "/etc/apparmor.d/abstractions/qt5-compose-cache-write",
                "/etc/apparmor.d/abstractions/qt5-settings-write",
                "/etc/apparmor.d/abstractions/recent-documents-write",
                "/etc/apparmor.d/abstractions/ruby",
                "/etc/apparmor.d/abstractions/samba",
                "/etc/apparmor.d/abstractions/samba-rpcd",
                "/etc/apparmor.d/abstractions/smbpass",
                "/etc/apparmor.d/abstractions/snap_browsers",
                "/etc/apparmor.d/abstractions/ssl_certs",
                "/etc/apparmor.d/abstractions/ssl_keys",
                "/etc/apparmor.d/abstractions/svn-repositories",
                "/etc/apparmor.d/abstractions/trash",
                "/etc/apparmor.d/abstractions/ubuntu-bittorrent-clients",
                "/etc/apparmor.d/abstractions/ubuntu-browsers",
                "/etc/apparmor.d/abstractions/ubuntu-browsers.d/chromium-browser",
                "/etc/apparmor.d/abstractions/ubuntu-browsers.d/java",
                "/etc/apparmor.d/abstractions/ubuntu-browsers.d/kde",
                "/etc/apparmor.d/abstractions/ubuntu-browsers.d/mailto",
                "/etc/apparmor.d/abstractions/ubuntu-browsers.d/multimedia",
                "/etc/apparmor.d/abstractions/ubuntu-browsers.d/plugins-common",
                "/etc/apparmor.d/abstractions/ubuntu-browsers.d/productivity",
                "/etc/apparmor.d/abstractions/ubuntu-browsers.d/text-editors",
                "/etc/apparmor.d/abstractions/ubuntu-browsers.d/ubuntu-integration",
                "/etc/apparmor.d/abstractions/ubuntu-browsers.d/ubuntu-integration-xul",
                "/etc/apparmor.d/abstractions/ubuntu-browsers.d/user-files",
                "/etc/apparmor.d/abstractions/ubuntu-console-browsers",
                "/etc/apparmor.d/abstractions/ubuntu-console-email",
                "/etc/apparmor.d/abstractions/ubuntu-email",
                "/etc/apparmor.d/abstractions/ubuntu-feed-readers",
                "/etc/apparmor.d/abstractions/ubuntu-gnome-terminal",
                "/etc/apparmor.d/abstractions/ubuntu-helpers",
                "/etc/apparmor.d/abstractions/ubuntu-konsole",
                "/etc/apparmor.d/abstractions/ubuntu-media-players",
                "/etc/apparmor.d/abstractions/ubuntu-unity7-base",
                "/etc/apparmor.d/abstractions/ubuntu-unity7-launcher",
                "/etc/apparmor.d/abstractions/ubuntu-unity7-messaging",
                "/etc/apparmor.d/abstractions/ubuntu-xterm",
                "/etc/apparmor.d/abstractions/user-download",
                "/etc/apparmor.d/abstractions/user-mail",
                "/etc/apparmor.d/abstractions/user-manpages",
                "/etc/apparmor.d/abstractions/user-tmp",
                "/etc/apparmor.d/abstractions/user-write",
                "/etc/apparmor.d/abstractions/video",
                "/etc/apparmor.d/abstractions/vulkan",
                "/etc/apparmor.d/abstractions/wayland",
                "/etc/apparmor.d/abstractions/web-data",
                "/etc/apparmor.d/abstractions/winbind",
                "/etc/apparmor.d/abstractions/wutmp",
                "/etc/apparmor.d/abstractions/xad",
                "/etc/apparmor.d/abstractions/xdg-desktop",
                "/etc/apparmor.d/abstractions/xdg-open",
                "/etc/apparmor.d/bin.toybox",
                "/etc/apparmor.d/local/README",
                "/etc/apparmor.d/lsb_release",
                "/etc/apparmor.d/nvidia_modprobe",
                "/etc/apparmor.d/opt.brave.com.brave.brave",
                "/etc/apparmor.d/opt.google.chrome.chrome",
                "/etc/apparmor.d/opt.microsoft.msedge.msedge",
                "/etc/apparmor.d/opt.vivaldi.vivaldi-bin",
                "/etc/apparmor.d/tunables/alias",
                "/etc/apparmor.d/tunables/apparmorfs",
                "/etc/apparmor.d/tunables/dovecot",
                "/etc/apparmor.d/tunables/etc",
                "/etc/apparmor.d/tunables/global",
                "/etc/apparmor.d/tunables/home",
                "/etc/apparmor.d/tunables/home.d/site.local",
                "/etc/apparmor.d/tunables/kernelvars",
                "/etc/apparmor.d/tunables/multiarch",
                "/etc/apparmor.d/tunables/multiarch.d/site.local",
                "/etc/apparmor.d/tunables/proc",
                "/etc/apparmor.d/tunables/run",
                "/etc/apparmor.d/tunables/securityfs",
                "/etc/apparmor.d/tunables/share",
                "/etc/apparmor.d/tunables/sys",
                "/etc/apparmor.d/tunables/xdg-user-dirs",
                "/etc/apparmor.d/usr.bin.buildah",
                "/etc/apparmor.d/usr.bin.busybox",
                "/etc/apparmor.d/usr.bin.cam",
                "/etc/apparmor.d/usr.bin.ch-checkns",
                "/etc/apparmor.d/usr.bin.ch-run",
                "/etc/apparmor.d/usr.bin.crun",
                "/etc/apparmor.d/usr.bin.flatpak",
                "/etc/apparmor.d/usr.bin.ipa_verify",
                "/etc/apparmor.d/usr.bin.lc-compliance",
                "/etc/apparmor.d/usr.bin.libcamerify",
                "/etc/apparmor.d/usr.bin.lxc-attach",
                "/etc/apparmor.d/usr.bin.lxc-create",
                "/etc/apparmor.d/usr.bin.lxc-destroy",
                "/etc/apparmor.d/usr.bin.lxc-execute",
                "/etc/apparmor.d/usr.bin.lxc-stop",
                "/etc/apparmor.d/usr.bin.lxc-unshare",
                "/etc/apparmor.d/usr.bin.lxc-usernsexec",
                "/etc/apparmor.d/usr.bin.mmdebstrap",
                "/etc/apparmor.d/usr.bin.podman",
                "/etc/apparmor.d/usr.bin.qcam",
                "/etc/apparmor.d/usr.bin.rootlesskit",
                "/etc/apparmor.d/usr.bin.rpm",
                "/etc/apparmor.d/usr.bin.sbuild",
                "/etc/apparmor.d/usr.bin.sbuild-abort",
                "/etc/apparmor.d/usr.bin.sbuild-apt",
                "/etc/apparmor.d/usr.bin.sbuild-checkpackages",
                "/etc/apparmor.d/usr.bin.sbuild-clean",
                "/etc/apparmor.d/usr.bin.sbuild-createchroot",
                "/etc/apparmor.d/usr.bin.sbuild-distupgrade",
                "/etc/apparmor.d/usr.bin.sbuild-hold",
                "/etc/apparmor.d/usr.bin.sbuild-shell",
                "/etc/apparmor.d/usr.bin.sbuild-unhold",
                "/etc/apparmor.d/usr.bin.sbuild-update",
                "/etc/apparmor.d/usr.bin.sbuild-upgrade",
                "/etc/apparmor.d/usr.bin.slirp4netns",
                "/etc/apparmor.d/usr.bin.stress-ng",
                "/etc/apparmor.d/usr.bin.thunderbird",
                "/etc/apparmor.d/usr.bin.trinity",
                "/etc/apparmor.d/usr.bin.tup",
                "/etc/apparmor.d/usr.bin.userbindmount",
                "/etc/apparmor.d/usr.bin.uwsgi-core",
                "/etc/apparmor.d/usr.bin.vdens",
                "/etc/apparmor.d/usr.bin.vpnns",
                "/etc/apparmor.d/usr.lib.multiarch.opera.opera",
                "/etc/apparmor.d/usr.lib.multiarch.qt5.libexec.QtWebEngineProcess",
                "/etc/apparmor.d/usr.lib.qt6.libexec.QtWebEngineProcess",
                "/etc/apparmor.d/usr.libexec.multiarch.bazel.linux-sandbox",
                "/etc/apparmor.d/usr.libexec.virtiofsd",
                "/etc/apparmor.d/usr.sbin.runc",
                "/etc/apparmor.d/usr.sbin.sbuild-adduser",
                "/etc/apparmor.d/usr.sbin.sbuild-destroychroot",
                "/etc/apparmor.d/usr.share.code.bin.code",
                "/etc/apparmor/parser.conf",
                "/etc/init.d/apparmor"
            ],
            "pre_depends": "",
            "description": "user-space parser utility for AppArmor apparmor provides the system initialization scripts needed to use the AppArmor Mandatory Access Control system, including the AppArmor Parser which is required to convert AppArmor text profiles into machine-readable policies that are loaded into the kernel for use with the AppArmor Linux Security Module.",
            "source": "",
            "homepage": "https://apparmor.net/"
        },
        {
            "package": "appstream",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "0.16.2-1",
            "section": "admin",
            "installed_size": 308,
            "depends": "shared-mime-info, libappstream4 (>= 0.16.0), libc6 (>= 2.34), libglib2.0-0 (>= 2.75.3)",
            "conffiles": [
                "/etc/appstream.conf",
                "/etc/apt/apt.conf.d/50appstream"
            ],
            "pre_depends": "",
            "description": "Software component metadata management AppStream is a metadata specification which permits software components to provide information about themselves to automated systems and end-users before the software is actually installed. The AppStream project provides facilities to easily access and transform this metadata, as well as a few additional services to allow building feature-rich software centers and similar applications. . This package provides tools to generate, maintain and query the AppStream data pool of installed and available software, and enables integration with the APT package manager. . The 'appstreamcli' tool can be used for accessing the software component pool as well as for working with AppStream metadata directly, including validating it for compliance with the specification.",
            "source": "",
            "homepage": "https://www.freedesktop.org/wiki/Distributions/AppStream/"
        },
        {
            "package": "apt",
            "status": "install ok installed",
            "priority": "important",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "2.7.3",
            "section": "admin",
            "installed_size": 4129,
            "depends": "base-passwd (>= 3.6.1) | adduser, gpgv | gpgv2 | gpgv1, libapt-pkg6.0 (>= 2.7.3), ubuntu-keyring, libc6 (>= 2.34), libgcc-s1 (>= 3.3.1), libgnutls30 (>= 3.7.5), libseccomp2 (>= 2.4.2), libstdc++6 (>= 13.1), libsystemd0",
            "conffiles": [
                "/etc/apt/apt.conf.d/01-vendor-ubuntu",
                "/etc/apt/apt.conf.d/01autoremove",
                "/etc/cron.daily/apt-compat",
                "/etc/logrotate.d/apt"
            ],
            "pre_depends": "",
            "description": "commandline package manager This package provides commandline tools for searching and managing as well as querying information about packages as a low-level access to all features of the libapt-pkg library. . These include:  * apt-get for retrieval of packages and information about them    from authenticated sources and for installation, upgrade and    removal of packages together with their dependencies  * apt-cache for querying available information about installed    as well as installable packages  * apt-cdrom to use removable media as a source for packages  * apt-config as an interface to the configuration settings  * apt-key as an interface to manage authentication keys",
            "source": "",
            "homepage": ""
        },
        {
            "package": "apt-config-icons",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "all",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "0.16.2-1",
            "section": "misc",
            "installed_size": 23,
            "depends": "appstream /etc/apt/apt.conf.d/60icons abbeb258b1bce3b278457c3989f5a0d5",
            "conffiles": [
                "/etc/apt/apt.conf.d/60icons"
            ],
            "pre_depends": "",
            "description": "APT configuration snippet to enable icon downloads This package contains an APT configuration snippet that enables the download of an icon tarball containing application icons for display in software centers. Icons get downloaded directly from the archive via regular cache updates, and match the software currently available in the respective archive suites. . This snippet enables icons of the default icon size set in the AppStream specification for software centers, 64x64px. It also enables downloads of smaller 48x48px icons.",
            "source": "appstream",
            "homepage": "https://www.freedesktop.org/wiki/Distributions/AppStream/"
        },
        {
            "package": "apt-config-icons-hidpi",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "all",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "0.16.2-1",
            "section": "misc",
            "installed_size": 23,
            "depends": "apt-config-icons /etc/apt/apt.conf.d/60icons-hidpi fa6ae619b4e13d628ac94984e07f30aa",
            "conffiles": [
                "/etc/apt/apt.conf.d/60icons-hidpi"
            ],
            "pre_depends": "",
            "description": "APT configuration snippet to enable HiDPI icon downloads This package contains an APT configuration snippet that enables the download of an icon tarball containing application icons for display in software centers. Icons get downloaded directly from the archive via regular cache updates, and match the software currently available in the respective archive suites. . This snippet enables icons in the size of 64x64@2, that are suitable for HiDPI displays.",
            "source": "appstream",
            "homepage": "https://www.freedesktop.org/wiki/Distributions/AppStream/"
        },
        {
            "package": "apt-transport-https",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "all",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "2.7.3",
            "section": "oldlibs",
            "installed_size": 36,
            "depends": "apt (>= 1.5~alpha4)",
            "conffiles": [],
            "pre_depends": "",
            "description": "transitional package for https support This is a dummy transitional package - https support has been moved into the apt package in 1.5. It can be safely removed.",
            "source": "apt",
            "homepage": ""
        },
        {
            "package": "apt-utils",
            "status": "install ok installed",
            "priority": "important",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "2.7.3",
            "section": "admin",
            "installed_size": 672,
            "depends": "apt (= 2.7.3), libapt-pkg6.0 (>= 2.7.3), libc6 (>= 2.34), libdb5.3, libgcc-s1 (>= 3.3.1), libstdc++6 (>= 13.1)",
            "conffiles": [],
            "pre_depends": "",
            "description": "package management related utility programs This package contains some less used commandline utilities related to package management with APT. .  * apt-extracttemplates is used by debconf to prompt for configuration    questions before installation.  * apt-ftparchive is used to create Packages and other index files    needed to publish an archive of Debian packages  * apt-sortpkgs is a Packages/Sources file normalizer.",
            "source": "apt",
            "homepage": ""
        },
        {
            "package": "aptdaemon",
            "status": "install ok installed",
            "priority": "extra",
            "architecture": "all",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "1.1.1+bzr982-0ubuntu44",
            "section": "admin",
            "installed_size": 156,
            "depends": "python3:any (>= 3.2~), gir1.2-glib-2.0, python3-aptdaemon (= 1.1.1+bzr982-0ubuntu44), python3-gi, policykit-1",
            "conffiles": [
                "/etc/apt/apt.conf.d/20dbus",
                "/etc/dbus-1/system.d/org.debian.apt.conf"
            ],
            "pre_depends": "",
            "description": "transaction based package management service Aptdaemon allows normal users to perform package management tasks, e.g. refreshing the cache, upgrading the system, installing or removing software packages. . Currently it comes with the following main features: .  - Programming language independent D-Bus interface, which allows one to    write clients in several languages  - Runs only if required (D-Bus activation)  - Fine grained privilege management using PolicyKit, e.g. allowing all    desktop user to query for updates without entering a password  - Support for media changes during installation from DVD/CDROM  - Support for debconf (Debian's package configuration system)  - Support for attaching a terminal to the underlying dpkg call . This package contains the aptd script and all the data files required to run the daemon. Moreover it contains the aptdcon script, which is a command line client for aptdaemon. The API is not stable yet.",
            "source": "",
            "homepage": "https://launchpad.net/aptdaemon"
        },
        {
            "package": "aptdaemon-data",
            "status": "install ok installed",
            "priority": "extra",
            "architecture": "all",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "1.1.1+bzr982-0ubuntu44",
            "section": "admin",
            "installed_size": 228,
            "depends": "",
            "conffiles": [],
            "pre_depends": "",
            "description": "data files for clients Aptdaemon is a transaction based package management daemon. It allows normal users to perform package management tasks, e.g. refreshing the cache, upgrading the system, installing or removing software packages. . This package provides common data files (e.g. icons) for aptdaemon clients.",
            "source": "aptdaemon",
            "homepage": "https://launchpad.net/aptdaemon"
        },
        {
            "package": "apturl-common",
            "status": "deinstall ok config-files",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Michael Vogt <mvo@ubuntu.com>",
            "version": "0.5.2ubuntu22",
            "section": "admin",
            "installed_size": 168,
            "depends": "python3:any (>= 3.2~), python3-apt, python3-update-manager /etc/firefox/pref/apturl.js 127752b25e18c94a368c4327858926a7",
            "conffiles": [],
            "pre_depends": "",
            "description": "install packages using the apt protocol - common data AptUrl is a simple graphical application that takes an URL (which follows the apt-protocol) as a command line option, parses it and carries out the operations that the URL describes (that is, it asks the user if he wants the indicated packages to be installed and if the answer is positive does so for him). . This package contains the common data shared between the frontends.",
            "source": "apturl",
            "homepage": ""
        },
        {
            "package": "arj",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "3.10.22-26",
            "section": "utils",
            "installed_size": 518,
            "depends": "libc6 (>= 2.34) /etc/rearj.cfg 09221c7fffdc54f60b889da19a8e316b",
            "conffiles": [
                "/etc/rearj.cfg"
            ],
            "pre_depends": "",
            "description": "archiver for .arj files This package is an open source version of the arj archiver. This version has been created with the intent to preserve maximum compatibility and retain the feature set of original ARJ archiver as provided by ARJ Software, Inc.",
            "source": "",
            "homepage": "https://sf.net/projects/arj/"
        },
        {
            "package": "arping",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "2.23-1",
            "section": "net",
            "installed_size": 81,
            "depends": "libc6 (>= 2.34), libnet1 (>= 1.1.3), libpcap0.8 (>= 1.5.1), libseccomp2 (>= 1.0.1)",
            "conffiles": [],
            "pre_depends": "",
            "description": "sends IP and/or ARP pings (to the MAC address) The arping utility sends ARP and/or ICMP requests to the specified host and displays the replies. The host may be specified by its hostname, its IP address, or its MAC address.",
            "source": "",
            "homepage": "http://www.habets.pp.se/synscan/programs.php?prog=arping"
        },
        {
            "package": "aspell",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "0.60.8-5",
            "section": "text",
            "installed_size": 336,
            "depends": "libaspell15 (= 0.60.8-5), libc6 (>= 2.34), libncursesw6 (>= 6), libstdc++6 (>= 5), libtinfo6 (>= 6), dictionaries-common",
            "conffiles": [],
            "pre_depends": "",
            "description": "GNU Aspell spell-checker GNU Aspell is a spell-checker which can be used either as a standalone application or embedded in other programs.  Its main feature is that it does a much better job of suggesting possible spellings than just about any other spell-checker available for the English language, including Ispell and Microsoft Word.  It also has many other technical enhancements over Ispell such as using shared memory for dictionaries and intelligently handling personal dictionaries when more than one Aspell process is open at once. . Aspell is designed to be a drop-in replacement for Ispell.",
            "source": "",
            "homepage": "http://aspell.net/"
        },
        {
            "package": "aspell-en",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "all",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "2020.12.07-0-1",
            "section": "text",
            "installed_size": 429,
            "depends": "aspell, dictionaries-common",
            "conffiles": [],
            "pre_depends": "",
            "description": "English dictionary for GNU Aspell This package contains all the required files to add support for English language to the GNU Aspell spell checker. . American, British, Canadian and Australian spellings are included.",
            "source": "",
            "homepage": "http://aspell.net/"
        },
        {
            "package": "at-spi2-common",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "all",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "2.50.0-1",
            "section": "misc",
            "installed_size": 52,
            "depends": "",
            "conffiles": [],
            "pre_depends": "",
            "description": "Assistive Technology Service Provider Interface (common files) This package contains the common resource files of GNOME Accessibility.",
            "source": "at-spi2-core",
            "homepage": "https://wiki.gnome.org/Accessibility"
        },
        {
            "package": "at-spi2-core",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "2.50.0-1",
            "section": "misc",
            "installed_size": 177,
            "depends": "libatspi2.0-0 (>= 2.9.90), libc6 (>= 2.34), libdbus-1-3 (>= 1.9.14), libglib2.0-0 (>= 2.75.3), libsystemd0, libx11-6, libxtst6, at-spi2-common, gsettings-desktop-schemas /etc/X11/Xsession.d/90qt-a11y afc7b6dfce4d98efa295023045b20424 /etc/environment.d/90qt-a11y.conf 4f76c97d1817370071bed644d921f142 /etc/xdg/Xwayland-session.d/00-at-spi da53e8f602edbb788b3cd6dbb056e45d /etc/xdg/autostart/at-spi-dbus-bus.desktop b97f071f92cfc4af379984b27cbb7304",
            "conffiles": [
                "/etc/X11/Xsession.d/90qt-a11y",
                "/etc/environment.d/90qt-a11y.conf",
                "/etc/xdg/Xwayland-session.d/00-at-spi",
                "/etc/xdg/autostart/at-spi-dbus-bus.desktop"
            ],
            "pre_depends": "",
            "description": "Assistive Technology Service Provider Interface (D-Bus core) This package contains the core components of GNOME Accessibility.",
            "source": "",
            "homepage": "https://wiki.gnome.org/Accessibility"
        },
        {
            "package": "attr",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "1:2.5.1-4",
            "section": "utils",
            "installed_size": 136,
            "depends": "libattr1 (= 1:2.5.1-4), libc6 (>= 2.34)",
            "conffiles": [],
            "pre_depends": "",
            "description": "utilities for manipulating filesystem extended attributes A set of tools for manipulating extended attributes on filesystem objects, in particular getfattr(1) and setfattr(1). An attr(1) command is also provided which is largely compatible with the SGI IRIX tool of the same name.",
            "source": "",
            "homepage": "https://savannah.nongnu.org/projects/attr/"
        },
        {
            "package": "autoconf",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "all",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "2.71-3",
            "section": "devel",
            "installed_size": 2024,
            "depends": "perl (>> 5.005), m4 (>= 1.4.13), debianutils (>= 1.8)",
            "conffiles": [
                "/etc/emacs/site-start.d/50autoconf.el"
            ],
            "pre_depends": "",
            "description": "automatic configure script builder The standard for FSF source packages.  This is only useful if you write your own programs or if you extensively modify other people's programs. . For an extensive library of additional Autoconf macros, install the `autoconf-archive' package. . This version of autoconf is not compatible with scripts meant for Autoconf 2.13 or earlier.",
            "source": "",
            "homepage": "https://www.gnu.org/software/autoconf/"
        },
        {
            "package": "automake",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "all",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "1:1.16.5-1.3",
            "section": "devel",
            "installed_size": 1581,
            "depends": "autoconf, autotools-dev",
            "conffiles": [],
            "pre_depends": "",
            "description": "Tool for generating GNU Standards-compliant Makefiles Automake is a tool for automatically generating `Makefile.in's from files called `Makefile.am'. . The goal of Automake is to remove the burden of Makefile maintenance from the back of the individual GNU maintainer (and put it on the back of the Automake maintainer). . The `Makefile.am' is basically a series of `make' macro definitions (with rules being thrown in occasionally).  The generated `Makefile.in's are compliant with the GNU Makefile standards. . Automake 1.16 fails to work in a number of situations that Automake 1.11, and 1.15 did, so some previous versions are available as separate packages.",
            "source": "automake-1.16",
            "homepage": "https://www.gnu.org/software/automake/"
        },
        {
            "package": "autopoint",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "all",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "0.21-13",
            "section": "devel",
            "installed_size": 444,
            "depends": "xz-utils",
            "conffiles": [],
            "pre_depends": "",
            "description": "tool for setting up gettext infrastructure in a source package The `autopoint' program copies standard gettext infrastructure files into a source package.  It extracts from a macro call of the form `AM_GNU_GETTEXT_VERSION(VERSION)', found in the package's `configure.in' or `configure.ac' file, the gettext version used by the package, and copies the infrastructure files belonging to this version into the package.",
            "source": "gettext",
            "homepage": "https://www.gnu.org/software/gettext/"
        },
        {
            "package": "autotools-dev",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "all",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "20220109.1",
            "section": "devel",
            "installed_size": 134,
            "depends": "",
            "conffiles": [],
            "pre_depends": "",
            "description": "Update infrastructure for config.{guess,sub} files This package installs an up-to-date version of config.guess and config.sub, used by the automake and libtool packages.  It provides the canonical copy of those files for other packages as well. . It also documents in /usr/share/doc/autotools-dev/README.Debian.gz best practices and guidelines for using autoconf, automake and friends on Debian packages.  This is a must-read for any developers packaging software that uses the GNU autotools, or GNU gettext. . Additionally this package provides seamless integration into Debhelper or CDBS, allowing maintainers to easily update config.{guess,sub} files in their packages.",
            "source": "",
            "homepage": "https://savannah.gnu.org/projects/config/"
        },
        {
            "package": "avahi-autoipd",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "0.8-10ubuntu1.1",
            "section": "net",
            "installed_size": 105,
            "depends": "libc6 (>= 2.38), libdaemon0 (>= 0.14), adduser",
            "conffiles": [
                "/etc/avahi/avahi-autoipd.action",
                "/etc/dhcp/dhclient-enter-hooks.d/avahi-autoipd",
                "/etc/dhcp/dhclient-exit-hooks.d/zzz_avahi-autoipd",
                "/etc/network/if-down.d/avahi-autoipd",
                "/etc/network/if-up.d/avahi-autoipd"
            ],
            "pre_depends": "",
            "description": "Avahi IPv4LL network address configuration daemon Avahi is a fully LGPL framework for Multicast DNS Service Discovery. It allows programs to publish and discover services and hosts running on a local network with no specific configuration. For example you can plug into a network and instantly find printers to print to, files to look at and people to talk to. . This tool implements IPv4LL, \"Dynamic Configuration of IPv4 Link-Local Addresses\" (IETF RFC3927), a protocol for automatic IP address configuration from the link-local 169.254.0.0/16 range without the need for a central server. It is primarily intended to be used in ad-hoc networks which lack a DHCP server.",
            "source": "avahi",
            "homepage": "https://avahi.org/"
        },
        {
            "package": "avahi-daemon",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "0.8-10ubuntu1.1",
            "section": "net",
            "installed_size": 272,
            "depends": "libavahi-common3 (= 0.8-10ubuntu1.1), libavahi-core7 (= 0.8-10ubuntu1.1), libc6 (>= 2.38), libcap2 (>= 1:2.10), libdaemon0 (>= 0.14), libdbus-1-3 (>= 1.9.14), libexpat1 (>= 2.0.1), adduser, default-dbus-system-bus | dbus-system-bus",
            "conffiles": [
                "/etc/avahi/avahi-daemon.conf",
                "/etc/avahi/hosts",
                "/etc/default/avahi-daemon"
            ],
            "pre_depends": "",
            "description": "Avahi mDNS/DNS-SD daemon Avahi is a fully LGPL framework for Multicast DNS Service Discovery. It allows programs to publish and discover services and hosts running on a local network with no specific configuration. For example you can plug into a network and instantly find printers to print to, files to look at and people to talk to. . This package contains the Avahi Daemon which represents your machine on the network and allows other applications to publish and resolve mDNS/DNS-SD records.",
            "source": "avahi",
            "homepage": "https://avahi.org/"
        },
        {
            "package": "avahi-utils",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "0.8-10ubuntu1.1",
            "section": "net",
            "installed_size": 150,
            "depends": "libavahi-client3 (= 0.8-10ubuntu1.1), libavahi-common3 (= 0.8-10ubuntu1.1), libc6 (>= 2.38), libgdbm6 (>= 1.16), avahi-daemon (= 0.8-10ubuntu1.1)",
            "conffiles": [],
            "pre_depends": "",
            "description": "Avahi browsing, publishing and discovery utilities Avahi is a fully LGPL framework for Multicast DNS Service Discovery. It allows programs to publish and discover services and hosts running on a local network with no specific configuration.  For example you can plug into a network and instantly find printers to print to, files to look at and people to talk to. . This package contains several utilities that allow you to interact with the Avahi daemon, including publish, browsing and discovering services.",
            "source": "avahi",
            "homepage": "https://avahi.org/"
        },
        {
            "package": "baobab",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "45.0-1",
            "section": "gnome",
            "installed_size": 816,
            "depends": "dconf-gsettings-backend | gsettings-backend, libadwaita-1-0 (>= 1.4~beta), libc6 (>= 2.34), libcairo2 (>= 1.2.4), libglib2.0-0 (>= 2.56.0), libgtk-4-1 (>= 4.4.0), libpango-1.0-0 (>= 1.37.2)",
            "conffiles": [],
            "pre_depends": "",
            "description": "GNOME disk usage analyzer Disk Usage Analyzer is a graphical, menu-driven application to analyse disk usage in a GNOME environment. It can easily scan either the whole filesystem tree, or a specific user-requested directory branch (local or remote). . It also auto-detects in real-time any changes made to your home directory as far as any mounted/unmounted device. Disk Usage Analyzer also provides a full graphical treemap window for each selected folder.",
            "source": "",
            "homepage": "https://wiki.gnome.org/Apps/Baobab"
        },
        {
            "package": "base-files",
            "status": "install ok installed",
            "priority": "required",
            "architecture": "amd64",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "13ubuntu2",
            "section": "admin",
            "installed_size": 419,
            "depends": "libc6 (>= 2.34), libcrypt1 (>= 1:4.4.10-10ubuntu3)",
            "conffiles": [
                "/etc/debian_version",
                "/etc/dpkg/origins/debian",
                "/etc/dpkg/origins/ubuntu",
                "/etc/host.conf",
                "/etc/issue",
                "/etc/issue.net",
                "/etc/legal",
                "/etc/lsb-release",
                "/etc/profile.d/01-locale-fix.sh",
                "/etc/update-motd.d/00-header",
                "/etc/update-motd.d/10-help-text",
                "/etc/update-motd.d/50-motd-news"
            ],
            "pre_depends": "awk",
            "description": "Debian base system miscellaneous files This package contains the basic filesystem hierarchy of a Debian system, and several important miscellaneous files, such as /etc/debian_version, /etc/host.conf, /etc/issue, /etc/motd, /etc/profile, and others, and the text of several common licenses in use on Debian systems.",
            "source": "",
            "homepage": ""
        },
        {
            "package": "base-passwd",
            "status": "install ok installed",
            "priority": "required",
            "architecture": "amd64",
            "multi_arch": "foreign",
            "maintainer": "Colin Watson <cjwatson@debian.org>",
            "version": "3.6.1",
            "section": "admin",
            "installed_size": 233,
            "depends": "libc6 (>= 2.34), libdebconfclient0 (>= 0.145), libselinux1 (>= 3.1~)",
            "conffiles": [],
            "pre_depends": "",
            "description": "Debian base system master password and group files These are the canonical master copies of the user database files (/etc/passwd and /etc/group), containing the Debian-allocated user and group IDs. The update-passwd tool is provided to keep the system databases synchronized with these master files.",
            "source": "",
            "homepage": ""
        },
        {
            "package": "bash",
            "status": "install ok installed",
            "priority": "required",
            "architecture": "amd64",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "5.2.15-2ubuntu1",
            "section": "shells",
            "installed_size": 1896,
            "depends": "base-files (>= 2.1.12), debianutils (>= 5.6-0.1)",
            "conffiles": [
                "/etc/bash.bashrc",
                "/etc/skel/.bash_logout",
                "/etc/skel/.bashrc",
                "/etc/skel/.profile"
            ],
            "pre_depends": "libc6 (>= 2.36), libtinfo6 (>= 6)",
            "description": "GNU Bourne Again SHell Bash is an sh-compatible command language interpreter that executes commands read from the standard input or from a file.  Bash also incorporates useful features from the Korn and C shells (ksh and csh). . Bash is ultimately intended to be a conformant implementation of the IEEE POSIX Shell and Tools specification (IEEE Working Group 1003.2). . The Programmable Completion Code, by Ian Macdonald, is now found in the bash-completion package.",
            "source": "",
            "homepage": "http://tiswww.case.edu/php/chet/bash/bashtop.html"
        },
        {
            "package": "bash-completion",
            "status": "install ok installed",
            "priority": "standard",
            "architecture": "all",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "1:2.11-7",
            "section": "shells",
            "installed_size": 1454,
            "depends": "",
            "conffiles": [
                "/etc/bash_completion",
                "/etc/profile.d/bash_completion.sh"
            ],
            "pre_depends": "",
            "description": "programmable completion for the bash shell bash completion extends bash's standard completion behavior to achieve complex command lines with just a few keystrokes.  This project was conceived to produce programmable completion routines for the most common Linux/UNIX commands, reducing the amount of typing sysadmins and programmers need to do on a daily basis.",
            "source": "",
            "homepage": "https://github.com/scop/bash-completion"
        },
        {
            "package": "bc",
            "status": "install ok installed",
            "priority": "standard",
            "architecture": "amd64",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "1.07.1-3build1",
            "section": "math",
            "installed_size": 215,
            "depends": "libc6 (>= 2.34), libreadline8 (>= 6.0)",
            "conffiles": [],
            "pre_depends": "",
            "description": "GNU bc arbitrary precision calculator language GNU bc is an interactive algebraic language with arbitrary precision which follows the POSIX 1003.2 draft standard, with several extensions including multi-character variable names, an `else' statement and full Boolean expressions.  GNU bc does not require the separate GNU dc program.",
            "source": "",
            "homepage": "https://www.gnu.org/software/bc/"
        },
        {
            "package": "bind9-dnsutils",
            "status": "install ok installed",
            "priority": "standard",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "1:9.18.18-0ubuntu2",
            "section": "net",
            "installed_size": 489,
            "depends": "bind9-host | host, bind9-libs (= 1:9.18.18-0ubuntu2), libc6 (>= 2.38), libedit2 (>= 2.11-20080614-0), libidn2-0 (>= 2.0.0), libkrb5-3 (>= 1.6.dfsg.2)",
            "conffiles": [],
            "pre_depends": "",
            "description": "Clients provided with BIND 9 The Berkeley Internet Name Domain (BIND 9) implements an Internet domain name server.  BIND 9 is the most widely-used name server software on the Internet, and is supported by the Internet Software Consortium, www.isc.org. . This package delivers various client programs related to DNS that are derived from the BIND 9 source tree. .  - dig - query the DNS in various ways  - nslookup - the older way to do it  - nsupdate - perform dynamic updates (See RFC2136)",
            "source": "bind9",
            "homepage": "https://www.isc.org/downloads/bind/"
        },
        {
            "package": "bind9-host",
            "status": "install ok installed",
            "priority": "standard",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "1:9.18.18-0ubuntu2",
            "section": "net",
            "installed_size": 153,
            "depends": "bind9-libs (= 1:9.18.18-0ubuntu2), libc6 (>= 2.38), libidn2-0 (>= 2.0.0)",
            "conffiles": [],
            "pre_depends": "",
            "description": "DNS Lookup Utility This package provides the 'host' DNS lookup utility in the form that is bundled with the BIND 9 sources.",
            "source": "bind9",
            "homepage": "https://www.isc.org/downloads/bind/"
        },
        {
            "package": "bind9-libs",
            "status": "install ok installed",
            "priority": "standard",
            "architecture": "amd64",
            "multi_arch": "same",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "1:9.18.18-0ubuntu2",
            "section": "libs",
            "installed_size": 3373,
            "depends": "libuv1 (>= 1.40.0), libc6 (>= 2.38), libgssapi-krb5-2 (>= 1.17), libjson-c5 (>= 0.15), libkrb5-3 (>= 1.6.dfsg.2), liblmdb0 (>= 0.9.7), libmaxminddb0 (>= 1.3.0), libnghttp2-14 (>= 1.12.0), libssl3 (>= 3.0.0), libxml2 (>= 2.7.4), zlib1g (>= 1:1.1.4)",
            "conffiles": [],
            "pre_depends": "",
            "description": "Shared Libraries used by BIND 9 The Berkeley Internet Name Domain (BIND 9) implements an Internet domain name server.  BIND 9 is the most widely-used name server software on the Internet, and is supported by the Internet Software Consortium, www.isc.org. . This package contains a bundle of shared libraries used by BIND 9.",
            "source": "bind9",
            "homepage": "https://www.isc.org/downloads/bind/"
        },
        {
            "package": "binutils",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "2.41-5ubuntu1",
            "section": "devel",
            "installed_size": 158,
            "depends": "binutils-common (= 2.41-5ubuntu1), libbinutils (= 2.41-5ubuntu1), binutils-x86-64-linux-gnu (= 2.41-5ubuntu1)",
            "conffiles": [],
            "pre_depends": "",
            "description": "GNU assembler, linker and binary utilities The programs in this package are used to assemble, link and manipulate binary and object files.  They may be used in conjunction with a compiler and various libraries to build programs.",
            "source": "",
            "homepage": "https://www.gnu.org/software/binutils/"
        },
        {
            "package": "binutils-common",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "same",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "2.41-5ubuntu1",
            "section": "devel",
            "installed_size": 576,
            "depends": "",
            "conffiles": [],
            "pre_depends": "",
            "description": "Common files for the GNU assembler, linker and binary utilities This package contains the localization files used by binutils packages for various target architectures and parts of the binutils documentation. It is not useful on its own.",
            "source": "binutils",
            "homepage": "https://www.gnu.org/software/binutils/"
        },
        {
            "package": "binutils-x86-64-linux-gnu",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "allowed",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "2.41-5ubuntu1",
            "section": "devel",
            "installed_size": 11562,
            "depends": "binutils-common (= 2.41-5ubuntu1), libbinutils (= 2.41-5ubuntu1), libc6 (>= 2.38), libctf-nobfd0 (>= 2.36), libctf0 (>= 2.36), libgcc-s1 (>= 4.2), libgprofng0 (>= 2.41), libjansson4 (>= 2.14), libsframe1 (>= 2.41), libstdc++6 (>= 13.1), libzstd1 (>= 1.5.5), zlib1g (>= 1:1.1.4)",
            "conffiles": [],
            "pre_depends": "",
            "description": "GNU binary utilities, for x86-64-linux-gnu target This package provides GNU assembler, linker and binary utilities for the x86-64-linux-gnu target. . You don't need this package unless you plan to cross-compile programs for x86-64-linux-gnu and x86-64-linux-gnu is not your native platform.",
            "source": "binutils",
            "homepage": "https://www.gnu.org/software/binutils/"
        },
        {
            "package": "binwalk",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "all",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "2.3.4+dfsg1-1",
            "section": "devel",
            "installed_size": 15,
            "depends": "python3-binwalk, python3:any",
            "conffiles": [],
            "pre_depends": "",
            "description": "tool library for analyzing binary blobs and executable code Binwalk is a tool for searching a given binary image for embedded files and executable code. Specifically, it is designed for identifying files and code embedded inside of firmware images. Binwalk uses the libmagic library, so it is compatible with magic signatures created for the Unix file utility. . Binwalk also includes a custom magic signature file which contains improved signatures for files that are commonly found in firmware images such as compressed/archived files, firmware headers, Linux kernels, bootloaders, filesystems, etc. . This package is an empty package, because the binary tool is already provided with the library, dependency of this package.",
            "source": "",
            "homepage": "https://github.com/ReFirmLabs/binwalk"
        },
        {
            "package": "blt",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "2.5.3+dfsg-4.1build2",
            "section": "libs",
            "installed_size": 29,
            "depends": "tk8.6-blt2.5 (= 2.5.3+dfsg-4.1build2)",
            "conffiles": [],
            "pre_depends": "",
            "description": "graphics extension library for Tcl/Tk - run-time BLT is a library of useful extensions for the Tcl language and the popular Tk graphical toolkit.  It adds a vector and tree data type, background execution and some debugging tools to Tcl, and provides several new widgets for Tk, including graphs, bar-charts, trees, tabs, splines and hyper-links, as well as a new geometry manager, drag & drop support, and more. . This package is a dummy package which depends on the current BLT library.",
            "source": "",
            "homepage": "http://blt.sourceforge.net/"
        },
        {
            "package": "bluez",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Bluetooth team <ubuntu-bluetooth@lists.ubuntu.com>",
            "version": "5.68-0ubuntu1.1",
            "section": "admin",
            "installed_size": 4080,
            "depends": "libc6 (>= 2.38), libdbus-1-3 (>= 1.9.14), libglib2.0-0 (>= 2.75.3), libreadline8 (>= 6.0), libudev1 (>= 196), kmod, udev, lsb-base, dbus",
            "conffiles": [
                "/etc/bluetooth/input.conf",
                "/etc/bluetooth/main.conf",
                "/etc/bluetooth/network.conf",
                "/etc/dbus-1/system.d/bluetooth.conf",
                "/etc/init.d/bluetooth"
            ],
            "pre_depends": "",
            "description": "Bluetooth tools and daemons This package contains tools and system daemons for using Bluetooth devices. . BlueZ is the official Linux Bluetooth protocol stack. It is an Open Source project distributed under GNU General Public License (GPL).",
            "source": "",
            "homepage": "http://www.bluez.org"
        },
        {
            "package": "bluez-cups",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Bluetooth team <ubuntu-bluetooth@lists.ubuntu.com>",
            "version": "5.68-0ubuntu1.1",
            "section": "admin",
            "installed_size": 71,
            "depends": "libc6 (>= 2.38), libdbus-1-3 (>= 1.9.14), libglib2.0-0 (>= 2.28), cups",
            "conffiles": [],
            "pre_depends": "",
            "description": "Bluetooth printer driver for CUPS This package contains a driver to let CUPS print to Bluetooth-connected printers. . BlueZ is the official Linux Bluetooth protocol stack. It is an Open Source project distributed under GNU General Public License (GPL).",
            "source": "bluez",
            "homepage": "http://www.bluez.org"
        },
        {
            "package": "bluez-obexd",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Bluetooth team <ubuntu-bluetooth@lists.ubuntu.com>",
            "version": "5.68-0ubuntu1.1",
            "section": "admin",
            "installed_size": 667,
            "depends": "libc6 (>= 2.38), libdbus-1-3 (>= 1.9.14), libglib2.0-0 (>= 2.77.0), libical3 (>= 3.0.0)",
            "conffiles": [],
            "pre_depends": "",
            "description": "bluez obex daemon This package contains a OBEX(OBject EXchange) daemon. . OBEX is communication protocol to facilitate the exchange of the binary object between the devices. . This was the software that is independent as obexd, but this has been integrated into BlueZ from BlueZ 5.0. . BlueZ is the official Linux Bluetooth protocol stack. It is an Open Source project distributed under GNU General Public License (GPL).",
            "source": "bluez",
            "homepage": "http://www.bluez.org"
        },
        {
            "package": "bolt",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "0.9.5-1",
            "section": "admin",
            "installed_size": 468,
            "depends": "libc6 (>= 2.34), libglib2.0-0 (>= 2.55.2), libpolkit-gobject-1-0 (>= 0.99), libudev1 (>= 183)",
            "conffiles": [],
            "pre_depends": "",
            "description": "system daemon to manage thunderbolt 3 devices Thunderbolt 3 features different security modes that require devices to be authorized before they can be used. The D-Bus API can be used to list devices, enroll them (authorize and store them in the local database) and forget them again (remove previously enrolled devices). It also emits signals if new devices are connected (or removed). During enrollment devices can be set to be automatically authorized as soon as they are connected.  A command line tool, called boltctl, can be used to control the daemon and perform all the above mentioned tasks.",
            "source": "",
            "homepage": "https://gitlab.freedesktop.org/bolt/bolt"
        },
        {
            "package": "branding-ubuntu",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "all",
            "multi_arch": "",
            "maintainer": "Scott Ritchie <scottritchie@ubuntu.com>",
            "version": "0.11",
            "section": "graphics",
            "installed_size": 472,
            "depends": "",
            "conffiles": [],
            "pre_depends": "",
            "description": "Replacement artwork with Ubuntu branding The branding-ubuntu package is a series of replacement artworks for packages to make them more Ubuntu specific and fit in with the overall theme.  Removal of the branding package should cause branded applications to fall back to their default artwork.",
            "source": "",
            "homepage": "https://wiki.ubuntu.com/branding"
        },
        {
            "package": "bridge-utils",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "1.7.1-1ubuntu1",
            "section": "net",
            "installed_size": 114,
            "depends": "libc6 (>= 2.34)",
            "conffiles": [
                "/etc/default/bridge-utils"
            ],
            "pre_depends": "",
            "description": "Utilities for configuring the Linux Ethernet bridge This package contains utilities for configuring the Linux Ethernet bridge in Linux. The Linux Ethernet bridge can be used for connecting multiple Ethernet devices together. The connecting is fully transparent: hosts connected to one Ethernet device see hosts connected to the other Ethernet devices directly.",
            "source": "",
            "homepage": ""
        },
        {
            "package": "brltty",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "6.5-7ubuntu2",
            "section": "admin",
            "installed_size": 10000,
            "depends": "libasound2 (>= 1.0.16), libbluetooth3 (>= 4.91), libbrlapi0.8 (>= 6.5), libc6 (>= 2.38), libcap2 (>= 1:2.10), libdbus-1-3 (>= 1.9.14), libexpat1 (>= 2.0.1), libglib2.0-0 (>= 2.16.0), libgpm2 (>= 1.20.7), libicu72 (>= 72.1~rc-1~), liblouis20 (>= 3.26.0), libncursesw6 (>= 6), libpcre2-32-0 (>= 10.22), libpolkit-gobject-1-0 (>= 0.101), libsystemd0, libtinfo6 (>= 6), lsb-base, polkitd | policykit-1, initramfs-tools (>= 0.40ubuntu30)",
            "conffiles": [
                "/etc/brltty.conf",
                "/etc/brltty/Attributes/invleft_right.atb",
                "/etc/brltty/Attributes/left_right.atb",
                "/etc/brltty/Attributes/upper_lower.atb",
                "/etc/brltty/Contraction/af.ctb",
                "/etc/brltty/Contraction/am.ctb",
                "/etc/brltty/Contraction/countries.cti",
                "/etc/brltty/Contraction/de-1998.ctb",
                "/etc/brltty/Contraction/de-2015.ctb",
                "/etc/brltty/Contraction/de-g0.ctb",
                "/etc/brltty/Contraction/de-g1.ctb",
                "/etc/brltty/Contraction/de-g2.ctb",
                "/etc/brltty/Contraction/de-wort.cti",
                "/etc/brltty/Contraction/de.ctb",
                "/etc/brltty/Contraction/en-ueb-g2.ctb",
                "/etc/brltty/Contraction/en-us-g2.ctb",
                "/etc/brltty/Contraction/en.ctb",
                "/etc/brltty/Contraction/en_US.ctb",
                "/etc/brltty/Contraction/es.ctb",
                "/etc/brltty/Contraction/fr-g1.ctb",
                "/etc/brltty/Contraction/fr-g2.ctb",
                "/etc/brltty/Contraction/fr.ctb",
                "/etc/brltty/Contraction/ha.ctb",
                "/etc/brltty/Contraction/id.ctb",
                "/etc/brltty/Contraction/ipa.ctb",
                "/etc/brltty/Contraction/ja.ctb",
                "/etc/brltty/Contraction/ko-g0.ctb",
                "/etc/brltty/Contraction/ko-g1.ctb",
                "/etc/brltty/Contraction/ko-g2.ctb",
                "/etc/brltty/Contraction/ko.ctb",
                "/etc/brltty/Contraction/latex-access.ctb",
                "/etc/brltty/Contraction/letters-latin.cti",
                "/etc/brltty/Contraction/lt.ctb",
                "/etc/brltty/Contraction/mg.ctb",
                "/etc/brltty/Contraction/mun.ctb",
                "/etc/brltty/Contraction/nabcc.cti",
                "/etc/brltty/Contraction/nl.ctb",
                "/etc/brltty/Contraction/none.ctb",
                "/etc/brltty/Contraction/ny.ctb",
                "/etc/brltty/Contraction/pt.ctb",
                "/etc/brltty/Contraction/ru.ctb",
                "/etc/brltty/Contraction/si.ctb",
                "/etc/brltty/Contraction/spaces.cti",
                "/etc/brltty/Contraction/sw.ctb",
                "/etc/brltty/Contraction/th.ctb",
                "/etc/brltty/Contraction/zh-tw.ctb",
                "/etc/brltty/Contraction/zh-tw.cti",
                "/etc/brltty/Contraction/zh_TW.ctb",
                "/etc/brltty/Contraction/zu.ctb",
                "/etc/brltty/Input/al/abt_basic.kti",
                "/etc/brltty/Input/al/abt_extra.kti",
                "/etc/brltty/Input/al/abt_large.ktb",
                "/etc/brltty/Input/al/abt_small.ktb",
                "/etc/brltty/Input/al/bc-etouch.kti",
                "/etc/brltty/Input/al/bc-smartpad.kti",
                "/etc/brltty/Input/al/bc-thumb.kti",
                "/etc/brltty/Input/al/bc.kti",
                "/etc/brltty/Input/al/bc640.ktb",
                "/etc/brltty/Input/al/bc680.ktb",
                "/etc/brltty/Input/al/el.ktb",
                "/etc/brltty/Input/al/sat_common.kti",
                "/etc/brltty/Input/al/sat_large.ktb",
                "/etc/brltty/Input/al/sat_nav.kti",
                "/etc/brltty/Input/al/sat_small.ktb",
                "/etc/brltty/Input/al/sat_speech.kti",
                "/etc/brltty/Input/al/sat_tumblers.kti",
                "/etc/brltty/Input/al/voyager.ktb",
                "/etc/brltty/Input/android-chords.kti",
                "/etc/brltty/Input/at/all.ktb",
                "/etc/brltty/Input/ba/all.txt",
                "/etc/brltty/Input/bd/all.txt",
                "/etc/brltty/Input/bg/all.ktb",
                "/etc/brltty/Input/bl/18.txt",
                "/etc/brltty/Input/bl/40_m20_m40.txt",
                "/etc/brltty/Input/bm/NLS_Zoomax.ktb",
                "/etc/brltty/Input/bm/b2g.ktb",
                "/etc/brltty/Input/bm/b9b10.kti",
                "/etc/brltty/Input/bm/b9b11b10.kti",
                "/etc/brltty/Input/bm/command.kti",
                "/etc/brltty/Input/bm/connect.ktb",
                "/etc/brltty/Input/bm/conny.ktb",
                "/etc/brltty/Input/bm/d6.kti",
                "/etc/brltty/Input/bm/default.ktb",
                "/etc/brltty/Input/bm/display6.kti",
                "/etc/brltty/Input/bm/display7.kti",
                "/etc/brltty/Input/bm/dm80p.ktb",
                "/etc/brltty/Input/bm/emulate6.kti",
                "/etc/brltty/Input/bm/front10.kti",
                "/etc/brltty/Input/bm/front6.kti",
                "/etc/brltty/Input/bm/horizontal.kti",
                "/etc/brltty/Input/bm/inka.ktb",
                "/etc/brltty/Input/bm/joystick.kti",
                "/etc/brltty/Input/bm/keyboard.kti",
                "/etc/brltty/Input/bm/navpad.kti",
                "/etc/brltty/Input/bm/orbit.ktb",
                "/etc/brltty/Input/bm/pro.ktb",
                "/etc/brltty/Input/bm/pronto.ktb",
                "/etc/brltty/Input/bm/pv.ktb",
                "/etc/brltty/Input/bm/rb.ktb",
                "/etc/brltty/Input/bm/routing.kti",
                "/etc/brltty/Input/bm/routing6.kti",
                "/etc/brltty/Input/bm/routing7.kti",
                "/etc/brltty/Input/bm/status.kti",
                "/etc/brltty/Input/bm/sv.ktb",
                "/etc/brltty/Input/bm/ultra.ktb",
                "/etc/brltty/Input/bm/v40.ktb",
                "/etc/brltty/Input/bm/v80.ktb",
                "/etc/brltty/Input/bm/vertical.kti",
                "/etc/brltty/Input/bm/vk.ktb",
                "/etc/brltty/Input/bm/wheels.kti",
                "/etc/brltty/Input/bn/all.ktb",
                "/etc/brltty/Input/bn/input.kti",
                "/etc/brltty/Input/bp/all.kti",
                "/etc/brltty/Input/cb/all.ktb",
                "/etc/brltty/Input/ce/all.ktb",
                "/etc/brltty/Input/ce/novem.ktb",
                "/etc/brltty/Input/chords.kti",
                "/etc/brltty/Input/cn/all.ktb",
                "/etc/brltty/Input/ec/all.txt",
                "/etc/brltty/Input/ec/spanish.txt",
                "/etc/brltty/Input/eu/all.txt",
                "/etc/brltty/Input/eu/braille.kti",
                "/etc/brltty/Input/eu/clio.ktb",
                "/etc/brltty/Input/eu/common.kti",
                "/etc/brltty/Input/eu/esys_large.ktb",
                "/etc/brltty/Input/eu/esys_medium.ktb",
                "/etc/brltty/Input/eu/esys_small.ktb",
                "/etc/brltty/Input/eu/esytime.ktb",
                "/etc/brltty/Input/eu/iris.ktb",
                "/etc/brltty/Input/eu/joysticks.kti",
                "/etc/brltty/Input/eu/routing.kti",
                "/etc/brltty/Input/eu/sw12.kti",
                "/etc/brltty/Input/eu/sw34.kti",
                "/etc/brltty/Input/eu/sw56.kti",
                "/etc/brltty/Input/fa/all.ktb",
                "/etc/brltty/Input/fs/bumpers.kti",
                "/etc/brltty/Input/fs/common.kti",
                "/etc/brltty/Input/fs/focus.kti",
                "/etc/brltty/Input/fs/focus1.ktb",
                "/etc/brltty/Input/fs/focus14.ktb",
                "/etc/brltty/Input/fs/focus40.ktb",
                "/etc/brltty/Input/fs/focus80.ktb",
                "/etc/brltty/Input/fs/keyboard.kti",
                "/etc/brltty/Input/fs/pacmate.ktb",
                "/etc/brltty/Input/fs/rockers.kti",
                "/etc/brltty/Input/fs/speech.kti",
                "/etc/brltty/Input/hd/mbl.ktb",
                "/etc/brltty/Input/hd/pfl.ktb",
                "/etc/brltty/Input/hm/beetle.ktb",
                "/etc/brltty/Input/hm/braille.kti",
                "/etc/brltty/Input/hm/common.kti",
                "/etc/brltty/Input/hm/contexts.kti",
                "/etc/brltty/Input/hm/edge.ktb",
                "/etc/brltty/Input/hm/f14.kti",
                "/etc/brltty/Input/hm/f18.kti",
                "/etc/brltty/Input/hm/fnkey.kti",
                "/etc/brltty/Input/hm/left.kti",
                "/etc/brltty/Input/hm/letters.kti",
                "/etc/brltty/Input/hm/pan.ktb",
                "/etc/brltty/Input/hm/pan.kti",
                "/etc/brltty/Input/hm/qwerty.ktb",
                "/etc/brltty/Input/hm/qwerty.kti",
                "/etc/brltty/Input/hm/right.kti",
                "/etc/brltty/Input/hm/scroll.ktb",
                "/etc/brltty/Input/hm/scroll.kti",
                "/etc/brltty/Input/hm/sync.ktb",
                "/etc/brltty/Input/ht/ab.ktb",
                "/etc/brltty/Input/ht/ab.kti",
                "/etc/brltty/Input/ht/ab_s.ktb",
                "/etc/brltty/Input/ht/ac4.ktb",
                "/etc/brltty/Input/ht/alo.ktb",
                "/etc/brltty/Input/ht/as40.ktb",
                "/etc/brltty/Input/ht/bb.ktb",
                "/etc/brltty/Input/ht/bbp.ktb",
                "/etc/brltty/Input/ht/bkwm.ktb",
                "/etc/brltty/Input/ht/brln.ktb",
                "/etc/brltty/Input/ht/bs.kti",
                "/etc/brltty/Input/ht/bs40.ktb",
                "/etc/brltty/Input/ht/bs80.ktb",
                "/etc/brltty/Input/ht/cb40.ktb",
                "/etc/brltty/Input/ht/dots.kti",
                "/etc/brltty/Input/ht/easy.ktb",
                "/etc/brltty/Input/ht/input.kti",
                "/etc/brltty/Input/ht/joystick.kti",
                "/etc/brltty/Input/ht/keypad.kti",
                "/etc/brltty/Input/ht/mc88.ktb",
                "/etc/brltty/Input/ht/mdlr.ktb",
                "/etc/brltty/Input/ht/me.kti",
                "/etc/brltty/Input/ht/me64.ktb",
                "/etc/brltty/Input/ht/me88.ktb",
                "/etc/brltty/Input/ht/rockers.kti",
                "/etc/brltty/Input/ht/wave.ktb",
                "/etc/brltty/Input/hw/B80.ktb",
                "/etc/brltty/Input/hw/BI14.ktb",
                "/etc/brltty/Input/hw/BI20X.ktb",
                "/etc/brltty/Input/hw/BI32.ktb",
                "/etc/brltty/Input/hw/BI40.ktb",
                "/etc/brltty/Input/hw/BI40X.ktb",
                "/etc/brltty/Input/hw/C20.ktb",
                "/etc/brltty/Input/hw/M40.ktb",
                "/etc/brltty/Input/hw/NLS.ktb",
                "/etc/brltty/Input/hw/braille.kti",
                "/etc/brltty/Input/hw/command.kti",
                "/etc/brltty/Input/hw/joystick.kti",
                "/etc/brltty/Input/hw/one.ktb",
                "/etc/brltty/Input/hw/routing.kti",
                "/etc/brltty/Input/hw/thumb.kti",
                "/etc/brltty/Input/hw/thumb_legacy.kti",
                "/etc/brltty/Input/hw/touch.ktb",
                "/etc/brltty/Input/ic/bb.ktb",
                "/etc/brltty/Input/ic/chords.kti",
                "/etc/brltty/Input/ic/common.kti",
                "/etc/brltty/Input/ic/nvda.ktb",
                "/etc/brltty/Input/ic/route.kti",
                "/etc/brltty/Input/ic/toggle.kti",
                "/etc/brltty/Input/ir/all.kti",
                "/etc/brltty/Input/ir/brl.ktb",
                "/etc/brltty/Input/ir/pc.ktb",
                "/etc/brltty/Input/lb/all.txt",
                "/etc/brltty/Input/lt/all.txt",
                "/etc/brltty/Input/mb/all.txt",
                "/etc/brltty/Input/md/common.kti",
                "/etc/brltty/Input/md/default.ktb",
                "/etc/brltty/Input/md/fk.ktb",
                "/etc/brltty/Input/md/fk_s.ktb",
                "/etc/brltty/Input/md/fkeys.kti",
                "/etc/brltty/Input/md/kbd.ktb",
                "/etc/brltty/Input/md/keyboard.kti",
                "/etc/brltty/Input/md/status.kti",
                "/etc/brltty/Input/menu.kti",
                "/etc/brltty/Input/mm/common.kti",
                "/etc/brltty/Input/mm/pocket.ktb",
                "/etc/brltty/Input/mm/smart.ktb",
                "/etc/brltty/Input/mn/all.txt",
                "/etc/brltty/Input/mt/bd1_3.ktb",
                "/etc/brltty/Input/mt/bd1_3.kti",
                "/etc/brltty/Input/mt/bd1_3s.ktb",
                "/etc/brltty/Input/mt/bd1_6.ktb",
                "/etc/brltty/Input/mt/bd1_6.kti",
                "/etc/brltty/Input/mt/bd1_6s.ktb",
                "/etc/brltty/Input/mt/bd2.ktb",
                "/etc/brltty/Input/mt/status.kti",
                "/etc/brltty/Input/nav.kti",
                "/etc/brltty/Input/no/all.txt",
                "/etc/brltty/Input/np/all.ktb",
                "/etc/brltty/Input/pg/all.ktb",
                "/etc/brltty/Input/pm/2d_l.ktb",
                "/etc/brltty/Input/pm/2d_s.ktb",
                "/etc/brltty/Input/pm/bar.kti",
                "/etc/brltty/Input/pm/c.ktb",
                "/etc/brltty/Input/pm/c_486.ktb",
                "/etc/brltty/Input/pm/el2d_80s.ktb",
                "/etc/brltty/Input/pm/el40c.ktb",
                "/etc/brltty/Input/pm/el40s.ktb",
                "/etc/brltty/Input/pm/el60c.ktb",
                "/etc/brltty/Input/pm/el66s.ktb",
                "/etc/brltty/Input/pm/el70s.ktb",
                "/etc/brltty/Input/pm/el80_ii.ktb",
                "/etc/brltty/Input/pm/el80c.ktb",
                "/etc/brltty/Input/pm/el80s.ktb",
                "/etc/brltty/Input/pm/el_2d_40.ktb",
                "/etc/brltty/Input/pm/el_2d_66.ktb",
                "/etc/brltty/Input/pm/el_2d_80.ktb",
                "/etc/brltty/Input/pm/el_40_p.ktb",
                "/etc/brltty/Input/pm/el_80.ktb",
                "/etc/brltty/Input/pm/elb_tr_20.ktb",
                "/etc/brltty/Input/pm/elb_tr_32.ktb",
                "/etc/brltty/Input/pm/elba_20.ktb",
                "/etc/brltty/Input/pm/elba_32.ktb",
                "/etc/brltty/Input/pm/front13.kti",
                "/etc/brltty/Input/pm/front9.kti",
                "/etc/brltty/Input/pm/ib_80.ktb",
                "/etc/brltty/Input/pm/keyboard.kti",
                "/etc/brltty/Input/pm/keys.kti",
                "/etc/brltty/Input/pm/live.ktb",
                "/etc/brltty/Input/pm/routing.kti",
                "/etc/brltty/Input/pm/status0.kti",
                "/etc/brltty/Input/pm/status13.kti",
                "/etc/brltty/Input/pm/status2.kti",
                "/etc/brltty/Input/pm/status20.kti",
                "/etc/brltty/Input/pm/status22.kti",
                "/etc/brltty/Input/pm/status4.kti",
                "/etc/brltty/Input/pm/switches.kti",
                "/etc/brltty/Input/pm/trio.ktb",
                "/etc/brltty/Input/sk/bdp.ktb",
                "/etc/brltty/Input/sk/ntk.ktb",
                "/etc/brltty/Input/speech.kti",
                "/etc/brltty/Input/tn/all.txt",
                "/etc/brltty/Input/toggle.kti",
                "/etc/brltty/Input/ts/nav.kti",
                "/etc/brltty/Input/ts/nav20.ktb",
                "/etc/brltty/Input/ts/nav40.ktb",
                "/etc/brltty/Input/ts/nav80.ktb",
                "/etc/brltty/Input/ts/nav_large.kti",
                "/etc/brltty/Input/ts/nav_small.kti",
                "/etc/brltty/Input/ts/pb.kti",
                "/etc/brltty/Input/ts/pb40.ktb",
                "/etc/brltty/Input/ts/pb65.ktb",
                "/etc/brltty/Input/ts/pb80.ktb",
                "/etc/brltty/Input/ts/pb_large.kti",
                "/etc/brltty/Input/ts/pb_small.kti",
                "/etc/brltty/Input/ts/routing.kti",
                "/etc/brltty/Input/tt/all.txt",
                "/etc/brltty/Input/vd/all.txt",
                "/etc/brltty/Input/vo/all.ktb",
                "/etc/brltty/Input/vo/all.kti",
                "/etc/brltty/Input/vo/bp.ktb",
                "/etc/brltty/Input/vr/all.txt",
                "/etc/brltty/Input/vs/all.txt",
                "/etc/brltty/Keyboard/braille.ktb",
                "/etc/brltty/Keyboard/braille.kti",
                "/etc/brltty/Keyboard/desktop.ktb",
                "/etc/brltty/Keyboard/desktop.kti",
                "/etc/brltty/Keyboard/keypad.ktb",
                "/etc/brltty/Keyboard/kp_say.kti",
                "/etc/brltty/Keyboard/kp_speak.kti",
                "/etc/brltty/Keyboard/laptop.ktb",
                "/etc/brltty/Keyboard/sun_type6.ktb",
                "/etc/brltty/Text/alias.tti",
                "/etc/brltty/Text/ar.ttb",
                "/etc/brltty/Text/as.ttb",
                "/etc/brltty/Text/ascii-basic.tti",
                "/etc/brltty/Text/awa.ttb",
                "/etc/brltty/Text/bengali.tti",
                "/etc/brltty/Text/bg.ttb",
                "/etc/brltty/Text/bh.ttb",
                "/etc/brltty/Text/blocks.tti",
                "/etc/brltty/Text/bn.ttb",
                "/etc/brltty/Text/bo.ttb",
                "/etc/brltty/Text/boxes.tti",
                "/etc/brltty/Text/bra.ttb",
                "/etc/brltty/Text/brf.ttb",
                "/etc/brltty/Text/common.tti",
                "/etc/brltty/Text/cs.ttb",
                "/etc/brltty/Text/ctl-latin.tti",
                "/etc/brltty/Text/cy.ttb",
                "/etc/brltty/Text/da-1252.ttb",
                "/etc/brltty/Text/da-lt.ttb",
                "/etc/brltty/Text/da.ttb",
                "/etc/brltty/Text/de-chess.tti",
                "/etc/brltty/Text/de.ttb",
                "/etc/brltty/Text/devanagari.tti",
                "/etc/brltty/Text/dra.ttb",
                "/etc/brltty/Text/el.ttb",
                "/etc/brltty/Text/en-chess.tti",
                "/etc/brltty/Text/en-na-ascii.tti",
                "/etc/brltty/Text/en-nabcc.ttb",
                "/etc/brltty/Text/en.ttb",
                "/etc/brltty/Text/en_CA.ttb",
                "/etc/brltty/Text/en_GB.ttb",
                "/etc/brltty/Text/en_US.ttb",
                "/etc/brltty/Text/eo.ttb",
                "/etc/brltty/Text/es.ttb",
                "/etc/brltty/Text/et.ttb",
                "/etc/brltty/Text/fi.ttb",
                "/etc/brltty/Text/fr-2007.ttb",
                "/etc/brltty/Text/fr-cbifs.ttb",
                "/etc/brltty/Text/fr-vs.ttb",
                "/etc/brltty/Text/fr.ttb",
                "/etc/brltty/Text/fr_CA.ttb",
                "/etc/brltty/Text/fr_FR.ttb",
                "/etc/brltty/Text/ga.ttb",
                "/etc/brltty/Text/gd.ttb",
                "/etc/brltty/Text/gon.ttb",
                "/etc/brltty/Text/greek.tti",
                "/etc/brltty/Text/gu.ttb",
                "/etc/brltty/Text/gujarati.tti",
                "/etc/brltty/Text/gurmukhi.tti",
                "/etc/brltty/Text/he.ttb",
                "/etc/brltty/Text/hi.ttb",
                "/etc/brltty/Text/hr.ttb",
                "/etc/brltty/Text/hu.ttb",
                "/etc/brltty/Text/hy.ttb",
                "/etc/brltty/Text/is.ttb",
                "/etc/brltty/Text/it.ttb",
                "/etc/brltty/Text/kannada.tti",
                "/etc/brltty/Text/kha.ttb",
                "/etc/brltty/Text/kn.ttb",
                "/etc/brltty/Text/kok.ttb",
                "/etc/brltty/Text/kru.ttb",
                "/etc/brltty/Text/lt.ttb",
                "/etc/brltty/Text/ltr-alias.tti",
                "/etc/brltty/Text/ltr-cyrillic.tti",
                "/etc/brltty/Text/ltr-dot8.tti",
                "/etc/brltty/Text/ltr-latin.tti",
                "/etc/brltty/Text/ltr-tibetan.tti",
                "/etc/brltty/Text/lv.ttb",
                "/etc/brltty/Text/malayalam.tti",
                "/etc/brltty/Text/mg.ttb",
                "/etc/brltty/Text/mi.ttb",
                "/etc/brltty/Text/ml.ttb",
                "/etc/brltty/Text/mni.ttb",
                "/etc/brltty/Text/mr.ttb",
                "/etc/brltty/Text/mt.ttb",
                "/etc/brltty/Text/mun.ttb",
                "/etc/brltty/Text/mwr.ttb",
                "/etc/brltty/Text/ne.ttb",
                "/etc/brltty/Text/new.ttb",
                "/etc/brltty/Text/nl.ttb",
                "/etc/brltty/Text/nl_BE.ttb",
                "/etc/brltty/Text/nl_NL.ttb",
                "/etc/brltty/Text/no-generic.ttb",
                "/etc/brltty/Text/no-oup.ttb",
                "/etc/brltty/Text/no.ttb",
                "/etc/brltty/Text/num-alias.tti",
                "/etc/brltty/Text/num-dot6.tti",
                "/etc/brltty/Text/num-dot8.tti",
                "/etc/brltty/Text/num-french.tti",
                "/etc/brltty/Text/num-nemd8.tti",
                "/etc/brltty/Text/num-nemeth.tti",
                "/etc/brltty/Text/nwc.ttb",
                "/etc/brltty/Text/or.ttb",
                "/etc/brltty/Text/oriya.tti",
                "/etc/brltty/Text/pa.ttb",
                "/etc/brltty/Text/pi.ttb",
                "/etc/brltty/Text/pl.ttb",
                "/etc/brltty/Text/pt.ttb",
                "/etc/brltty/Text/punc-alternate.tti",
                "/etc/brltty/Text/punc-basic.tti",
                "/etc/brltty/Text/punc-tibetan.tti",
                "/etc/brltty/Text/ro.ttb",
                "/etc/brltty/Text/ru.ttb",
                "/etc/brltty/Text/sa.ttb",
                "/etc/brltty/Text/sat.ttb",
                "/etc/brltty/Text/sd.ttb",
                "/etc/brltty/Text/se.ttb",
                "/etc/brltty/Text/sk.ttb",
                "/etc/brltty/Text/sl.ttb",
                "/etc/brltty/Text/sv-1989.ttb",
                "/etc/brltty/Text/sv-1996.ttb",
                "/etc/brltty/Text/sv.ttb",
                "/etc/brltty/Text/sw.ttb",
                "/etc/brltty/Text/ta.ttb",
                "/etc/brltty/Text/tamil.tti",
                "/etc/brltty/Text/te.ttb",
                "/etc/brltty/Text/telugu.tti",
                "/etc/brltty/Text/tr.ttb",
                "/etc/brltty/Text/uk.ttb",
                "/etc/brltty/Text/vi.ttb",
                "/etc/brltty/Text/win-1252.tti"
            ],
            "pre_depends": "",
            "description": "Access software for a blind person using a braille display BRLTTY is a daemon which provides access to the console (text mode) for a blind person using a braille display.  It drives the braille display and provides complete screen review functionality. The following display models are supported:  * Alva/Optelec (ABT3xx, Delphi, Satellite, Braille System 40, BC 640/680)  * Baum  * BrailComm  * BrailleLite  * BrailleNote  * Cebra  * EcoBraille  * EuroBraille (AzerBraille, Clio, Esys, Iris, NoteBraille, Scriba)  * Freedom Scientific (Focus and PacMate)  * Handy Tech  * HIMS (Braille Sense, SyncBraille)  * HumanWare (Brailliant)  * Iris  * LogText 32  * MDV  * Metec (BD-40)  * NinePoint  * Papenmeier  * Pegasus  * Seika  * Tieman (Voyager, CombiBraille, MiniBraille, MultiBraille,            BraillePen/EasyLink)  * Tivomatic (Albatross)  * TSI (Navigator, PowerBraille)  * VideoBraille  * VisioBraille . BRLTTY also provides a client/server based infrastructure for applications wishing to utilize a Braille display.  The daemon process listens for incoming TCP/IP connections on a certain port.  A shared object library for clients is provided in the package libbrlapi0.8.  A static library, header files and documentation is provided in package libbrlapi-dev.  Bindings to other programming languages can be found in cl-brlapi (Lisp), libbrlapi-java (Java) and python3-brlapi (Python).",
            "source": "",
            "homepage": "https://brltty.com"
        },
        {
            "package": "bsdextrautils",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "2.39.1-4ubuntu2",
            "section": "utils",
            "installed_size": 289,
            "depends": "libc6 (>= 2.38), libsmartcols1 (>= 2.39.1), libtinfo6 (>= 6)",
            "conffiles": [],
            "pre_depends": "",
            "description": "extra utilities from 4.4BSD-Lite This package contains some extra BSD utilities: col, colcrt, colrm, column, hd, hexdump, look, ul and write. Other BSD utilities are provided by bsdutils and calendar.",
            "source": "util-linux",
            "homepage": "https://www.kernel.org/pub/linux/utils/util-linux/"
        },
        {
            "package": "bsdutils",
            "status": "install ok installed",
            "priority": "required",
            "architecture": "amd64",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "1:2.39.1-4ubuntu2",
            "section": "utils",
            "installed_size": 286,
            "depends": "",
            "conffiles": [],
            "pre_depends": "libc6 (>= 2.38), libsystemd0",
            "description": "basic utilities from 4.4BSD-Lite This package contains the bare minimum of BSD utilities needed for a Debian system: logger, renice, script, scriptlive, scriptreplay and wall. The remaining standard BSD utilities are provided by bsdextrautils.",
            "source": "util-linux (2.39.1-4ubuntu2)",
            "homepage": "https://www.kernel.org/pub/linux/utils/util-linux/"
        },
        {
            "package": "bubblewrap",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "0.8.0-2",
            "section": "admin",
            "installed_size": 126,
            "depends": "libc6 (>= 2.34), libcap2 (>= 1:2.10), libselinux1 (>= 3.1~)",
            "conffiles": [],
            "pre_depends": "",
            "description": "utility for unprivileged chroot and namespace manipulation bubblewrap uses Linux namespaces to launch unprivileged containers. These containers can be used to sandbox semi-trusted applications such as Flatpak apps, image/video thumbnailers and web browser components, or to run programs in a different library stack such as a Flatpak runtime or a different Debian release. . By default, this package relies on a kernel with user namespaces enabled. Official Debian and Ubuntu kernels are suitable. . On kernels without user namespaces, system administrators can make the bwrap executable setuid root, allowing it to create unprivileged containers even though ordinary user processes cannot.",
            "source": "",
            "homepage": "https://github.com/containers/bubblewrap"
        },
        {
            "package": "build-essential",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "12.10ubuntu1",
            "section": "devel",
            "installed_size": 17,
            "depends": "libc6-dev | libc-dev, gcc (>= 4:12.3), g++ (>= 4:12.3), make, dpkg-dev (>= 1.17.11)",
            "conffiles": [],
            "pre_depends": "",
            "description": "Informational list of build-essential packages If you do not plan to build Debian packages, you don't need this package.  Starting with dpkg (>= 1.14.18) this package is required for building Debian packages. . This package contains an informational list of packages which are considered essential for building Debian packages.  This package also depends on the packages on that list, to make it easy to have the build-essential packages installed. . If you have this package installed, you only need to install whatever a package specifies as its build-time dependencies to build the package.  Conversely, if you are determining what your package needs to build-depend on, you can always leave out the packages this package depends on. . This package is NOT the definition of what packages are build-essential; the real definition is in the Debian Policy Manual. This package contains merely an informational list, which is all most people need.   However, if this package and the manual disagree, the manual is correct.",
            "source": "",
            "homepage": ""
        },
        {
            "package": "busybox-initramfs",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "1:1.36.1-3ubuntu1",
            "section": "shells",
            "installed_size": 344,
            "depends": "libc6 (>= 2.34)",
            "conffiles": [],
            "pre_depends": "",
            "description": "Standalone shell setup for initramfs BusyBox combines tiny versions of many common UNIX utilities into a single small executable. It provides minimalist replacements for the most common utilities you would usually find on your desktop system (i.e., ls, cp, mv, mount, tar, etc.). The utilities in BusyBox generally have fewer options than their full-featured GNU cousins; however, the options that are included provide the expected functionality and behave very much like their GNU counterparts. . busybox-initramfs provides a simple stand alone shell that provides only the basic utilities needed for the initramfs.",
            "source": "busybox",
            "homepage": "http://www.busybox.net"
        },
        {
            "package": "busybox-static",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "1:1.36.1-3ubuntu1",
            "section": "shells",
            "installed_size": 2116,
            "depends": "",
            "conffiles": [],
            "pre_depends": "",
            "description": "Standalone rescue shell with tons of builtin utilities BusyBox combines tiny versions of many common UNIX utilities into a single small executable. It provides minimalist replacements for the most common utilities you would usually find on your desktop system (i.e., ls, cp, mv, mount, tar, etc.).  The utilities in BusyBox generally have fewer options than their full-featured GNU cousins; however, the options that are included provide the expected functionality and behave very much like their GNU counterparts. . busybox-static provides you with a statically linked simple stand alone shell that provides all the utilities available in BusyBox. This package is intended to be used as a rescue shell, in the event that you screw up your system. Invoke \"busybox sh\" and you have a standalone shell ready to save your system from certain destruction. Invoke \"busybox\", and it will list the available builtin commands.",
            "source": "busybox",
            "homepage": "http://www.busybox.net"
        },
        {
            "package": "bzip2",
            "status": "install ok installed",
            "priority": "important",
            "architecture": "amd64",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "1.0.8-5build1",
            "section": "utils",
            "installed_size": 114,
            "depends": "libbz2-1.0 (= 1.0.8-5build1), libc6 (>= 2.34)",
            "conffiles": [],
            "pre_depends": "",
            "description": "high-quality block-sorting file compressor - utilities bzip2 is a freely available, patent free, data compressor. . bzip2 compresses files using the Burrows-Wheeler block-sorting text compression algorithm, and Huffman coding.  Compression is generally considerably better than that achieved by more conventional LZ77/LZ78-based compressors, and approaches the performance of the PPM family of statistical compressors. . The archive file format of bzip2 (.bz2) is incompatible with that of its predecessor, bzip (.bz).",
            "source": "",
            "homepage": "https://sourceware.org/bzip2/"
        },
        {
            "package": "ca-certificates",
            "status": "install ok installed",
            "priority": "important",
            "architecture": "all",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "20230311ubuntu1",
            "section": "misc",
            "installed_size": 377,
            "depends": "openssl (>= 1.1.1), debconf (>= 0.5) | debconf-2.0",
            "conffiles": [],
            "pre_depends": "",
            "description": "Common CA certificates Contains the certificate authorities shipped with Mozilla's browser to allow SSL-based applications to check for the authenticity of SSL connections. . Please note that Debian can neither confirm nor deny whether the certificate authorities whose certificates are included in this package have in any way been audited for trustworthiness or RFC 3647 compliance. Full responsibility to assess them belongs to the local system administrator.",
            "source": "",
            "homepage": ""
        },
        {
            "package": "ca-certificates-java",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "all",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "20230710",
            "section": "java",
            "installed_size": 43,
            "depends": "ca-certificates (>= 20210120)",
            "conffiles": [
                "/etc/ca-certificates/update.d/jks-keystore",
                "/etc/default/cacerts"
            ],
            "pre_depends": "",
            "description": "Common CA certificates (JKS keystore) This package uses the hooks of the ca-certificates package to update the cacerts JKS keystore used for many java runtimes.",
            "source": "",
            "homepage": ""
        },
        {
            "package": "calendar",
            "status": "deinstall ok config-files",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "12.1.7",
            "section": "utils",
            "installed_size": 411,
            "depends": "libbsd0 (>= 0.2.0), libc6 (>= 2.4), cpp",
            "conffiles": [],
            "pre_depends": "",
            "description": "display upcoming dates and provide reminders This package contains the \"calendar\" program commonly found on BSD-style systems, which displays upcoming relevant dates on a wide variety of calendars.",
            "source": "bsdmainutils",
            "homepage": ""
        },
        {
            "package": "cheese",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "44.1-1",
            "section": "gnome",
            "installed_size": 453,
            "depends": "libc6 (>= 2.34), libcanberra-gtk3-0 (>= 0.25), libcheese-gtk25 (= 44.1-1), libcheese8 (= 44.1-1), libclutter-1.0-0 (>= 1.16), libclutter-gtk-1.0-0 (>= 0.91.8), libgdk-pixbuf-2.0-0 (>= 2.25.2), libglib2.0-0 (>= 2.40.0), libgnome-desktop-3-20 (>= 3.17.92), libgstreamer1.0-0 (>= 1.0.0), libgtk-3-0 (>= 3.21.5), cheese-common (>= 44.1-1), gnome-video-effects",
            "conffiles": [],
            "pre_depends": "",
            "description": "tool to take pictures and videos from your webcam A webcam application that supports image and video capture. Makes it easy to take photos and videos of you, your friends, pets or whatever you want. Allows you to apply fancy visual effects, fine-control image settings and has features such as Multi-Burst mode, Countdown timer for photos.",
            "source": "",
            "homepage": "https://wiki.gnome.org/Apps/Cheese"
        },
        {
            "package": "cheese-common",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "all",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "44.1-1",
            "section": "gnome",
            "installed_size": 904,
            "depends": "dconf-gsettings-backend | gsettings-backend",
            "conffiles": [],
            "pre_depends": "",
            "description": "Common files for the Cheese tool to take pictures and videos A webcam application that supports image and video capture. Makes it easy to take photos and videos of you, your friends, pets or whatever you want. Allows you to apply fancy visual effects, fine-control image settings and has features such as Multi-Burst mode, Countdown timer for photos. . This package contains the common files and translations.",
            "source": "cheese",
            "homepage": "https://wiki.gnome.org/Apps/Cheese"
        },
        {
            "package": "chrome-gnome-shell",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "all",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "42.1-4",
            "section": "oldlibs",
            "installed_size": 8,
            "depends": "gnome-browser-connector",
            "conffiles": [],
            "pre_depends": "",
            "description": "GNOME Shell integration for browsers - transitional package This empty transitional package is here to ensure smooth upgrades.",
            "source": "gnome-browser-connector",
            "homepage": "https://wiki.gnome.org/Projects/GnomeShellIntegration"
        },
        {
            "package": "cifs-utils",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "2:7.0-2",
            "section": "otherosfs",
            "installed_size": 325,
            "depends": "libc6 (>= 2.34), libcap-ng0 (>= 0.7.9), libgssapi-krb5-2 (>= 1.17), libkeyutils1 (>= 1.4), libkrb5-3 (>= 1.13~alpha1+dfsg), libpam0g (>= 0.99.7.1), libtalloc2 (>= 2.0.4~git20101213), libwbclient0 (>= 2:4.0.3+dfsg1), python3",
            "conffiles": [
                "/etc/request-key.d/cifs.idmap.conf",
                "/etc/request-key.d/cifs.spnego.conf"
            ],
            "pre_depends": "",
            "description": "Common Internet File System utilities The SMB/CIFS protocol provides support for cross-platform file sharing with Microsoft Windows, OS X, and other Unix systems. . This package provides utilities for managing mounts of CIFS network file systems.",
            "source": "",
            "homepage": "https://www.samba.org/~jlayton/cifs-utils/"
        },
        {
            "package": "cmake",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "3.27.4-1",
            "section": "devel",
            "installed_size": 36311,
            "depends": "libarchive13 (>= 3.3.3), libc6 (>= 2.38), libcurl4 (>= 7.16.2), libexpat1 (>= 2.0.1), libgcc-s1 (>= 3.0), libjsoncpp25 (>= 1.9.5), librhash0 (>= 1.2.6), libstdc++6 (>= 13.1), libuv1 (>= 1.38.0), zlib1g (>= 1:1.1.4), cmake-data (= 3.27.4-1), procps",
            "conffiles": [],
            "pre_depends": "",
            "description": "cross-platform, open-source make system CMake is used to control the software compilation process using simple platform and compiler independent configuration files. CMake generates native makefiles and workspaces that can be used in the compiler environment of your choice. CMake is quite sophisticated: it is possible to support complex environments requiring system configuration, pre-processor generation, code generation, and template instantiation.",
            "source": "",
            "homepage": "https://cmake.org/"
        },
        {
            "package": "cmake-data",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "all",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "3.27.4-1",
            "section": "devel",
            "installed_size": 10830,
            "depends": "",
            "conffiles": [],
            "pre_depends": "",
            "description": "CMake data files (modules, templates and documentation) CMake is used to control the software compilation process using simple platform and compiler independent configuration files. CMake generates native makefiles and workspaces that can be used in the compiler environment of your choice. CMake is quite sophisticated: it is possible to support complex environments requiring system configuration, pre-processor generation, code generation, and template instantiation. . This package provides CMake architecture independent data files (modules, templates, documentation etc.). Unless you have cmake installed, you probably do not need this package.",
            "source": "cmake",
            "homepage": "https://cmake.org/"
        },
        {
            "package": "cmatrix",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "2.0-4",
            "section": "misc",
            "installed_size": 50,
            "depends": "libc6 (>= 2.34), libncurses6 (>= 6), libtinfo6 (>= 6)",
            "conffiles": [],
            "pre_depends": "",
            "description": "simulates the display from \"The Matrix\" Screen saver for the terminal based in the movie \"The Matrix\". It works in terminals of all dimensions and have the following features:  * Support terminal resize.  * Screen saver mode: any key closes it.  * Selectable color.  * Change text scroll rate.",
            "source": "",
            "homepage": "https://github.com/abishekvashok/cmatrix"
        },
        {
            "package": "code",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Microsoft Corporation <vscode-linux@microsoft.com>",
            "version": "1.85.1-1702462158",
            "section": "devel",
            "installed_size": 373465,
            "depends": "ca-certificates, libasound2 (>= 1.0.17), libatk-bridge2.0-0 (>= 2.5.3), libatk1.0-0 (>= 2.2.0), libatspi2.0-0 (>= 2.9.90), libc6 (>= 2.14), libc6 (>= 2.16), libc6 (>= 2.17), libc6 (>= 2.2.5), libcairo2 (>= 1.6.0), libcurl3-gnutls | libcurl3-nss | libcurl4 | libcurl3, libdbus-1-3 (>= 1.5.12), libdrm2 (>= 2.4.75), libexpat1 (>= 2.0.1), libgbm1 (>= 17.1.0~rc2), libglib2.0-0 (>= 2.37.3), libgssapi-krb5-2, libgtk-3-0 (>= 3.9.10), libgtk-3-0 (>= 3.9.10) | libgtk-4-1, libkrb5-3, libnspr4 (>= 2:4.9-2~), libnss3 (>= 2:3.30), libnss3 (>= 3.26), libpango-1.0-0 (>= 1.14.0), libx11-6, libx11-6 (>= 2:1.4.99.1), libxcb1 (>= 1.9.2), libxcomposite1 (>= 1:0.4.4-1), libxdamage1 (>= 1:1.1), libxext6, libxfixes3, libxkbcommon0 (>= 0.4.1), libxkbfile1, libxrandr2, xdg-utils (>= 1.0.2)",
            "conffiles": [],
            "pre_depends": "",
            "description": "Code editing. Redefined. Visual Studio Code is a new choice of tool that combines the simplicity of a code editor with what developers need for the core edit-build-debug cycle. See https://code.visualstudio.com/docs/setup/linux for installation instructions and FAQ.",
            "source": "",
            "homepage": "https://code.visualstudio.com/"
        },
        {
            "package": "colord",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "foreign",
            "maintainer": "Christopher James Halse Rogers <raof@ubuntu.com>",
            "version": "1.4.6-3",
            "section": "graphics",
            "installed_size": 974,
            "depends": "acl, adduser, colord-data, polkitd | policykit-1 (>= 0.103), dconf-gsettings-backend | gsettings-backend, libc6 (>= 2.34), libcolord2 (>= 1.4.3), libcolorhug2 (>= 0.1.30), libdbus-1-3 (>= 1.9.14), libglib2.0-0 (>= 2.75.3), libgudev-1.0-0 (>= 146), libgusb2 (>= 0.2.7), liblcms2-2 (>= 2.2+git20110628), libpolkit-gobject-1-0 (>= 0.103), libsane1 (>= 1.0.27), libsqlite3-0 (>= 3.5.9), libsystemd0",
            "conffiles": [],
            "pre_depends": "",
            "description": "system service to manage device colour profiles -- system daemon colord is a system service that makes it easy to manage, install and generate colour profiles to accurately colour manage input and output devices. . It provides a D-Bus API for system frameworks to query, a persistent data store, and a mechanism for session applications to set system policy. . This package contains the dbus-activated colord system daemon.",
            "source": "",
            "homepage": "https://www.freedesktop.org/software/colord/"
        },
        {
            "package": "colord-data",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "all",
            "multi_arch": "",
            "maintainer": "Christopher James Halse Rogers <raof@ubuntu.com>",
            "version": "1.4.6-3",
            "section": "graphics",
            "installed_size": 992,
            "depends": "",
            "conffiles": [],
            "pre_depends": "",
            "description": "system service to manage device colour profiles -- data files colord is a system service that makes it easy to manage, install and generate colour profiles to accurately colour manage input and output devices. . It provides a D-Bus API for system frameworks to query, a persistent data store, and a mechanism for session applications to set system policy. . This package contains data for the colord system daemon.",
            "source": "colord",
            "homepage": "https://www.freedesktop.org/software/colord/"
        },
        {
            "package": "command-not-found",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "all",
            "multi_arch": "",
            "maintainer": "Michael Vogt <michael.vogt@ubuntu.com>",
            "version": "23.04.0",
            "section": "admin",
            "installed_size": 29,
            "depends": "python3-commandnotfound (= 23.04.0)",
            "conffiles": [
                "/etc/apt/apt.conf.d/50command-not-found",
                "/etc/zsh_command_not_found"
            ],
            "pre_depends": "",
            "description": "Suggest installation of packages in interactive bash sessions This package will install a handler for command_not_found that looks up programs not currently installed but available from the repositories.",
            "source": "",
            "homepage": ""
        },
        {
            "package": "compiz-core",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "1:0.9.14.2+22.10.20220822-0ubuntu4",
            "section": "x11",
            "installed_size": 5834,
            "depends": "libc6 (>= 2.38), libgcc-s1 (>= 3.0), libglib2.0-0 (>= 2.30.0), libglibmm-2.4-1v5 (>= 2.66.6), libice6 (>= 1:1.0.0), libsigc++-2.0-0v5 (>= 2.2.0), libsm6, libstartup-notification0 (>= 0.7), libstdc++6 (>= 11), libx11-6, libxcursor1 (>> 1.1.2), libxext6, libxi6 (>= 2:1.2.99.4), libxinerama1 (>= 2:1.1.4), libxrandr2",
            "conffiles": [],
            "pre_depends": "",
            "description": "OpenGL window and compositing manager Compiz brings to life a variety of visual effects that make the Linux desktop easier to use, more powerful and intuitive, and more accessible for users with special needs. . Compiz combines together a window manager and a composite manager using OpenGL for rendering. A \"window manager\" allows the manipulation of the multiple applications and dialog windows that are presented on the screen. A \"composite manager\" allows windows and other graphics to be combined together to create composite images. Compiz achieves its stunning effects by doing both of these functions.",
            "source": "compiz",
            "homepage": "https://launchpad.net/compiz"
        },
        {
            "package": "compiz-plugins-default",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "same",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "1:0.9.14.2+22.10.20220822-0ubuntu4",
            "section": "x11",
            "installed_size": 6005,
            "depends": "compiz-core (= 1:0.9.14.2+22.10.20220822-0ubuntu4), libdecoration0 (= 1:0.9.14.2+22.10.20220822-0ubuntu4), libc6 (>= 2.38), libcairo2 (>= 1.2.4), libgcc-s1 (>= 3.0), libglx0, libopengl0, libpng16-16 (>= 1.6.2-1), libstdc++6 (>= 13.1), libx11-6, libxcomposite1 (>= 1:0.4.5), libxdamage1 (>= 1:1.1), libxext6 (>= 2:1.3.0), libxfixes3 (>= 1:4.0.1), libxml2 (>= 2.7.4), libxrandr2 (>= 4.3), libxrender1",
            "conffiles": [],
            "pre_depends": "",
            "description": "OpenGL window and compositing manager - default plugins Compiz brings to life a variety of visual effects that make the Linux desktop easier to use, more powerful and intuitive, and more accessible for users with special needs. . This package contains the default set of core Compiz plugins.",
            "source": "compiz",
            "homepage": "https://launchpad.net/compiz"
        },
        {
            "package": "compizconfig-settings-manager",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "all",
            "multi_arch": "allowed",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "1:0.9.14.2+22.10.20220822-0ubuntu4",
            "section": "x11",
            "installed_size": 4377,
            "depends": "python3:any, gir1.2-gdkpixbuf-2.0, gir1.2-gtk-3.0 (>= 3.22), gir1.2-pango-1.0, python3-compizconfig (>= 1:0.9.14.2+22.10.20220822-0ubuntu4), python3-gi, python3-gi-cairo",
            "conffiles": [],
            "pre_depends": "",
            "description": "Compiz configuration settings manager The OpenCompositing Project brings 3D desktop visual effects that improve usability of the X Window System and provide increased productivity. . This package contains the compizconfig settings manager.",
            "source": "compiz",
            "homepage": "https://launchpad.net/compiz"
        },
        {
            "package": "conmon",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Lokesh Mandvekar <lsm5@fedoraproject.org>",
            "version": "100:2.1.2~0",
            "section": "devel",
            "installed_size": 450,
            "depends": "libglib2.0-0",
            "conffiles": [],
            "pre_depends": "",
            "description": "OCI container runtime monitor",
            "source": "",
            "homepage": "https://github.com/containers/conmon.git"
        },
        {
            "package": "conntrack",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "1:1.4.7-1",
            "section": "net",
            "installed_size": 115,
            "depends": "libc6 (>= 2.34), libmnl0 (>= 1.0.3-4~), libnetfilter-conntrack3 (>= 1.0.9)",
            "conffiles": [],
            "pre_depends": "",
            "description": "Program to modify the conntrack tables conntrack is a userspace command line program targeted at system administrators. It enables them to view and manage the in-kernel connection tracking state table.",
            "source": "conntrack-tools",
            "homepage": "https://conntrack-tools.netfilter.org/"
        },
        {
            "package": "console-setup",
            "status": "install ok installed",
            "priority": "important",
            "architecture": "all",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "1.222ubuntu1",
            "section": "utils",
            "installed_size": 431,
            "depends": "console-setup-linux | console-setup-freebsd | hurd, xkb-data (>= 0.9), keyboard-configuration (= 1.222ubuntu1), debconf (>= 0.5) | debconf-2.0",
            "conffiles": [],
            "pre_depends": "debconf | debconf-2.0",
            "description": "console font and keymap setup program This package provides the console with the same keyboard configuration scheme as the X Window System. As a result, there is no need to duplicate or change the keyboard files just to make simple customizations such as the use of dead keys, the key functioning as AltGr or Compose key, the key(s) to switch between Latin and non-Latin mode, etc. . The package also installs console fonts supporting many of the world's languages.  It provides an unified set of font faces - the classic VGA, the simplistic Fixed, and the cleaned Terminus, TerminusBold and TerminusBoldVGA.",
            "source": "",
            "homepage": ""
        },
        {
            "package": "console-setup-linux",
            "status": "install ok installed",
            "priority": "important",
            "architecture": "all",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "1.222ubuntu1",
            "section": "utils",
            "installed_size": 2201,
            "depends": "kbd (>= 0.99-12) | console-tools (>= 1:0.2.3-16), keyboard-configuration (= 1.222ubuntu1), init-system-helpers (>= 1.29~) | initscripts",
            "conffiles": [
                "/etc/console-setup/compose.ARMSCII-8.inc",
                "/etc/console-setup/compose.CP1251.inc",
                "/etc/console-setup/compose.CP1255.inc",
                "/etc/console-setup/compose.CP1256.inc",
                "/etc/console-setup/compose.GEORGIAN-ACADEMY.inc",
                "/etc/console-setup/compose.GEORGIAN-PS.inc",
                "/etc/console-setup/compose.IBM1133.inc",
                "/etc/console-setup/compose.ISIRI-3342.inc",
                "/etc/console-setup/compose.ISO-8859-1.inc",
                "/etc/console-setup/compose.ISO-8859-10.inc",
                "/etc/console-setup/compose.ISO-8859-11.inc",
                "/etc/console-setup/compose.ISO-8859-13.inc",
                "/etc/console-setup/compose.ISO-8859-14.inc",
                "/etc/console-setup/compose.ISO-8859-15.inc",
                "/etc/console-setup/compose.ISO-8859-16.inc",
                "/etc/console-setup/compose.ISO-8859-2.inc",
                "/etc/console-setup/compose.ISO-8859-3.inc",
                "/etc/console-setup/compose.ISO-8859-4.inc",
                "/etc/console-setup/compose.ISO-8859-5.inc",
                "/etc/console-setup/compose.ISO-8859-6.inc",
                "/etc/console-setup/compose.ISO-8859-7.inc",
                "/etc/console-setup/compose.ISO-8859-8.inc",
                "/etc/console-setup/compose.ISO-8859-9.inc",
                "/etc/console-setup/compose.KOI8-R.inc",
                "/etc/console-setup/compose.KOI8-U.inc",
                "/etc/console-setup/compose.TIS-620.inc",
                "/etc/console-setup/compose.VISCII.inc",
                "/etc/console-setup/remap.inc",
                "/etc/console-setup/vtrgb",
                "/etc/console-setup/vtrgb.vga",
                "/etc/init.d/console-setup.sh",
                "/etc/init.d/keyboard-setup.sh"
            ],
            "pre_depends": "",
            "description": "Linux specific part of console-setup This package includes fonts in psf format and definitions of various 8-bit charmaps.",
            "source": "console-setup",
            "homepage": ""
        },
        {
            "package": "containerd.io",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Containerd team <help@containerd.io>",
            "version": "1.6.26-1",
            "section": "devel",
            "installed_size": 115616,
            "depends": "libc6 (>= 2.34), libseccomp2 (>= 2.5.0)",
            "conffiles": [
                "/etc/containerd/config.toml"
            ],
            "pre_depends": "",
            "description": "An open and reliable container runtime",
            "source": "",
            "homepage": "https://containerd.io"
        },
        {
            "package": "containernetworking-plugins",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "1.1.1+ds1-3build1",
            "section": "golang",
            "installed_size": 46644,
            "depends": "iptables, libc6 (>= 2.34)",
            "conffiles": [],
            "pre_depends": "",
            "description": "standard networking plugins - binaries This package contains binaries of the Container Networking Initiative's official plugins: . Interfaces  - bridge: Creates a bridge, adds the host and the container to it.  - ipvlan: Adds an ipvlan interface in the container.  - loopback: Set the state of loopback interface to up.  - macvlan: Creates a new MAC address, forwards all traffic             to that to the container.  - ptp: Creates a veth pair.  - vlan: Allocates a vlan device.  - host-device: Move an already-existing device into a container. . IPAM: IP Address Management  - dhcp: Runs a daemon to make DHCP requests on behalf of the container.  - host-local: Maintains a local database of allocated IPs  - static: Allocates a static IPv4/IPv6 address. . Other  - tuning: Tweaks sysctl parameters of an existing interface  - portmap: An iptables-based portmapping plugin.             Maps ports from the host's address space to the container.  - bandwidth: Allows bandwidth-limiting through use of traffic control tbf.  - sbr: Configures source based routing for an interface.  - firewall: Uses iptables or firewalld to add rules to allow traffic              to/from the container.",
            "source": "golang-github-containernetworking-plugins",
            "homepage": "https://github.com/containernetworking/plugins"
        },
        {
            "package": "containers-common",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "all",
            "multi_arch": "",
            "maintainer": "Lokesh Mandvekar <lsm5@fedoraproject.org>",
            "version": "100:1-22",
            "section": "devel",
            "installed_size": 112,
            "depends": "",
            "conffiles": [
                "/etc/containers/policy.json",
                "/etc/containers/registries.conf",
                "/etc/containers/registries.conf.d/000-shortnames.conf",
                "/etc/containers/registries.d/default.yaml",
                "/etc/containers/storage.conf"
            ],
            "pre_depends": "",
            "description": "Configuration files for working with image signatures.",
            "source": "",
            "homepage": "https://github.com/projectatomic/skopeo"
        },
        {
            "package": "coreutils",
            "status": "install ok installed",
            "priority": "required",
            "architecture": "amd64",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "9.1-1ubuntu2",
            "section": "utils",
            "installed_size": 7000,
            "depends": "",
            "conffiles": [],
            "pre_depends": "libacl1 (>= 2.2.23), libattr1 (>= 1:2.4.44), libc6 (>= 2.34), libgmp10 (>= 2:6.2.1+dfsg1), libselinux1 (>= 3.1~)",
            "description": "GNU core utilities This package contains the basic file, shell and text manipulation utilities which are expected to exist on every operating system. . Specifically, this package includes: arch base64 basename cat chcon chgrp chmod chown chroot cksum comm cp csplit cut date dd df dir dircolors dirname du echo env expand expr factor false flock fmt fold groups head hostid id install join link ln logname ls md5sum mkdir mkfifo mknod mktemp mv nice nl nohup nproc numfmt od paste pathchk pinky pr printenv printf ptx pwd readlink realpath rm rmdir runcon sha*sum seq shred sleep sort split stat stty sum sync tac tail tee test timeout touch tr true truncate tsort tty uname unexpand uniq unlink users vdir wc who whoami yes",
            "source": "",
            "homepage": "http://gnu.org/software/coreutils"
        },
        {
            "package": "cpdb-backend-cups",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "2.0~b5-0ubuntu2",
            "section": "net",
            "installed_size": 100,
            "depends": "libc6 (>= 2.34), libcpdb2 (>= 2.0~b4-0ubuntu3), libcups2 (>= 2.2.2), libcupsfilters2 (>= 2.0~b4-0ubuntu1), libglib2.0-0 (>= 2.38.0)",
            "conffiles": [],
            "pre_depends": "",
            "description": "Common Print Dialog Backends - CUPS/IPP Backend This is the CUPS/IPP backend for print dialogs using the Common Print Dialog Backends concept of OpenPrinting. It makes the dialog list CUPS print queues and driverless-capable IPP printers and allows printing on these using the dialog.",
            "source": "",
            "homepage": "https://github.com/OpenPrinting/cpdb-backend-cups"
        },
        {
            "package": "cpio",
            "status": "install ok installed",
            "priority": "important",
            "architecture": "amd64",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "2.13+dfsg-7.1",
            "section": "utils",
            "installed_size": 324,
            "depends": "libc6 (>= 2.34)",
            "conffiles": [],
            "pre_depends": "",
            "description": "GNU cpio -- a program to manage archives of files GNU cpio is a tool for creating and extracting archives, or copying files from one place to another.  It handles a number of cpio formats as well as reading and writing tar files.",
            "source": "",
            "homepage": "https://www.gnu.org/software/cpio/"
        },
        {
            "package": "cpp",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "allowed",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "4:13.2.0-1ubuntu1",
            "section": "interpreters",
            "installed_size": 53,
            "depends": "cpp-13 (>= 13.2.0-2~)",
            "conffiles": [],
            "pre_depends": "",
            "description": "GNU C preprocessor (cpp) The GNU C preprocessor is a macro processor that is used automatically by the GNU C compiler to transform programs before actual compilation. . This package has been separated from gcc for the benefit of those who require the preprocessor but not the compiler. . This is a dependency package providing the default GNU C preprocessor.",
            "source": "gcc-defaults (1.208ubuntu1)",
            "homepage": ""
        },
        {
            "package": "cpp-10",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Core developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "10.5.0-1ubuntu1",
            "section": "interpreters",
            "installed_size": 26167,
            "depends": "gcc-10-base (= 10.5.0-1ubuntu1), libc6 (>= 2.34), libgmp10 (>= 2:6.2.1+dfsg1), libisl23 (>= 0.15), libmpc3 (>= 1.1.0), libmpfr6 (>= 3.1.3), libzstd1 (>= 1.5.5), zlib1g (>= 1:1.1.4)",
            "conffiles": [],
            "pre_depends": "",
            "description": "GNU C preprocessor A macro processor that is used automatically by the GNU C compiler to transform programs before actual compilation. . This package has been separated from gcc for the benefit of those who require the preprocessor but not the compiler.",
            "source": "gcc-10",
            "homepage": "http://gcc.gnu.org/"
        },
        {
            "package": "cpp-11",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Core developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "11.4.0-4ubuntu1",
            "section": "interpreters",
            "installed_size": 26296,
            "depends": "gcc-11-base (= 11.4.0-4ubuntu1), libc6 (>= 2.38), libgmp10 (>= 2:6.3.0+dfsg), libisl23 (>= 0.15), libmpc3 (>= 1.1.0), libmpfr6 (>= 3.1.3), libzstd1 (>= 1.5.5), zlib1g (>= 1:1.1.4)",
            "conffiles": [],
            "pre_depends": "",
            "description": "GNU C preprocessor A macro processor that is used automatically by the GNU C compiler to transform programs before actual compilation. . This package has been separated from gcc for the benefit of those who require the preprocessor but not the compiler.",
            "source": "gcc-11",
            "homepage": "http://gcc.gnu.org/"
        },
        {
            "package": "cpp-12",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Core developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "12.3.0-9ubuntu2",
            "section": "interpreters",
            "installed_size": 34582,
            "depends": "gcc-12-base (= 12.3.0-9ubuntu2), libc6 (>= 2.38), libgmp10 (>= 2:6.3.0+dfsg), libisl23 (>= 0.15), libmpc3 (>= 1.1.0), libmpfr6 (>= 3.1.3), libzstd1 (>= 1.5.5), zlib1g (>= 1:1.1.4)",
            "conffiles": [],
            "pre_depends": "",
            "description": "GNU C preprocessor A macro processor that is used automatically by the GNU C compiler to transform programs before actual compilation. . This package has been separated from gcc for the benefit of those who require the preprocessor but not the compiler.",
            "source": "gcc-12",
            "homepage": "http://gcc.gnu.org/"
        },
        {
            "package": "cpp-13",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Core developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "13.2.0-4ubuntu3",
            "section": "interpreters",
            "installed_size": 30725,
            "depends": "gcc-13-base (= 13.2.0-4ubuntu3), libc6 (>= 2.38), libgmp10 (>= 2:6.3.0+dfsg), libisl23 (>= 0.15), libmpc3 (>= 1.1.0), libmpfr6 (>= 3.1.3), libzstd1 (>= 1.5.5), zlib1g (>= 1:1.1.4)",
            "conffiles": [],
            "pre_depends": "",
            "description": "GNU C preprocessor A macro processor that is used automatically by the GNU C compiler to transform programs before actual compilation. . This package has been separated from gcc for the benefit of those who require the preprocessor but not the compiler.",
            "source": "gcc-13",
            "homepage": "http://gcc.gnu.org/"
        },
        {
            "package": "cpp-9",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Core developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "9.5.0-4ubuntu2",
            "section": "interpreters",
            "installed_size": 28029,
            "depends": "gcc-9-base (= 9.5.0-4ubuntu2), libc6 (>= 2.38), libgmp10 (>= 2:6.3.0+dfsg), libisl23 (>= 0.15), libmpc3 (>= 1.1.0), libmpfr6 (>= 3.1.3), zlib1g (>= 1:1.1.4)",
            "conffiles": [],
            "pre_depends": "",
            "description": "GNU C preprocessor A macro processor that is used automatically by the GNU C compiler to transform programs before actual compilation. . This package has been separated from gcc for the benefit of those who require the preprocessor but not the compiler.",
            "source": "gcc-9",
            "homepage": "http://gcc.gnu.org/"
        },
        {
            "package": "cpu-checker",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Kees Cook <kees@ubuntu.com>",
            "version": "0.7-1.3build1",
            "section": "utils",
            "installed_size": 21,
            "depends": "msr-tools",
            "conffiles": [],
            "pre_depends": "",
            "description": "tools to help evaluate certain CPU (or BIOS) features There are some CPU features that are filtered or disabled by system BIOSes. This set of tools seeks to help identify when certain features are in this state, based on kernel values, CPU flags and other conditions. Supported feature tests are NX/XD and VMX/SVM.",
            "source": "",
            "homepage": "https://launchpad.net/cpu-checker"
        },
        {
            "package": "cracklib-runtime",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "2.9.6-5build1",
            "section": "admin",
            "installed_size": 584,
            "depends": "file, libcrack2 (>= 2.9.6-5build1), libc6 (>= 2.34)",
            "conffiles": [
                "/etc/cracklib/cracklib.conf",
                "/etc/logcheck/ignore.d.paranoid/cracklib-runtime"
            ],
            "pre_depends": "",
            "description": "runtime support for password checker library cracklib2 Run-time support programs which use the shared library in libcrack2 including programs to build the password dictionary databases used by the functions in the shared library.",
            "source": "cracklib2",
            "homepage": "https://github.com/cracklib/cracklib"
        },
        {
            "package": "cramfsswap",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "1.4.2",
            "section": "utils",
            "installed_size": 29,
            "depends": "libc6 (>= 2.4), zlib1g (>= 1:1.1.4)",
            "conffiles": [],
            "pre_depends": "",
            "description": "swap endianness of a cram filesystem (cramfs) cramfs is a highly compressed and size optimized Linux filesystem which is mainly used for embedded applications. The problem with cramfs is that it is endianness sensitive, meaning you can't mount a cramfs for a big endian target on a little endian machine and vice versa. This is often especially a problem in the development phase. . cramfsswap solves that problem by allowing you to swap to endianness of a cramfs filesystem.",
            "source": "",
            "homepage": ""
        },
        {
            "package": "cri-o",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Peter Hunt <haircommander@fedoraproject.org>",
            "version": "1.24.6~0",
            "section": "devel",
            "installed_size": 93809,
            "depends": "libgpgme11, libseccomp2, conmon, containers-common (>= 0.1.27) | golang-github-containers-common, tzdata",
            "conffiles": [
                "/etc/cni/net.d/100-crio-bridge.conf",
                "/etc/cni/net.d/200-loopback.conf",
                "/etc/crictl.yaml",
                "/etc/crio/crio.conf",
                "/etc/crio/crio.conf.d/01-crio-runc.conf",
                "/etc/default/crio"
            ],
            "pre_depends": "",
            "description": "OCI-based implementation of Kubernetes Container Runtime Interface.",
            "source": "",
            "homepage": "https://github.com/cri-o/cri-o"
        },
        {
            "package": "cri-o-runc",
            "status": "install ok installed",
            "priority": "extra",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Lokesh Mandvekar <lsm5@fedoraproject.org>",
            "version": "1.1.9~1",
            "section": "devel",
            "installed_size": 9782,
            "depends": "libc6 (>= 2.34), libseccomp2 (>= 2.5.0)",
            "conffiles": [],
            "pre_depends": "",
            "description": "Open Container Project - runtime \"runc\" is a command line client for running applications packaged according to the Open Container Format (OCF) and is a compliant implementation of the Open Container Project specification. This package is a fork of the \"runc' package, specifically for cri-o.",
            "source": "",
            "homepage": "https://github.com/opencontainers/runc"
        },
        {
            "package": "cri-tools",
            "status": "install ok installed",
            "priority": "optional",
            "architecture": "amd64",
            "multi_arch": "",
            "maintainer": "Kubernetes Authors <kubernetes-dev@googlegroups.com>",
            "version": "1.26.0-00",
            "section": "misc",
            "installed_size": 51358,
            "depends": "",
            "conffiles": [],
            "pre_depends": "",
            "description": "Container Runtime Interface Tools Binaries that interact with the container runtime through the container runtime interface",
            "source": "",
            "homepage": "https://kubernetes.io"
        },
        {
            "package": "cron",
            "status": "install ok installed",
            "priority": "standard",
            "architecture": "amd64",
            "multi_arch": "foreign",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "3.0pl1-163ubuntu1",
            "section": "admin",
            "installed_size": 210,
            "depends": "libc6 (>= 2.34), libpam0g (>= 0.99.7.1), libselinux1 (>= 3.1~), sensible-utils, libpam-runtime",
            "conffiles": [
                "/etc/default/cron",
                "/etc/init.d/cron",
                "/etc/pam.d/cron"
            ],
            "pre_depends": "init-system-helpers (>= 1.54~), cron-daemon-common",
            "description": "process scheduling daemon The cron daemon is a background process that runs particular programs at particular times (for example, every minute, day, week, or month), as specified in a crontab. By default, users may also create crontabs of their own so that processes are run on their behalf. . Output from the commands is usually mailed to the system administrator (or to the user in question); you should probably install a mail system as well so that you can receive these messages. . This cron package does not provide any system maintenance tasks. Basic periodic maintenance tasks are provided by other packages, such as checksecurity.",
            "source": "",
            "homepage": "https://ftp.isc.org/isc/cron/"
        },
        {
            "package": "cron-daemon-common",
            "status": "install ok installed",
            "priority": "important",
            "architecture": "all",
            "multi_arch": "",
            "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
            "version": "3.0pl1-163ubuntu1",
            "section": "admin",
            "installed_size": 41,
            "depends": "adduser",
            "conffiles": [
                "/etc/cron.d/.placeholder",
                "/etc/cron.daily/.placeholder",
                "/etc/cron.hourly/.placeholder",
                "/etc/cron.monthly/.placeholder",
                "/etc/cron.weekly/.placeholder",
                "/etc/cron.yearly/.placeholder",
                "/etc/crontab"
            ],
            "pre_depends": "",
            "description": "process scheduling daemon's configuration files The cron daemon is a background process that runs particular programs at particular times (for example, every minute, day, week, or month), as specified in a crontab. By default, users may also create crontabs of their own so that processes are run on their behalf. . This package provides configuration files which must be there to define scheduled process sets.",
            "source": "cron",
            "homepage": "https://ftp.isc.org/isc/cron/"
        },
        ...(omitted)...
    ],
    "rpm": [],
    "docker": {
        "Containers": [
            {
                "Id": "6f54e8d8dfa70d3311073822a4f718a5215a770efa7db1ba9952d179ea0549cc",
                "Names": [
                    "/airflow-airflow-triggerer-1"
                ],
                "Image": "apache/airflow:2.7.2",
                "ImageID": "sha256:cf153fd692defe76b8abab371c8b64c21d7a838500a9ae06250d5984b2d51324",
                "Command": "/usr/bin/dumb-init -- /entrypoint triggerer",
                "Created": 1698736049,
                "Ports": [
                    {
                        "PrivatePort": 8080,
                        "Type": "tcp"
                    }
                ],
                "Labels": {
                    "com.docker.compose.config-hash": "b8eb36ac939e1e15021923787d6809aa2bbf19402cc92051158babdd6d864b95",
                    "com.docker.compose.container-number": "1",
                    "com.docker.compose.depends_on": "postgres:service_healthy:false,airflow-init:service_completed_successfully:false,redis:service_healthy:false",
                    "com.docker.compose.image": "sha256:cf153fd692defe76b8abab371c8b64c21d7a838500a9ae06250d5984b2d51324",
                    "com.docker.compose.oneoff": "False",
                    "com.docker.compose.project": "airflow",
                    "com.docker.compose.project.config_files": "/home/ish/test/airflow/docker-compose.yaml",
                    "com.docker.compose.project.working_dir": "/home/ish/test/airflow",
                    "com.docker.compose.service": "airflow-triggerer",
                    "com.docker.compose.version": "2.21.0",
                    "org.apache.airflow.component": "airflow",
                    "org.apache.airflow.distro": "debian",
                    "org.apache.airflow.image": "airflow",
                    "org.apache.airflow.main-image.build-id": "",
                    "org.apache.airflow.main-image.commit-sha": "c8b25cb3eea2bcdf951ed7c1d7d0a1f9f04db206",
                    "org.apache.airflow.module": "airflow",
                    "org.apache.airflow.uid": "50000",
                    "org.apache.airflow.version": "2.7.2",
                    "org.opencontainers.image.authors": "dev@airflow.apache.org",
                    "org.opencontainers.image.created": "",
                    "org.opencontainers.image.description": "Reference, production-ready Apache Airflow image",
                    "org.opencontainers.image.documentation": "https://airflow.apache.org/docs/docker-stack/index.html",
                    "org.opencontainers.image.licenses": "Apache-2.0",
                    "org.opencontainers.image.ref.name": "airflow",
                    "org.opencontainers.image.revision": "c8b25cb3eea2bcdf951ed7c1d7d0a1f9f04db206",
                    "org.opencontainers.image.source": "https://github.com/apache/airflow",
                    "org.opencontainers.image.title": "Production Airflow Image",
                    "org.opencontainers.image.url": "https://airflow.apache.org",
                    "org.opencontainers.image.vendor": "Apache Software Foundation",
                    "org.opencontainers.image.version": "2.7.2"
                },
                "State": "running",
                "Status": "Up 21 hours (healthy)",
                "HostConfig": {
                    "NetworkMode": "airflow_default"
                },
                "NetworkSettings": {
                    "Networks": {
                        "airflow_default": {
                            "IPAMConfig": null,
                            "Links": null,
                            "Aliases": null,
                            "NetworkID": "9cc4698f0d3c6a5e474b0fbefc28ba3c1aa15497e03d25479029b4908aa94954",
                            "EndpointID": "75336674d6aed783eced80f3502c3d678dfea72bd3955cb600ee6d9d67d3f7ec",
                            "Gateway": "192.168.64.1",
                            "IPAddress": "192.168.64.7",
                            "IPPrefixLen": 20,
                            "IPv6Gateway": "",
                            "GlobalIPv6Address": "",
                            "GlobalIPv6PrefixLen": 0,
                            "MacAddress": "02:42:c0:a8:40:07",
                            "DriverOpts": null
                        }
                    }
                },
                "Mounts": [
                    {
                        "Type": "bind",
                        "Source": "/var/run/docker.sock",
                        "Destination": "/var/run/docker.sock",
                        "Mode": "rw",
                        "RW": true,
                        "Propagation": "rprivate"
                    },
                    {
                        "Type": "bind",
                        "Source": "/data",
                        "Destination": "/data",
                        "Mode": "rw",
                        "RW": true,
                        "Propagation": "rprivate"
                    },
                    {
                        "Type": "bind",
                        "Source": "/home/ish/test/airflow/config",
                        "Destination": "/opt/airflow/config",
                        "Mode": "rw",
                        "RW": true,
                        "Propagation": "rprivate"
                    },
                    {
                        "Type": "bind",
                        "Source": "/home/ish/test/airflow/dags",
                        "Destination": "/opt/airflow/dags",
                        "Mode": "rw",
                        "RW": true,
                        "Propagation": "rprivate"
                    },
                    {
                        "Type": "bind",
                        "Source": "/home/ish/test/airflow/logs",
                        "Destination": "/opt/airflow/logs",
                        "Mode": "rw",
                        "RW": true,
                        "Propagation": "rprivate"
                    },
                    {
                        "Type": "bind",
                        "Source": "/home/ish/test/airflow/plugins",
                        "Destination": "/opt/airflow/plugins",
                        "Mode": "rw",
                        "RW": true,
                        "Propagation": "rprivate"
                    }
                ]
            },
            {
                "Id": "1695bcf92e27beecb18c442c5c66054a4ee7bfdf739e1254a7d8b4851cc366cd",
                "Names": [
                    "/airflow-airflow-worker-1"
                ],
                "Image": "apache/airflow:2.7.2",
                "ImageID": "sha256:cf153fd692defe76b8abab371c8b64c21d7a838500a9ae06250d5984b2d51324",
                "Command": "/usr/bin/dumb-init -- /entrypoint celery worker",
                "Created": 1698736049,
                "Ports": [
                    {
                        "PrivatePort": 8080,
                        "Type": "tcp"
                    }
                ],
                "Labels": {
                    "com.docker.compose.config-hash": "3f7e10fee7cbbaa03ffe768c175956ffbbbe8c5bb43d9003be8a2b7699d44c9b",
                    "com.docker.compose.container-number": "1",
                    "com.docker.compose.depends_on": "postgres:service_healthy:false,airflow-init:service_completed_successfully:false,redis:service_healthy:false",
                    "com.docker.compose.image": "sha256:cf153fd692defe76b8abab371c8b64c21d7a838500a9ae06250d5984b2d51324",
                    "com.docker.compose.oneoff": "False",
                    "com.docker.compose.project": "airflow",
                    "com.docker.compose.project.config_files": "/home/ish/test/airflow/docker-compose.yaml",
                    "com.docker.compose.project.working_dir": "/home/ish/test/airflow",
                    "com.docker.compose.service": "airflow-worker",
                    "com.docker.compose.version": "2.21.0",
                    "org.apache.airflow.component": "airflow",
                    "org.apache.airflow.distro": "debian",
                    "org.apache.airflow.image": "airflow",
                    "org.apache.airflow.main-image.build-id": "",
                    "org.apache.airflow.main-image.commit-sha": "c8b25cb3eea2bcdf951ed7c1d7d0a1f9f04db206",
                    "org.apache.airflow.module": "airflow",
                    "org.apache.airflow.uid": "50000",
                    "org.apache.airflow.version": "2.7.2",
                    "org.opencontainers.image.authors": "dev@airflow.apache.org",
                    "org.opencontainers.image.created": "",
                    "org.opencontainers.image.description": "Reference, production-ready Apache Airflow image",
                    "org.opencontainers.image.documentation": "https://airflow.apache.org/docs/docker-stack/index.html",
                    "org.opencontainers.image.licenses": "Apache-2.0",
                    "org.opencontainers.image.ref.name": "airflow",
                    "org.opencontainers.image.revision": "c8b25cb3eea2bcdf951ed7c1d7d0a1f9f04db206",
                    "org.opencontainers.image.source": "https://github.com/apache/airflow",
                    "org.opencontainers.image.title": "Production Airflow Image",
                    "org.opencontainers.image.url": "https://airflow.apache.org",
                    "org.opencontainers.image.vendor": "Apache Software Foundation",
                    "org.opencontainers.image.version": "2.7.2"
                },
                "State": "running",
                "Status": "Up 21 hours (healthy)",
                "HostConfig": {
                    "NetworkMode": "airflow_default"
                },
                "NetworkSettings": {
                    "Networks": {
                        "airflow_default": {
                            "IPAMConfig": null,
                            "Links": null,
                            "Aliases": null,
                            "NetworkID": "9cc4698f0d3c6a5e474b0fbefc28ba3c1aa15497e03d25479029b4908aa94954",
                            "EndpointID": "7fcf708045d8aeb69a2480a58e0552088352cbbc0ccde5987d7d5faf3713f14e",
                            "Gateway": "192.168.64.1",
                            "IPAddress": "192.168.64.4",
                            "IPPrefixLen": 20,
                            "IPv6Gateway": "",
                            "GlobalIPv6Address": "",
                            "GlobalIPv6PrefixLen": 0,
                            "MacAddress": "02:42:c0:a8:40:04",
                            "DriverOpts": null
                        }
                    }
                },
                "Mounts": [
                    {
                        "Type": "bind",
                        "Source": "/home/ish/test/airflow/config",
                        "Destination": "/opt/airflow/config",
                        "Mode": "rw",
                        "RW": true,
                        "Propagation": "rprivate"
                    },
                    {
                        "Type": "bind",
                        "Source": "/home/ish/test/airflow/dags",
                        "Destination": "/opt/airflow/dags",
                        "Mode": "rw",
                        "RW": true,
                        "Propagation": "rprivate"
                    },
                    {
                        "Type": "bind",
                        "Source": "/home/ish/test/airflow/logs",
                        "Destination": "/opt/airflow/logs",
                        "Mode": "rw",
                        "RW": true,
                        "Propagation": "rprivate"
                    },
                    {
                        "Type": "bind",
                        "Source": "/home/ish/test/airflow/plugins",
                        "Destination": "/opt/airflow/plugins",
                        "Mode": "rw",
                        "RW": true,
                        "Propagation": "rprivate"
                    },
                    {
                        "Type": "bind",
                        "Source": "/var/run/docker.sock",
                        "Destination": "/var/run/docker.sock",
                        "Mode": "rw",
                        "RW": true,
                        "Propagation": "rprivate"
                    },
                    {
                        "Type": "bind",
                        "Source": "/data",
                        "Destination": "/data",
                        "Mode": "rw",
                        "RW": true,
                        "Propagation": "rprivate"
                    }
                ]
            },
            {
                "Id": "a1885bb3e64138ea93e7c3e481d9de903d8d81d761ef6a73c75f5fd8bba4a7c9",
                "Names": [
                    "/airflow-airflow-scheduler-1"
                ],
                "Image": "apache/airflow:2.7.2",
                "ImageID": "sha256:cf153fd692defe76b8abab371c8b64c21d7a838500a9ae06250d5984b2d51324",
                "Command": "/usr/bin/dumb-init -- /entrypoint scheduler",
                "Created": 1698736049,
                "Ports": [
                    {
                        "PrivatePort": 8080,
                        "Type": "tcp"
                    }
                ],
                "Labels": {
                    "com.docker.compose.config-hash": "b7f602f7f38f5ae7e0233a0c69101927517beb1b3405dc88e2b8151d63532137",
                    "com.docker.compose.container-number": "1",
                    "com.docker.compose.depends_on": "redis:service_healthy:false,postgres:service_healthy:false,airflow-init:service_completed_successfully:false",
                    "com.docker.compose.image": "sha256:cf153fd692defe76b8abab371c8b64c21d7a838500a9ae06250d5984b2d51324",
                    "com.docker.compose.oneoff": "False",
                    "com.docker.compose.project": "airflow",
                    "com.docker.compose.project.config_files": "/home/ish/test/airflow/docker-compose.yaml",
                    "com.docker.compose.project.working_dir": "/home/ish/test/airflow",
                    "com.docker.compose.service": "airflow-scheduler",
                    "com.docker.compose.version": "2.21.0",
                    "org.apache.airflow.component": "airflow",
                    "org.apache.airflow.distro": "debian",
                    "org.apache.airflow.image": "airflow",
                    "org.apache.airflow.main-image.build-id": "",
                    "org.apache.airflow.main-image.commit-sha": "c8b25cb3eea2bcdf951ed7c1d7d0a1f9f04db206",
                    "org.apache.airflow.module": "airflow",
                    "org.apache.airflow.uid": "50000",
                    "org.apache.airflow.version": "2.7.2",
                    "org.opencontainers.image.authors": "dev@airflow.apache.org",
                    "org.opencontainers.image.created": "",
                    "org.opencontainers.image.description": "Reference, production-ready Apache Airflow image",
                    "org.opencontainers.image.documentation": "https://airflow.apache.org/docs/docker-stack/index.html",
                    "org.opencontainers.image.licenses": "Apache-2.0",
                    "org.opencontainers.image.ref.name": "airflow",
                    "org.opencontainers.image.revision": "c8b25cb3eea2bcdf951ed7c1d7d0a1f9f04db206",
                    "org.opencontainers.image.source": "https://github.com/apache/airflow",
                    "org.opencontainers.image.title": "Production Airflow Image",
                    "org.opencontainers.image.url": "https://airflow.apache.org",
                    "org.opencontainers.image.vendor": "Apache Software Foundation",
                    "org.opencontainers.image.version": "2.7.2"
                },
                "State": "running",
                "Status": "Up 21 hours (healthy)",
                "HostConfig": {
                    "NetworkMode": "airflow_default"
                },
                "NetworkSettings": {
                    "Networks": {
                        "airflow_default": {
                            "IPAMConfig": null,
                            "Links": null,
                            "Aliases": null,
                            "NetworkID": "9cc4698f0d3c6a5e474b0fbefc28ba3c1aa15497e03d25479029b4908aa94954",
                            "EndpointID": "ac244cd8cd2d63d0be7d50c3ab34caac30d825d380ae5f2668f4a9a789aeda52",
                            "Gateway": "192.168.64.1",
                            "IPAddress": "192.168.64.3",
                            "IPPrefixLen": 20,
                            "IPv6Gateway": "",
                            "GlobalIPv6Address": "",
                            "GlobalIPv6PrefixLen": 0,
                            "MacAddress": "02:42:c0:a8:40:03",
                            "DriverOpts": null
                        }
                    }
                },
                "Mounts": [
                    {
                        "Type": "bind",
                        "Source": "/data",
                        "Destination": "/data",
                        "Mode": "rw",
                        "RW": true,
                        "Propagation": "rprivate"
                    },
                    {
                        "Type": "bind",
                        "Source": "/home/ish/test/airflow/config",
                        "Destination": "/opt/airflow/config",
                        "Mode": "rw",
                        "RW": true,
                        "Propagation": "rprivate"
                    },
                    {
                        "Type": "bind",
                        "Source": "/home/ish/test/airflow/dags",
                        "Destination": "/opt/airflow/dags",
                        "Mode": "rw",
                        "RW": true,
                        "Propagation": "rprivate"
                    },
                    {
                        "Type": "bind",
                        "Source": "/home/ish/test/airflow/logs",
                        "Destination": "/opt/airflow/logs",
                        "Mode": "rw",
                        "RW": true,
                        "Propagation": "rprivate"
                    },
                    {
                        "Type": "bind",
                        "Source": "/home/ish/test/airflow/plugins",
                        "Destination": "/opt/airflow/plugins",
                        "Mode": "rw",
                        "RW": true,
                        "Propagation": "rprivate"
                    },
                    {
                        "Type": "bind",
                        "Source": "/var/run/docker.sock",
                        "Destination": "/var/run/docker.sock",
                        "Mode": "rw",
                        "RW": true,
                        "Propagation": "rprivate"
                    }
                ]
            },
            {
                "Id": "3ad1b1d9a240c892dcd45b2981903f24f93e50257cc318f59580afe2fa782c30",
                "Names": [
                    "/airflow-airflow-webserver-1"
                ],
                "Image": "apache/airflow:2.7.2",
                "ImageID": "sha256:cf153fd692defe76b8abab371c8b64c21d7a838500a9ae06250d5984b2d51324",
                "Command": "/usr/bin/dumb-init -- /entrypoint webserver",
                "Created": 1698736049,
                "Ports": [
                    {
                        "IP": "0.0.0.0",
                        "PrivatePort": 8080,
                        "PublicPort": 8080,
                        "Type": "tcp"
                    },
                    {
                        "IP": "::",
                        "PrivatePort": 8080,
                        "PublicPort": 8080,
                        "Type": "tcp"
                    }
                ],
                "Labels": {
                    "com.docker.compose.config-hash": "8016cc0e12ad230b41a089cf353de6ae844c56089b5c6935823854bee44b1118",
                    "com.docker.compose.container-number": "1",
                    "com.docker.compose.depends_on": "airflow-init:service_completed_successfully:false,redis:service_healthy:false,postgres:service_healthy:false",
                    "com.docker.compose.image": "sha256:cf153fd692defe76b8abab371c8b64c21d7a838500a9ae06250d5984b2d51324",
                    "com.docker.compose.oneoff": "False",
                    "com.docker.compose.project": "airflow",
                    "com.docker.compose.project.config_files": "/home/ish/test/airflow/docker-compose.yaml",
                    "com.docker.compose.project.working_dir": "/home/ish/test/airflow",
                    "com.docker.compose.service": "airflow-webserver",
                    "com.docker.compose.version": "2.21.0",
                    "org.apache.airflow.component": "airflow",
                    "org.apache.airflow.distro": "debian",
                    "org.apache.airflow.image": "airflow",
                    "org.apache.airflow.main-image.build-id": "",
                    "org.apache.airflow.main-image.commit-sha": "c8b25cb3eea2bcdf951ed7c1d7d0a1f9f04db206",
                    "org.apache.airflow.module": "airflow",
                    "org.apache.airflow.uid": "50000",
                    "org.apache.airflow.version": "2.7.2",
                    "org.opencontainers.image.authors": "dev@airflow.apache.org",
                    "org.opencontainers.image.created": "",
                    "org.opencontainers.image.description": "Reference, production-ready Apache Airflow image",
                    "org.opencontainers.image.documentation": "https://airflow.apache.org/docs/docker-stack/index.html",
                    "org.opencontainers.image.licenses": "Apache-2.0",
                    "org.opencontainers.image.ref.name": "airflow",
                    "org.opencontainers.image.revision": "c8b25cb3eea2bcdf951ed7c1d7d0a1f9f04db206",
                    "org.opencontainers.image.source": "https://github.com/apache/airflow",
                    "org.opencontainers.image.title": "Production Airflow Image",
                    "org.opencontainers.image.url": "https://airflow.apache.org",
                    "org.opencontainers.image.vendor": "Apache Software Foundation",
                    "org.opencontainers.image.version": "2.7.2"
                },
                "State": "running",
                "Status": "Up 21 hours (healthy)",
                "HostConfig": {
                    "NetworkMode": "airflow_default"
                },
                "NetworkSettings": {
                    "Networks": {
                        "airflow_default": {
                            "IPAMConfig": null,
                            "Links": null,
                            "Aliases": null,
                            "NetworkID": "9cc4698f0d3c6a5e474b0fbefc28ba3c1aa15497e03d25479029b4908aa94954",
                            "EndpointID": "a9236539bb967682f8a98e266d1a67f97ef24da03c55b61eaef4c54d90a33498",
                            "Gateway": "192.168.64.1",
                            "IPAddress": "192.168.64.5",
                            "IPPrefixLen": 20,
                            "IPv6Gateway": "",
                            "GlobalIPv6Address": "",
                            "GlobalIPv6PrefixLen": 0,
                            "MacAddress": "02:42:c0:a8:40:05",
                            "DriverOpts": null
                        }
                    }
                },
                "Mounts": [
                    {
                        "Type": "bind",
                        "Source": "/data",
                        "Destination": "/data",
                        "Mode": "rw",
                        "RW": true,
                        "Propagation": "rprivate"
                    },
                    {
                        "Type": "bind",
                        "Source": "/home/ish/test/airflow/config",
                        "Destination": "/opt/airflow/config",
                        "Mode": "rw",
                        "RW": true,
                        "Propagation": "rprivate"
                    },
                    {
                        "Type": "bind",
                        "Source": "/home/ish/test/airflow/dags",
                        "Destination": "/opt/airflow/dags",
                        "Mode": "rw",
                        "RW": true,
                        "Propagation": "rprivate"
                    },
                    {
                        "Type": "bind",
                        "Source": "/home/ish/test/airflow/logs",
                        "Destination": "/opt/airflow/logs",
                        "Mode": "rw",
                        "RW": true,
                        "Propagation": "rprivate"
                    },
                    {
                        "Type": "bind",
                        "Source": "/home/ish/test/airflow/plugins",
                        "Destination": "/opt/airflow/plugins",
                        "Mode": "rw",
                        "RW": true,
                        "Propagation": "rprivate"
                    },
                    {
                        "Type": "bind",
                        "Source": "/var/run/docker.sock",
                        "Destination": "/var/run/docker.sock",
                        "Mode": "rw",
                        "RW": true,
                        "Propagation": "rprivate"
                    }
                ]
            },
            {
                "Id": "4c8eb2892ae807545f6a5be1980551bd64df6a82081eb0b5c249ad0112bc2028",
                "Names": [
                    "/airflow-airflow-init-1"
                ],
                "Image": "apache/airflow:2.7.2",
                "ImageID": "sha256:cf153fd692defe76b8abab371c8b64c21d7a838500a9ae06250d5984b2d51324",
                "Command": "/bin/bash -c 'function ver() {\n  printf \"%04d%04d%04d%04d\" ${1//./ }\n}\nairflow_version=$(AIRFLOW__LOGGING__LOGGING_LEVEL=INFO && gosu airflow airflow version)\nairflow_version_comparable=$(ver ${airflow_version})\nmin_airflow_version=2.2.0\nmin_airflow_version_comparable=$(ver ${min_airflow_version})\nif (( airflow_version_comparable < min_airflow_version_comparable )); then\n  echo\n  echo -e \"\\033[1;31mERROR!!!: Too old Airflow version ${airflow_version}!\\e[0m\"\n  echo \"The minimum Airflow version supported: ${min_airflow_version}. Only use this or higher!\"\n  echo\n  exit 1\nfi\nif [[ -z \"1000\" ]]; then\n  echo\n  echo -e \"\\033[1;33mWARNING!!!: AIRFLOW_UID not set!\\e[0m\"\n  echo \"If you are on Linux, you SHOULD follow the instructions below to set \"\n  echo \"AIRFLOW_UID environment variable, otherwise files will be owned by root.\"\n  echo \"For other operating systems you can get rid of the warning with manually created .env file:\"\n  echo \"    See: https://airflow.apache.org/docs/apache-airflow/stable/howto/docker-compose/index.html#setting-the-right-airflow-user\"\n  echo\nfi\none_meg=1048576\nmem_available=$(($(getconf _PHYS_PAGES) * $(getconf PAGE_SIZE) / one_meg))\ncpus_available=$(grep -cE 'cpu[0-9]+' /proc/stat)\ndisk_available=$(df / | tail -1 | awk '{print $4}')\nwarning_resources=\"false\"\nif (( mem_available < 4000 )) ; then\n  echo\n  echo -e \"\\033[1;33mWARNING!!!: Not enough memory available for Docker.\\e[0m\"\n  echo \"At least 4GB of memory required. You have $(numfmt --to iec $((mem_available * one_meg)))\"\n  echo\n  warning_resources=\"true\"\nfi\nif (( cpus_available < 2 )); then\n  echo\n  echo -e \"\\033[1;33mWARNING!!!: Not enough CPUS available for Docker.\\e[0m\"\n  echo \"At least 2 CPUs recommended. You have ${cpus_available}\"\n  echo\n  warning_resources=\"true\"\nfi\nif (( disk_available < one_meg * 10 )); then\n  echo\n  echo -e \"\\033[1;33mWARNING!!!: Not enough Disk space available for Docker.\\e[0m\"\n  echo \"At least 10 GBs recommended. You have $(numfmt --to iec $((disk_available * 1024 )))\"\n  echo\n  warning_resources=\"true\"\nfi\nif [[ ${warning_resources} == \"true\" ]]; then\n  echo\n  echo -e \"\\033[1;33mWARNING!!!: You have not enough resources to run Airflow (see above)!\\e[0m\"\n  echo \"Please follow the instructions to increase amount of resources available:\"\n  echo \"   https://airflow.apache.org/docs/apache-airflow/stable/howto/docker-compose/index.html#before-you-begin\"\n  echo\nfi\nmkdir -p /sources/logs /sources/dags /sources/plugins\nchown -R \"1000:0\" /sources/{logs,dags,plugins}\nexec /entrypoint airflow version\n'",
                "Created": 1698735959,
                "Ports": [],
                "Labels": {
                    "com.docker.compose.config-hash": "c8784713e9c15a140977f48d232b75e195ca9119b48d269e3ebaa17880ff24d5",
                    "com.docker.compose.container-number": "1",
                    "com.docker.compose.depends_on": "postgres:service_healthy:false,redis:service_healthy:false",
                    "com.docker.compose.image": "sha256:cf153fd692defe76b8abab371c8b64c21d7a838500a9ae06250d5984b2d51324",
                    "com.docker.compose.oneoff": "False",
                    "com.docker.compose.project": "airflow",
                    "com.docker.compose.project.config_files": "/home/ish/test/airflow/docker-compose.yaml",
                    "com.docker.compose.project.working_dir": "/home/ish/test/airflow",
                    "com.docker.compose.service": "airflow-init",
                    "com.docker.compose.version": "2.21.0",
                    "org.apache.airflow.component": "airflow",
                    "org.apache.airflow.distro": "debian",
                    "org.apache.airflow.image": "airflow",
                    "org.apache.airflow.main-image.build-id": "",
                    "org.apache.airflow.main-image.commit-sha": "c8b25cb3eea2bcdf951ed7c1d7d0a1f9f04db206",
                    "org.apache.airflow.module": "airflow",
                    "org.apache.airflow.uid": "50000",
                    "org.apache.airflow.version": "2.7.2",
                    "org.opencontainers.image.authors": "dev@airflow.apache.org",
                    "org.opencontainers.image.created": "",
                    "org.opencontainers.image.description": "Reference, production-ready Apache Airflow image",
                    "org.opencontainers.image.documentation": "https://airflow.apache.org/docs/docker-stack/index.html",
                    "org.opencontainers.image.licenses": "Apache-2.0",
                    "org.opencontainers.image.ref.name": "airflow",
                    "org.opencontainers.image.revision": "c8b25cb3eea2bcdf951ed7c1d7d0a1f9f04db206",
                    "org.opencontainers.image.source": "https://github.com/apache/airflow",
                    "org.opencontainers.image.title": "Production Airflow Image",
                    "org.opencontainers.image.url": "https://airflow.apache.org",
                    "org.opencontainers.image.vendor": "Apache Software Foundation",
                    "org.opencontainers.image.version": "2.7.2"
                },
                "State": "exited",
                "Status": "Exited (0) 6 weeks ago",
                "HostConfig": {
                    "NetworkMode": "airflow_default"
                },
                "NetworkSettings": {
                    "Networks": {
                        "airflow_default": {
                            "IPAMConfig": null,
                            "Links": null,
                            "Aliases": null,
                            "NetworkID": "9cc4698f0d3c6a5e474b0fbefc28ba3c1aa15497e03d25479029b4908aa94954",
                            "EndpointID": "",
                            "Gateway": "",
                            "IPAddress": "",
                            "IPPrefixLen": 0,
                            "IPv6Gateway": "",
                            "GlobalIPv6Address": "",
                            "GlobalIPv6PrefixLen": 0,
                            "MacAddress": "",
                            "DriverOpts": null
                        }
                    }
                },
                "Mounts": [
                    {
                        "Type": "bind",
                        "Source": "/home/ish/test/airflow",
                        "Destination": "/sources",
                        "Mode": "rw",
                        "RW": true,
                        "Propagation": "rprivate"
                    }
                ]
            },
            {
                "Id": "86ea4619f01671b8a068e366ff0ed05d37f7c3d64983de0ad23902db05d11a70",
                "Names": [
                    "/airflow-postgres-1"
                ],
                "Image": "postgres:13",
                "ImageID": "sha256:5287b99d881510c75c727e197a4c2eee3516a68011eef55b0ccb40fe806f8c15",
                "Command": "docker-entrypoint.sh postgres",
                "Created": 1698735959,
                "Ports": [
                    {
                        "PrivatePort": 5432,
                        "Type": "tcp"
                    }
                ],
                "Labels": {
                    "com.docker.compose.config-hash": "8b0580b8805a36aa84d6a4144585137b426b856c40c56f1688a381ee83fcd6aa",
                    "com.docker.compose.container-number": "1",
                    "com.docker.compose.depends_on": "",
                    "com.docker.compose.image": "sha256:5287b99d881510c75c727e197a4c2eee3516a68011eef55b0ccb40fe806f8c15",
                    "com.docker.compose.oneoff": "False",
                    "com.docker.compose.project": "airflow",
                    "com.docker.compose.project.config_files": "/home/ish/test/airflow/docker-compose.yaml",
                    "com.docker.compose.project.working_dir": "/home/ish/test/airflow",
                    "com.docker.compose.service": "postgres",
                    "com.docker.compose.version": "2.21.0"
                },
                "State": "running",
                "Status": "Up 21 hours (healthy)",
                "HostConfig": {
                    "NetworkMode": "airflow_default"
                },
                "NetworkSettings": {
                    "Networks": {
                        "airflow_default": {
                            "IPAMConfig": null,
                            "Links": null,
                            "Aliases": null,
                            "NetworkID": "9cc4698f0d3c6a5e474b0fbefc28ba3c1aa15497e03d25479029b4908aa94954",
                            "EndpointID": "f9cc4de56bad89d59a82b74069c8fcbdaeffb4e64dab6f4b867facb64e3bc5e9",
                            "Gateway": "192.168.64.1",
                            "IPAddress": "192.168.64.2",
                            "IPPrefixLen": 20,
                            "IPv6Gateway": "",
                            "GlobalIPv6Address": "",
                            "GlobalIPv6PrefixLen": 0,
                            "MacAddress": "02:42:c0:a8:40:02",
                            "DriverOpts": null
                        }
                    }
                },
                "Mounts": [
                    {
                        "Type": "volume",
                        "Name": "airflow_postgres-db-volume",
                        "Source": "/var/lib/docker/volumes/airflow_postgres-db-volume/_data",
                        "Destination": "/var/lib/postgresql/data",
                        "Driver": "local",
                        "Mode": "z",
                        "RW": true,
                        "Propagation": ""
                    }
                ]
            },
            {
                "Id": "c449f2f7743c5253df3cccc76d575805d79e466f56050b040d48099d20776809",
                "Names": [
                    "/airflow-redis-1"
                ],
                "Image": "redis:latest",
                "ImageID": "sha256:5b0542ad1e7734b17905e99f80defc1f0a7748dd6d6f1648949eb45583d087de",
                "Command": "docker-entrypoint.sh redis-server",
                "Created": 1698735959,
                "Ports": [
                    {
                        "PrivatePort": 6379,
                        "Type": "tcp"
                    }
                ],
                "Labels": {
                    "com.docker.compose.config-hash": "2947691470bd3cfab2f627ee82c4a3da9f9ff3b26fe334382c80c69a9fbe6f67",
                    "com.docker.compose.container-number": "1",
                    "com.docker.compose.depends_on": "",
                    "com.docker.compose.image": "sha256:5b0542ad1e7734b17905e99f80defc1f0a7748dd6d6f1648949eb45583d087de",
                    "com.docker.compose.oneoff": "False",
                    "com.docker.compose.project": "airflow",
                    "com.docker.compose.project.config_files": "/home/ish/test/airflow/docker-compose.yaml",
                    "com.docker.compose.project.working_dir": "/home/ish/test/airflow",
                    "com.docker.compose.service": "redis",
                    "com.docker.compose.version": "2.21.0"
                },
                "State": "running",
                "Status": "Up 21 hours (healthy)",
                "HostConfig": {
                    "NetworkMode": "airflow_default"
                },
                "NetworkSettings": {
                    "Networks": {
                        "airflow_default": {
                            "IPAMConfig": null,
                            "Links": null,
                            "Aliases": null,
                            "NetworkID": "9cc4698f0d3c6a5e474b0fbefc28ba3c1aa15497e03d25479029b4908aa94954",
                            "EndpointID": "ae2a7b69247141896f08e6a6b551f02016e3c1cbb3b6ab08d04d76763248129d",
                            "Gateway": "192.168.64.1",
                            "IPAddress": "192.168.64.6",
                            "IPPrefixLen": 20,
                            "IPv6Gateway": "",
                            "GlobalIPv6Address": "",
                            "GlobalIPv6PrefixLen": 0,
                            "MacAddress": "02:42:c0:a8:40:06",
                            "DriverOpts": null
                        }
                    }
                },
                "Mounts": [
                    {
                        "Type": "volume",
                        "Name": "d282b3ca20bb6dc23a8a4b17b5dd204179db21948e594932e035d90f8f9e6d15",
                        "Source": "",
                        "Destination": "/data",
                        "Driver": "local",
                        "Mode": "",
                        "RW": true,
                        "Propagation": ""
                    }
                ]
            }
        ]
    },
    "podman": {
        "Containers": []
    }
}
```
</details>