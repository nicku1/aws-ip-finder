package main

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"log"
	"net"
	"os"
)

type VPC struct {
	vpcName string
}

type Subnet struct {
	vpcId      string
	subnetId   string
	subnetName string
	subnetCidr string
}

func isInSubnet(subnet *Subnet, ip string) bool {
	_, subnetNet, err := net.ParseCIDR(subnet.subnetCidr)
	if err != nil {
		log.Fatal(err)
	}

	return subnetNet.Contains(net.ParseIP(ip))
}

func validateIp(ip string) {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		log.Fatalf("Provided argument is not IP Address! Arg: %v", ip)
	}

	if parsed.To4() == nil {
		log.Fatal("This program currently does not support IPv6 addresses!")
	}
}

func main() {
	if len(os.Args) < 2 {
		println("Usage: ", os.Args[0], "<IPv4>")
		println("Example: ", os.Args[0], "10.0.13.57")
		os.Exit(1)
	}

	validateIp(os.Args[1])

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	client := ec2.NewFromConfig(cfg)
	filter := "is-default"
	vpcs, err := client.DescribeVpcs(context.TODO(), &ec2.DescribeVpcsInput{
		Filters: []types.Filter{
			{
				Name:   &filter,
				Values: []string{"false"},
			},
		},
	})

	if err != nil {
		log.Fatal(err)
	}

	vpcMap := make(map[string]VPC)
	vpcIdList := make([]string, 0)

	for _, v := range vpcs.Vpcs {
		var vpcName string
		for _, tag := range v.Tags {
			if *tag.Key == "Name" {
				vpcName = *tag.Value
				break
			}
			vpcName = ""
		}
		vpcIdList = append(vpcIdList, *v.VpcId)
		vpcMap[*v.VpcId] = VPC{
			vpcName: vpcName,
		}
	}

	filter = "vpc-id"
	subnets, err := client.DescribeSubnets(context.TODO(), &ec2.DescribeSubnetsInput{
		Filters: []types.Filter{
			{
				Name:   &filter,
				Values: vpcIdList,
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	subnetArray := make([]Subnet, len(subnets.Subnets))

	for i, v := range subnets.Subnets {
		var subnetName string
		for _, tag := range v.Tags {
			if *tag.Key == "Name" {
				subnetName = *tag.Value
				break
			}
			subnetName = ""
		}
		subnetArray[i] = Subnet{
			vpcId:      *v.VpcId,
			subnetId:   *v.SubnetId,
			subnetName: subnetName,
			subnetCidr: *v.CidrBlock,
		}
	}

	filter = "addresses.private-ip-address"
	interfaces, err := client.DescribeNetworkInterfaces(context.TODO(), &ec2.DescribeNetworkInterfacesInput{
		Filters: []types.Filter{
			{
				Name: &filter,
				Values: []string{
					os.Args[1],
				},
			},
		},
	})

	for _, subnet := range subnetArray {
		if isInSubnet(&subnet, os.Args[1]) {
			println("VPC: " + vpcMap[subnet.vpcId].vpcName)
			println("Subnet: " + subnet.subnetName)
			if len(interfaces.NetworkInterfaces) > 0 {
				if interfaces.NetworkInterfaces[0].Attachment.InstanceId != nil {
					println("Resource: ", *interfaces.NetworkInterfaces[0].Attachment.InstanceId)
				}
				if interfaces.NetworkInterfaces[0].Description != nil {
					println("Resource desc: ", *interfaces.NetworkInterfaces[0].Description)
				}
			}
			os.Exit(0)
		}
	}
	println("Not found")
}
