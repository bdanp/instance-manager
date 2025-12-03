package aws

import (
	"crypto/md5"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"instance-manager/pkg/cloud"
	"instance-manager/pkg/models"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// Provider implements the CloudProvider interface for AWS
type Provider struct {
	ec2Client *ec2.EC2
	region    string
}

// NewProvider creates a new AWS provider instance
func NewProvider(region, accessKey, secretKey string) (cloud.CloudProvider, error) {
	if region == "" {
		return nil, errors.New("region is required")
	}
	if accessKey == "" {
		return nil, errors.New("AWS_ACCESS_KEY_ID environment variable is required")
	}
	if secretKey == "" {
		return nil, errors.New("AWS_SECRET_ACCESS_KEY environment variable is required")
	}

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	return &Provider{
		ec2Client: ec2.New(sess),
		region:    region,
	}, nil
}

// ValidateCredentials checks if AWS credentials are valid
func (p *Provider) ValidateCredentials() error {
	_, err := p.ec2Client.DescribeRegions(&ec2.DescribeRegionsInput{})
	if err != nil {
		return fmt.Errorf("invalid AWS credentials: %w", err)
	}
	return nil
}

// CreateInstance creates a new EC2 instance
func (p *Provider) CreateInstance(config models.InstanceConfig) (*models.Instance, error) {
	// Read and import the public key
	keyName, err := p.importKeyPair(config.PublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to import key pair: %w", err)
	}

	// Get the default VPC and subnet
	subnetID, err := p.getDefaultSubnet(config.AvailabilityZone)
	if err != nil {
		return nil, fmt.Errorf("failed to get default subnet: %w", err)
	}

	// Create security group if it doesn't exist
	securityGroupID, err := p.createOrGetSecurityGroup()
	if err != nil {
		return nil, fmt.Errorf("failed to create security group: %w", err)
	}

	// Get the latest Amazon Linux 2 AMI
	amiID, err := p.getLatestAmazonLinuxAMI()
	if err != nil {
		// Fallback to a known working AMI ID based on region
		amiID = p.getAMIID()
	}

	// Launch the instance
	runResult, err := p.ec2Client.RunInstances(&ec2.RunInstancesInput{
		ImageId:      aws.String(amiID),
		InstanceType: aws.String(config.InstanceType),
		MinCount:     aws.Int64(1),
		MaxCount:     aws.Int64(1),
		KeyName:      aws.String(keyName),
		NetworkInterfaces: []*ec2.InstanceNetworkInterfaceSpecification{
			{
				DeviceIndex:              aws.Int64(0),
				SubnetId:                 aws.String(subnetID),
				Groups:                   []*string{aws.String(securityGroupID)},
				AssociatePublicIpAddress: aws.Bool(true), // This ensures public IP assignment
			},
		},
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String("instance"),
				Tags: []*ec2.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String("instance-manager"),
					},
					{
						Key:   aws.String("ManagedBy"),
						Value: aws.String("instance-manager"),
					},
					{
						Key:   aws.String("Duration"),
						Value: aws.String(config.Duration.String()),
					},
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to launch instance: %w", err)
	}

	instanceID := *runResult.Instances[0].InstanceId
	launchTime := time.Now()
	expiresAt := launchTime.Add(config.Duration)

	instance := &models.Instance{
		ID:               instanceID,
		InstanceType:     config.InstanceType,
		State:            "pending",
		LaunchTime:       launchTime,
		Duration:         config.Duration,
		AvailabilityZone: config.AvailabilityZone,
		KeyName:          keyName,
		Username:         "ec2-user", // Default username for Amazon Linux
		ExpiresAt:        expiresAt,
	}

	return instance, nil
}

// GetInstanceStatus retrieves the status of an instance
func (p *Provider) GetInstanceStatus(instanceID string) (*models.InstanceStatus, error) {
	result, err := p.ec2Client.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(instanceID)},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe instance: %w", err)
	}

	if len(result.Reservations) == 0 || len(result.Reservations[0].Instances) == 0 {
		return nil, errors.New("instance not found")
	}

	instance := result.Reservations[0].Instances[0]
	status := &models.InstanceStatus{
		ID:    instanceID,
		State: *instance.State.Name,
		Ready: *instance.State.Name == "running",
	}

	if instance.PublicIpAddress != nil {
		status.PublicIP = *instance.PublicIpAddress
	}
	if instance.PrivateIpAddress != nil {
		status.PrivateIP = *instance.PrivateIpAddress
	}

	// Get username from AMI
	status.Username = "ec2-user"

	return status, nil
}

