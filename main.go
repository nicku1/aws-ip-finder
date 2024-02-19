package aws_ip_finder

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
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
	cidrSplit := strings.Split(subnet.subnetCidr, "/")
	maskLength, err := strconv.Atoi(cidrSplit[1])
	if err != nil {
		log.Fatal(err)
	}
	subnetNet := net.IPNet{
		IP:   net.ParseIP(cidrSplit[0]),
		Mask: make(net.IPMask, maskLength),
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

	for _, v := range vpcs.Vpcs {
		var vpcName string
		for _, tag := range v.Tags {
			if *tag.Key == "Name" {
				vpcName = *tag.Value
				break
			}
			vpcName = ""
		}
		vpcMap[*v.VpcId] = VPC{
			vpcName: vpcName,
		}
	}

	//todo list of filters with vpc ids
	subnets, err := client.DescribeSubnets(context.TODO(), &ec2.DescribeSubnetsInput{})
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
}
