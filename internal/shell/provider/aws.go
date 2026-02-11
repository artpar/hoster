package provider

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	smithy "github.com/aws/smithy-go"

	coreprovider "github.com/artpar/hoster/internal/core/provider"
)

// AWSProvider implements Provider for AWS EC2.
type AWSProvider struct {
	accessKeyID     string
	secretAccessKey string
	logger          *slog.Logger
}

// NewAWSProvider creates a new AWS EC2 provider.
func NewAWSProvider(accessKeyID, secretAccessKey string, logger *slog.Logger) *AWSProvider {
	return &AWSProvider{
		accessKeyID:     accessKeyID,
		secretAccessKey: secretAccessKey,
		logger:          logger.With("provider", "aws"),
	}
}

func (p *AWSProvider) newClient(region string) *ec2.Client {
	return ec2.New(ec2.Options{
		Region:      region,
		Credentials: credentials.NewStaticCredentialsProvider(p.accessKeyID, p.secretAccessKey, ""),
	})
}

// CreateInstance provisions an EC2 instance with Docker pre-installed via user data.
func (p *AWSProvider) CreateInstance(ctx context.Context, req ProvisionRequest) (*ProvisionResult, error) {
	client := p.newClient(req.Region)

	// Import the SSH key (idempotent: delete existing key first if present)
	keyName := fmt.Sprintf("hoster-%s", req.InstanceName)
	_, _ = client.DeleteKeyPair(ctx, &ec2.DeleteKeyPairInput{
		KeyName: aws.String(keyName),
	})
	_, err := client.ImportKeyPair(ctx, &ec2.ImportKeyPairInput{
		KeyName:           aws.String(keyName),
		PublicKeyMaterial: []byte(req.SSHPublicKey),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to import SSH key: %w", err)
	}

	// Create security group
	sgName := fmt.Sprintf("hoster-%s", req.InstanceName)
	sgOut, err := client.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(sgName),
		Description: aws.String("Hoster managed node - " + req.InstanceName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create security group: %w", err)
	}

	// Allow SSH (22) and Docker-related ports
	_, err = client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: sgOut.GroupId,
		IpPermissions: []ec2types.IpPermission{
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(22),
				ToPort:     aws.Int32(22),
				IpRanges:   []ec2types.IpRange{{CidrIp: aws.String("0.0.0.0/0"), Description: aws.String("SSH")}},
			},
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(80),
				ToPort:     aws.Int32(80),
				IpRanges:   []ec2types.IpRange{{CidrIp: aws.String("0.0.0.0/0"), Description: aws.String("HTTP")}},
			},
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(443),
				ToPort:     aws.Int32(443),
				IpRanges:   []ec2types.IpRange{{CidrIp: aws.String("0.0.0.0/0"), Description: aws.String("HTTPS")}},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to configure security group: %w", err)
	}

	// Find latest Ubuntu 22.04 AMI
	amiOut, err := client.DescribeImages(ctx, &ec2.DescribeImagesInput{
		Filters: []ec2types.Filter{
			{Name: aws.String("name"), Values: []string{"ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*"}},
			{Name: aws.String("state"), Values: []string{"available"}},
		},
		Owners: []string{"099720109477"}, // Canonical
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find Ubuntu AMI: %w", err)
	}
	if len(amiOut.Images) == 0 {
		return nil, errors.New("no Ubuntu AMI found")
	}
	// Use the most recent image (they're sorted by creation date)
	ami := amiOut.Images[0]
	for _, img := range amiOut.Images[1:] {
		if aws.ToString(img.CreationDate) > aws.ToString(ami.CreationDate) {
			ami = img
		}
	}

	// Cloud-init user data to install Docker
	userData := dockerInstallUserData()

	// Launch instance
	runOut, err := client.RunInstances(ctx, &ec2.RunInstancesInput{
		ImageId:          ami.ImageId,
		InstanceType:     ec2types.InstanceType(req.Size),
		KeyName:          aws.String(keyName),
		SecurityGroupIds: []string{*sgOut.GroupId},
		MinCount:         aws.Int32(1),
		MaxCount:         aws.Int32(1),
		UserData:         aws.String(userData),
		TagSpecifications: []ec2types.TagSpecification{
			{
				ResourceType: ec2types.ResourceTypeInstance,
				Tags: []ec2types.Tag{
					{Key: aws.String("Name"), Value: aws.String(req.InstanceName)},
					{Key: aws.String("ManagedBy"), Value: aws.String("hoster")},
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to launch instance: %w", err)
	}

	if len(runOut.Instances) == 0 {
		return nil, errors.New("no instance returned from RunInstances")
	}

	instanceID := aws.ToString(runOut.Instances[0].InstanceId)
	p.logger.Info("EC2 instance launched", "instance_id", instanceID, "region", req.Region)

	// Wait for running state and public IP
	publicIP, err := p.waitForPublicIP(ctx, client, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed waiting for public IP: %w", err)
	}

	return &ProvisionResult{
		ProviderInstanceID: instanceID,
		PublicIP:           publicIP,
	}, nil
}

func (p *AWSProvider) waitForPublicIP(ctx context.Context, client *ec2.Client, instanceID string) (string, error) {
	for i := 0; i < 60; i++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(5 * time.Second):
		}

		out, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
			InstanceIds: []string{instanceID},
		})
		if err != nil {
			continue
		}

		for _, res := range out.Reservations {
			for _, inst := range res.Instances {
				if inst.PublicIpAddress != nil && *inst.PublicIpAddress != "" {
					return *inst.PublicIpAddress, nil
				}
			}
		}
	}
	return "", errors.New("timed out waiting for public IP")
}