// StartInstance starts a stopped EC2 instance
func (p *Provider) StartInstance(instanceID string) error {
	_, err := p.ec2Client.StartInstances(&ec2.StartInstancesInput{
		InstanceIds: []*string{aws.String(instanceID)},
	})
	if err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}
	return nil
}

// StopInstance stops a running EC2 instance
func (p *Provider) StopInstance(instanceID string) error {
	_, err := p.ec2Client.StopInstances(&ec2.StopInstancesInput{
		InstanceIds: []*string{aws.String(instanceID)},
	})
	if err != nil {
		return fmt.Errorf("failed to stop instance: %w", err)
	}
	return nil
}

// TerminateInstance terminates an EC2 instance
func (p *Provider) TerminateInstance(instanceID string) error {
	_, err := p.ec2Client.TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: []*string{aws.String(instanceID)},
	})
	if err != nil {
		return fmt.Errorf("failed to terminate instance: %w", err)
	}
	return nil
}

// ListInstances lists all instances managed by this tool
func (p *Provider) ListInstances() ([]*models.Instance, error) {
	result, err := p.ec2Client.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:ManagedBy"),
				Values: []*string{aws.String("instance-manager")},
			},
			{
				Name: aws.String("instance-state-name"),
				Values: []*string{
					aws.String("pending"),
					aws.String("running"),
					aws.String("stopping"),
					aws.String("stopped"),
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	var instances []*models.Instance
	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			inst := &models.Instance{
				ID:           *instance.InstanceId,
				InstanceType: *instance.InstanceType,
				State:        *instance.State.Name,
				LaunchTime:   *instance.LaunchTime,
			}

			if instance.PublicIpAddress != nil {
				inst.PublicIP = *instance.PublicIpAddress
			}
			if instance.PrivateIpAddress != nil {
				inst.PrivateIP = *instance.PrivateIpAddress
			}
			if instance.Placement != nil && instance.Placement.AvailabilityZone != nil {
				inst.AvailabilityZone = *instance.Placement.AvailabilityZone
			}
			if instance.KeyName != nil {
				inst.KeyName = *instance.KeyName
			}

			// Get duration from tags
			for _, tag := range instance.Tags {
				if *tag.Key == "Duration" {
					duration, err := time.ParseDuration(*tag.Value)
					if err == nil {
						inst.Duration = duration
						inst.ExpiresAt = inst.LaunchTime.Add(duration)
					}
				}
			}

			inst.Username = "ec2-user"
			instances = append(instances, inst)
		}
	}

	return instances, nil
}

// importKeyPair imports a public key to AWS
func (p *Provider) importKeyPair(publicKeyPath string) (string, error) {
	keyData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read public key file: %w", err)
	}

	// Generate a unique key name based on the key content
	hasher := md5.New()
	hasher.Write(keyData)
	keyName := fmt.Sprintf("instance-manager-%x", hasher.Sum(nil)[:8])

	// Check if key already exists
	_, err = p.ec2Client.DescribeKeyPairs(&ec2.DescribeKeyPairsInput{
		KeyNames: []*string{aws.String(keyName)},
	})
	if err == nil {
		// Key already exists
		return keyName, nil
	}

	// Import the key
	_, err = p.ec2Client.ImportKeyPair(&ec2.ImportKeyPairInput{
		KeyName:           aws.String(keyName),
		PublicKeyMaterial: keyData,
	})
	if err != nil {
		return "", fmt.Errorf("failed to import key pair: %w", err)
	}

	return keyName, nil
}

// getDefaultSubnet gets the default subnet for the specified AZ, or any available subnet
func (p *Provider) getDefaultSubnet(availabilityZone string) (string, error) {
	// First try to find default subnet in the specified AZ
	result, err := p.ec2Client.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("availability-zone"),
				Values: []*string{aws.String(availabilityZone)},
			},
			{
				Name:   aws.String("default-for-az"),
				Values: []*string{aws.String("true")},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to describe subnets: %w", err)
	}

	if len(result.Subnets) > 0 {
		return *result.Subnets[0].SubnetId, nil
	}

	// If no default subnet found, try to find any subnet in the specified AZ
	result, err = p.ec2Client.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("availability-zone"),
				Values: []*string{aws.String(availabilityZone)},
			},
			{
				Name:   aws.String("state"),
				Values: []*string{aws.String("available")},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to describe subnets: %w", err)
	}

	if len(result.Subnets) > 0 {
		return *result.Subnets[0].SubnetId, nil
	}

	// If still no subnet found, try to find any subnet in any AZ in the region
	result, err = p.ec2Client.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("state"),
				Values: []*string{aws.String("available")},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to describe subnets: %w", err)
	}

	if len(result.Subnets) == 0 {
		return "", fmt.Errorf("no available subnets found in region %s. Please create a VPC and subnet first", p.region)
	}

	// Use the first available subnet and log a warning
	fmt.Printf("Warning: No subnet found in %s, using subnet %s in %s\n",
		availabilityZone,
		*result.Subnets[0].SubnetId,
		*result.Subnets[0].AvailabilityZone)

	return *result.Subnets[0].SubnetId, nil
}