// DestroyInstance terminates an EC2 instance and cleans up SSH key and security group.
func (p *AWSProvider) DestroyInstance(ctx context.Context, req DestroyRequest) error {
	client := p.newClient(req.Region)

	_, err := client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{req.ProviderInstanceID},
	})
	if err != nil {
		// Treat "instance not found" as success — already terminated/deleted
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "InvalidInstanceID.NotFound" {
			p.logger.Info("EC2 instance already terminated", "instance_id", req.ProviderInstanceID)
		} else {
			return fmt.Errorf("failed to terminate instance %s: %w", req.ProviderInstanceID, err)
		}
	} else {
		p.logger.Info("EC2 instance terminated", "instance_id", req.ProviderInstanceID, "region", req.Region)
	}

	// Best-effort cleanup of SSH key
	keyName := fmt.Sprintf("hoster-%s", req.InstanceName)
	if _, err := client.DeleteKeyPair(ctx, &ec2.DeleteKeyPairInput{
		KeyName: aws.String(keyName),
	}); err != nil {
		p.logger.Warn("failed to delete SSH key pair during destroy", "key_name", keyName, "error", err)
	}

	// Best-effort cleanup of security group
	sgName := fmt.Sprintf("hoster-%s", req.InstanceName)
	if _, err := client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
		GroupName: aws.String(sgName),
	}); err != nil {
		p.logger.Warn("failed to delete security group during destroy", "sg_name", sgName, "error", err)
	}

	return nil
}

// ListRegions returns available AWS regions.
func (p *AWSProvider) ListRegions(ctx context.Context) ([]coreprovider.Region, error) {
	client := p.newClient("us-east-1")
	out, err := client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{
		Filters: []ec2types.Filter{
			{Name: aws.String("opt-in-status"), Values: []string{"opt-in-not-required", "opted-in"}},
		},
	})
	if err != nil {
		// Fall back to static catalog
		return coreprovider.AWSRegions(), nil
	}

	regions := make([]coreprovider.Region, 0, len(out.Regions))
	for _, r := range out.Regions {
		regions = append(regions, coreprovider.Region{
			ID:        aws.ToString(r.RegionName),
			Name:      aws.ToString(r.RegionName),
			Available: true,
		})
	}
	return regions, nil
}

// ListSizes returns available EC2 instance types.
func (p *AWSProvider) ListSizes(ctx context.Context, region string) ([]coreprovider.InstanceSize, error) {
	// EC2 DescribeInstanceTypes is complex and slow; use static catalog
	return coreprovider.AWSSizes(), nil
}

// dockerInstallUserData returns a base64-encoded cloud-init script that installs Docker.
func dockerInstallUserData() string {
	return `#!/bin/bash
set -e
apt-get update -y
apt-get install -y ca-certificates curl gnupg
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
chmod a+r /etc/apt/keyrings/docker.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
apt-get update -y
apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
systemctl enable docker
systemctl start docker
`
}