// createOrGetSecurityGroup creates or gets the security group for SSH access
func (p *Provider) createOrGetSecurityGroup() (string, error) {
	groupName := "instance-manager-sg"

	// Check if security group exists
	result, err := p.ec2Client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("group-name"),
				Values: []*string{aws.String(groupName)},
			},
		},
	})
	if err == nil && len(result.SecurityGroups) > 0 {
		return *result.SecurityGroups[0].GroupId, nil
	}

	// First try to get default VPC
	vpcResult, err := p.ec2Client.DescribeVpcs(&ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("is-default"),
				Values: []*string{aws.String("true")},
			},
		},
	})

	var vpcID string
	if err != nil || len(vpcResult.Vpcs) == 0 {
		// No default VPC, find any VPC
		vpcResult, err = p.ec2Client.DescribeVpcs(&ec2.DescribeVpcsInput{
			Filters: []*ec2.Filter{
				{
					Name:   aws.String("state"),
					Values: []*string{aws.String("available")},
				},
			},
		})
		if err != nil || len(vpcResult.Vpcs) == 0 {
			return "", fmt.Errorf("no available VPCs found. Please create a VPC first")
		}
		vpcID = *vpcResult.Vpcs[0].VpcId
		fmt.Printf("Warning: No default VPC found, using VPC %s\n", vpcID)
	} else {
		vpcID = *vpcResult.Vpcs[0].VpcId
	}

	// Create security group
	createResult, err := p.ec2Client.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(groupName),
		Description: aws.String("Security group for instance-manager"),
		VpcId:       aws.String(vpcID),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create security group: %w", err)
	}

	securityGroupID := *createResult.GroupId

	// Add SSH rule
	_, err = p.ec2Client.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(securityGroupID),
		IpPermissions: []*ec2.IpPermission{
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int64(22),
				ToPort:     aws.Int64(22),
				IpRanges: []*ec2.IpRange{
					{
						CidrIp: aws.String("0.0.0.0/0"),
					},
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to add SSH rule to security group: %w", err)
	}

	return securityGroupID, nil
}

// getAMIID returns a fallback AMI ID for Amazon Linux 2
func (p *Provider) getAMIID() string {
	// Updated AMI IDs for Amazon Linux 2 (as of late 2024)
	amiMap := map[string]string{
		"us-east-1a":     "ami-0c02fb55956c7d316", // Amazon Linux 2 AMI (HVM) - Kernel 5.10
		"us-east-2":      "ami-0f924dc71d44d23e2",
		"us-west-1":      "ami-0d382e80be7ffdae5",
		"us-west-2":      "ami-0c2d3e23eb6b42bd5",
		"eu-west-1":      "ami-0c9c942bd7bf113a2",
		"eu-central-1":   "ami-0a1ee2fb28fe05df3",
		"ap-southeast-1": "ami-0c802847a7dd848c0",
		"ap-northeast-1": "ami-0218d08a1f9dae831",
	}

	if ami, ok := amiMap[p.region]; ok {
		return ami
	}

	// Fallback to us-east-1a AMI if region not found
	return amiMap["us-east-1a"]
}

// getLatestAmazonLinuxAMI gets the latest Amazon Linux 2 AMI for the current region
func (p *Provider) getLatestAmazonLinuxAMI() (string, error) {
	result, err := p.ec2Client.DescribeImages(&ec2.DescribeImagesInput{
		Owners: []*string{aws.String("amazon")},
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("name"),
				Values: []*string{aws.String("amzn2-ami-hvm-*-x86_64-gp2")},
			},
			{
				Name:   aws.String("state"),
				Values: []*string{aws.String("available")},
			},
		},
	})
	if err != nil {
		return "", err
	}

	if len(result.Images) == 0 {
		return "", errors.New("no Amazon Linux 2 AMI found")
	}

	// Sort by creation date and return the latest
	latest := result.Images[0]
	for _, image := range result.Images[1:] {
		if image.CreationDate != nil && latest.CreationDate != nil {
			if strings.Compare(*image.CreationDate, *latest.CreationDate) > 0 {
				latest = image
			}
		}
	}

	return *latest.ImageId, nil
}
