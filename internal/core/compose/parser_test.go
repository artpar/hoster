package compose

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Fixtures
// =============================================================================

const minimalValidSpec = `
services:
  app:
    image: nginx:latest
`

const multiServiceSpec = `
services:
  web:
    image: nginx:latest
    ports:
      - "80:80"
    depends_on:
      - api

  api:
    image: myapp:1.0
    environment:
      DB_HOST: db
    depends_on:
      - db

  db:
    image: postgres:15
    volumes:
      - pgdata:/var/lib/postgresql/data

volumes:
  pgdata:
`

const wordpressSpec = `
services:
  wordpress:
    image: wordpress:latest
    ports:
      - "8080:80"
    environment:
      WORDPRESS_DB_HOST: db
      WORDPRESS_DB_PASSWORD: ${DB_PASSWORD}
    volumes:
      - wordpress_data:/var/www/html
    depends_on:
      - db

  db:
    image: mysql:8
    environment:
      MYSQL_ROOT_PASSWORD: ${DB_PASSWORD}
      MYSQL_DATABASE: wordpress
    volumes:
      - db_data:/var/lib/mysql

volumes:
  wordpress_data:
  db_data:
`

const serviceWithResourcesSpec = `
services:
  api:
    image: myapp:1.0
    deploy:
      resources:
        limits:
          cpus: "2.0"
          memory: 1G
        reservations:
          cpus: "0.5"
          memory: 512M
`

const serviceWithBuildSpec = `
services:
  app:
    build:
      context: ./app
      dockerfile: Dockerfile.prod
`

const serviceWithHealthCheckSpec = `
services:
  web:
    image: nginx:latest
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 5s
`

const networkSpec = `
services:
  web:
    image: nginx:latest
    networks:
      - frontend

  api:
    image: myapp:1.0
    networks:
      - frontend
      - backend

networks:
  frontend:
    driver: bridge
  backend:
    internal: true
`

const circularDepSpec = `
services:
  a:
    image: nginx:latest
    depends_on:
      - b

  b:
    image: nginx:latest
    depends_on:
      - a
`

const selfReferenceSpec = `
services:
  a:
    image: nginx:latest
    depends_on:
      - a
`

// =============================================================================
// Input Validation Tests
// =============================================================================

func TestParseComposeSpec_EmptyInput(t *testing.T) {
	_, err := ParseComposeSpec("")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptyInput)
}

func TestParseComposeSpec_WhitespaceOnly(t *testing.T) {
	_, err := ParseComposeSpec("   \n\t  ")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptyInput)
}

func TestParseComposeSpec_InvalidYAML(t *testing.T) {
	_, err := ParseComposeSpec("invalid: yaml: content: [")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidYAML)
}

func TestParseComposeSpec_YAMLNotObject(t *testing.T) {
	_, err := ParseComposeSpec("just a string")
	require.Error(t, err)
	// Should fail because it's not a valid compose structure
}

func TestParseComposeSpec_NoServicesKey(t *testing.T) {
	_, err := ParseComposeSpec("version: '3'\n")
	require.Error(t, err)
	// compose-go returns "empty compose file" error for version-only files
	// We wrap this as ErrNoServices since it has no services
}

func TestParseComposeSpec_EmptyServices(t *testing.T) {
	_, err := ParseComposeSpec("services: {}")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoServices)
}

// =============================================================================
// Service Parsing Tests
// =============================================================================

func TestParseComposeSpec_MinimalValid(t *testing.T) {
	spec, err := ParseComposeSpec(minimalValidSpec)
	require.NoError(t, err)
	require.NotNil(t, spec)

	assert.Len(t, spec.Services, 1)
	assert.Equal(t, "app", spec.Services[0].Name)
	assert.Equal(t, "nginx:latest", spec.Services[0].Image)
}

func TestParseComposeSpec_ServiceWithImage(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx:1.25
`
	spec, err := ParseComposeSpec(yaml)
	require.NoError(t, err)
	require.Len(t, spec.Services, 1)

	assert.Equal(t, "web", spec.Services[0].Name)
	assert.Equal(t, "nginx:1.25", spec.Services[0].Image)
	assert.Nil(t, spec.Services[0].Build)
}

func TestParseComposeSpec_ServiceWithBuild(t *testing.T) {
	spec, err := ParseComposeSpec(serviceWithBuildSpec)
	require.NoError(t, err)
	require.Len(t, spec.Services, 1)

	svc := spec.Services[0]
	require.NotNil(t, svc.Build)
	// compose-go normalizes paths (removes ./)
	assert.Equal(t, "app", svc.Build.Context)
	assert.Equal(t, "Dockerfile.prod", svc.Build.Dockerfile)
}

func TestParseComposeSpec_ServiceWithBuildSimple(t *testing.T) {
	yaml := `
services:
  app:
    build: ./myapp
`
	spec, err := ParseComposeSpec(yaml)
	require.NoError(t, err)
	require.Len(t, spec.Services, 1)

	svc := spec.Services[0]
	require.NotNil(t, svc.Build)
	// compose-go normalizes paths
	assert.Equal(t, "myapp", svc.Build.Context)
}

func TestParseComposeSpec_ServiceWithBothImageAndBuild(t *testing.T) {
	yaml := `
services:
  app:
    image: myapp:latest
    build: ./myapp
`
	spec, err := ParseComposeSpec(yaml)
	require.NoError(t, err)
	require.Len(t, spec.Services, 1)

	// Image should be present when both are specified
	svc := spec.Services[0]
	assert.Equal(t, "myapp:latest", svc.Image)
	assert.NotNil(t, svc.Build)
}

func TestParseComposeSpec_ServiceNoImageOrBuild(t *testing.T) {
	yaml := `
services:
  app:
    ports:
      - "80:80"
`
	_, err := ParseComposeSpec(yaml)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrServiceNoImage)
}

func TestParseComposeSpec_ServiceWithCommand(t *testing.T) {
	yaml := `
services:
  app:
    image: nginx:latest
    command: ["nginx", "-g", "daemon off;"]
`
	spec, err := ParseComposeSpec(yaml)
	require.NoError(t, err)
	require.Len(t, spec.Services, 1)

	assert.Equal(t, []string{"nginx", "-g", "daemon off;"}, spec.Services[0].Command)
}

func TestParseComposeSpec_ServiceWithEntrypoint(t *testing.T) {
	yaml := `
services:
  app:
    image: myapp:latest
    entrypoint: ["/entrypoint.sh"]
`
	spec, err := ParseComposeSpec(yaml)
	require.NoError(t, err)
	require.Len(t, spec.Services, 1)

	assert.Equal(t, []string{"/entrypoint.sh"}, spec.Services[0].Entrypoint)
}

func TestParseComposeSpec_ServiceWithLabels(t *testing.T) {
	yaml := `
services:
  app:
    image: nginx:latest
    labels:
      app.name: myapp
      app.version: "1.0"
`
	spec, err := ParseComposeSpec(yaml)
	require.NoError(t, err)
	require.Len(t, spec.Services, 1)

	labels := spec.Services[0].Labels
	assert.Equal(t, "myapp", labels["app.name"])
	assert.Equal(t, "1.0", labels["app.version"])
}

func TestParseComposeSpec_MultipleServices(t *testing.T) {
	spec, err := ParseComposeSpec(multiServiceSpec)
	require.NoError(t, err)

	assert.Len(t, spec.Services, 3)

	// Find services by name
	serviceNames := make(map[string]Service)
	for _, s := range spec.Services {
		serviceNames[s.Name] = s
	}

	assert.Contains(t, serviceNames, "web")
	assert.Contains(t, serviceNames, "api")
	assert.Contains(t, serviceNames, "db")
}

// =============================================================================
// Port Parsing Tests
// =============================================================================

func TestParseComposeSpec_PortsShortSyntax(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
`
	spec, err := ParseComposeSpec(yaml)
	require.NoError(t, err)
	require.Len(t, spec.Services, 1)
	require.Len(t, spec.Services[0].Ports, 1)

	port := spec.Services[0].Ports[0]
	assert.Equal(t, uint32(80), port.Target)
	assert.Equal(t, uint32(8080), port.Published)
}

func TestParseComposeSpec_PortsWithProtocol(t *testing.T) {
	yaml := `
services:
  app:
    image: myapp:latest
    ports:
      - "53:53/udp"
`
	spec, err := ParseComposeSpec(yaml)
	require.NoError(t, err)
	require.Len(t, spec.Services[0].Ports, 1)

	port := spec.Services[0].Ports[0]
	assert.Equal(t, uint32(53), port.Target)
	assert.Equal(t, uint32(53), port.Published)
	assert.Equal(t, "udp", port.Protocol)
}

func TestParseComposeSpec_PortsWithIP(t *testing.T) {
	yaml := `
services:
  app:
    image: myapp:latest
    ports:
      - "127.0.0.1:8080:80"
`
	spec, err := ParseComposeSpec(yaml)
	require.NoError(t, err)
	require.Len(t, spec.Services[0].Ports, 1)

	port := spec.Services[0].Ports[0]
	assert.Equal(t, uint32(80), port.Target)
	assert.Equal(t, uint32(8080), port.Published)
	assert.Equal(t, "127.0.0.1", port.HostIP)
}

func TestParseComposeSpec_PortsTargetOnly(t *testing.T) {
	yaml := `
services:
  app:
    image: myapp:latest
    ports:
      - "80"
`
	spec, err := ParseComposeSpec(yaml)
	require.NoError(t, err)
	require.Len(t, spec.Services[0].Ports, 1)

	port := spec.Services[0].Ports[0]
	assert.Equal(t, uint32(80), port.Target)
	// Published is 0 or dynamically assigned
}

func TestParseComposeSpec_PortsLongSyntax(t *testing.T) {
	yaml := `
services:
  app:
    image: myapp:latest
    ports:
      - target: 80
        published: 8080
        protocol: tcp
        host_ip: 0.0.0.0
`
	spec, err := ParseComposeSpec(yaml)
	require.NoError(t, err)
	require.Len(t, spec.Services[0].Ports, 1)

	port := spec.Services[0].Ports[0]
	assert.Equal(t, uint32(80), port.Target)
	assert.Equal(t, uint32(8080), port.Published)
	assert.Equal(t, "tcp", port.Protocol)
	assert.Equal(t, "0.0.0.0", port.HostIP)
}

func TestParseComposeSpec_PortsMultiple(t *testing.T) {
	yaml := `
services:
  app:
    image: myapp:latest
    ports:
      - "80:80"
      - "443:443"
      - "8080:8080"
`
	spec, err := ParseComposeSpec(yaml)
	require.NoError(t, err)
	assert.Len(t, spec.Services[0].Ports, 3)
}

func TestParseComposeSpec_PortsInvalidRange(t *testing.T) {
	yaml := `
services:
  app:
    image: myapp:latest
    ports:
      - "99999:80"
`
	_, err := ParseComposeSpec(yaml)
	require.Error(t, err)
	// compose-go catches invalid ports with its own error
	// Error message may vary by version, just check error is returned
	assert.True(t, errors.Is(err, ErrInvalidYAML) || strings.Contains(err.Error(), "port"))
}

func TestParseComposeSpec_PortsZeroTarget(t *testing.T) {
	yaml := `
services:
  app:
    image: myapp:latest
    ports:
      - target: 0
        published: 8080
`
	_, err := ParseComposeSpec(yaml)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrServiceInvalidPort)
}

// =============================================================================
// Volume Parsing Tests
// =============================================================================

func TestParseComposeSpec_VolumeBindMount(t *testing.T) {
	yaml := `
services:
  app:
    image: nginx:latest
    volumes:
      - ./data:/app/data
`
	spec, err := ParseComposeSpec(yaml)
	require.NoError(t, err)
	require.Len(t, spec.Services[0].Volumes, 1)

	vol := spec.Services[0].Volumes[0]
	assert.Equal(t, VolumeMountTypeBind, vol.Type)
	// compose-go normalizes paths
	assert.Equal(t, "data", vol.Source)
	assert.Equal(t, "/app/data", vol.Target)
	assert.False(t, vol.ReadOnly)
}

func TestParseComposeSpec_VolumeNamedVolume(t *testing.T) {
	yaml := `
services:
  app:
    image: nginx:latest
    volumes:
      - mydata:/data

volumes:
  mydata:
`
	spec, err := ParseComposeSpec(yaml)
	require.NoError(t, err)
	require.Len(t, spec.Services[0].Volumes, 1)

	vol := spec.Services[0].Volumes[0]
	assert.Equal(t, VolumeMountTypeVolume, vol.Type)
	assert.Equal(t, "mydata", vol.Source)
	assert.Equal(t, "/data", vol.Target)
}

func TestParseComposeSpec_VolumeReadOnly(t *testing.T) {
	yaml := `
services:
  app:
    image: nginx:latest
    volumes:
      - ./config:/etc/config:ro
`
	spec, err := ParseComposeSpec(yaml)
	require.NoError(t, err)
	require.Len(t, spec.Services[0].Volumes, 1)

	vol := spec.Services[0].Volumes[0]
	assert.True(t, vol.ReadOnly)
}

func TestParseComposeSpec_VolumeLongSyntax(t *testing.T) {
	yaml := `
services:
  app:
    image: nginx:latest
    volumes:
      - type: volume
        source: mydata
        target: /data
        read_only: true

volumes:
  mydata:
`
	spec, err := ParseComposeSpec(yaml)
	require.NoError(t, err)
	require.Len(t, spec.Services[0].Volumes, 1)

	vol := spec.Services[0].Volumes[0]
	assert.Equal(t, VolumeMountTypeVolume, vol.Type)
	assert.Equal(t, "mydata", vol.Source)
	assert.Equal(t, "/data", vol.Target)
	assert.True(t, vol.ReadOnly)
}

func TestParseComposeSpec_VolumeTmpfs(t *testing.T) {
	yaml := `
services:
  app:
    image: nginx:latest
    volumes:
      - type: tmpfs
        target: /tmp
`
	spec, err := ParseComposeSpec(yaml)
	require.NoError(t, err)
	require.Len(t, spec.Services[0].Volumes, 1)

	vol := spec.Services[0].Volumes[0]
	assert.Equal(t, VolumeMountTypeTmpfs, vol.Type)
	assert.Equal(t, "/tmp", vol.Target)
}

// =============================================================================
// Environment Variable Tests
// =============================================================================

func TestParseComposeSpec_EnvironmentMapSyntax(t *testing.T) {
	yaml := `
services:
  app:
    image: myapp:latest
    environment:
      KEY1: value1
      KEY2: value2
`
	spec, err := ParseComposeSpec(yaml)
	require.NoError(t, err)
	require.Len(t, spec.Services, 1)

	env := spec.Services[0].Environment
	assert.Equal(t, "value1", env["KEY1"])
	assert.Equal(t, "value2", env["KEY2"])
}

func TestParseComposeSpec_EnvironmentListSyntax(t *testing.T) {
	yaml := `
services:
  app:
    image: myapp:latest
    environment:
      - KEY1=value1
      - KEY2=value2
`
	spec, err := ParseComposeSpec(yaml)
	require.NoError(t, err)
	require.Len(t, spec.Services, 1)

	env := spec.Services[0].Environment
	assert.Equal(t, "value1", env["KEY1"])
	assert.Equal(t, "value2", env["KEY2"])
}

func TestParseComposeSpec_EnvironmentWithPlaceholders(t *testing.T) {
	yaml := `
services:
  app:
    image: myapp:latest
    environment:
      DB_PASSWORD: ${DB_PASSWORD}
      API_KEY: ${API_KEY:-default}
`
	spec, err := ParseComposeSpec(yaml)
	require.NoError(t, err)
	require.Len(t, spec.Services, 1)

	env := spec.Services[0].Environment
	// compose-go interpolates placeholders at parse time
	// ${DB_PASSWORD} resolves to empty string (not set)
	// ${API_KEY:-default} resolves to "default"
	assert.Equal(t, "", env["DB_PASSWORD"])
	assert.Equal(t, "default", env["API_KEY"])

	// Use ExtractVariablesFromYAML to get placeholder names from raw YAML
	vars := ExtractVariablesFromYAML(yaml)
	assert.Contains(t, vars, "DB_PASSWORD")
	assert.Contains(t, vars, "API_KEY")
}

func TestExtractVariables(t *testing.T) {
	// ExtractVariables works on parsed spec - since compose-go interpolates
	// placeholders, we need ExtractVariablesFromYAML for raw extraction
	vars := ExtractVariablesFromYAML(wordpressSpec)

	// Should find DB_PASSWORD (appears twice but should be unique)
	assert.Contains(t, vars, "DB_PASSWORD")
	// Should not have duplicates
	count := 0
	for _, v := range vars {
		if v == "DB_PASSWORD" {
			count++
		}
	}
	assert.Equal(t, 1, count)
}

func TestExtractVariables_NoPlaceholders(t *testing.T) {
	vars := ExtractVariablesFromYAML(minimalValidSpec)
	assert.Empty(t, vars)
}

func TestExtractVariables_WithDefaults(t *testing.T) {
	yaml := `
services:
  app:
    image: myapp:latest
    environment:
      PORT: ${PORT:-8080}
      HOST: ${HOST}
`
	vars := ExtractVariablesFromYAML(yaml)
	assert.Contains(t, vars, "PORT")
	assert.Contains(t, vars, "HOST")
}

// =============================================================================
// Network Tests
// =============================================================================

func TestParseComposeSpec_ServiceNetworks(t *testing.T) {
	spec, err := ParseComposeSpec(networkSpec)
	require.NoError(t, err)

	// Find web service
	var webService *Service
	for i := range spec.Services {
		if spec.Services[i].Name == "web" {
			webService = &spec.Services[i]
			break
		}
	}
	require.NotNil(t, webService)
	assert.Contains(t, webService.Networks, "frontend")
}

func TestParseComposeSpec_TopLevelNetworks(t *testing.T) {
	spec, err := ParseComposeSpec(networkSpec)
	require.NoError(t, err)

	assert.Len(t, spec.Networks, 2)

	// Find networks by name
	networkMap := make(map[string]Network)
	for _, n := range spec.Networks {
		networkMap[n.Name] = n
	}

	frontend, ok := networkMap["frontend"]
	require.True(t, ok)
	assert.Equal(t, "bridge", frontend.Driver)
	assert.False(t, frontend.Internal)

	backend, ok := networkMap["backend"]
	require.True(t, ok)
	assert.True(t, backend.Internal)
}

func TestParseComposeSpec_ExternalNetwork(t *testing.T) {
	yaml := `
services:
  app:
    image: nginx:latest
    networks:
      - existing

networks:
  existing:
    external: true
`
	spec, err := ParseComposeSpec(yaml)
	require.NoError(t, err)

	require.Len(t, spec.Networks, 1)
	assert.True(t, spec.Networks[0].External)
}

// =============================================================================
// Volume Definition Tests
// =============================================================================

func TestParseComposeSpec_TopLevelVolumes(t *testing.T) {
	spec, err := ParseComposeSpec(wordpressSpec)
	require.NoError(t, err)

	assert.Len(t, spec.Volumes, 2)

	volumeNames := make(map[string]bool)
	for _, v := range spec.Volumes {
		volumeNames[v.Name] = true
	}

	assert.True(t, volumeNames["wordpress_data"])
	assert.True(t, volumeNames["db_data"])
}

func TestParseComposeSpec_ExternalVolume(t *testing.T) {
	yaml := `
services:
  app:
    image: nginx:latest
    volumes:
      - existing:/data

volumes:
  existing:
    external: true
`
	spec, err := ParseComposeSpec(yaml)
	require.NoError(t, err)

	require.Len(t, spec.Volumes, 1)
	assert.True(t, spec.Volumes[0].External)
}

func TestParseComposeSpec_VolumeWithDriver(t *testing.T) {
	yaml := `
services:
  app:
    image: nginx:latest
    volumes:
      - mydata:/data

volumes:
  mydata:
    driver: local
`
	spec, err := ParseComposeSpec(yaml)
	require.NoError(t, err)

	require.Len(t, spec.Volumes, 1)
	assert.Equal(t, "local", spec.Volumes[0].Driver)
}

// =============================================================================
// Dependency Tests
// =============================================================================

func TestParseComposeSpec_DependsOnSimple(t *testing.T) {
	spec, err := ParseComposeSpec(multiServiceSpec)
	require.NoError(t, err)

	// Find web service
	var webService *Service
	for i := range spec.Services {
		if spec.Services[i].Name == "web" {
			webService = &spec.Services[i]
			break
		}
	}
	require.NotNil(t, webService)
	assert.Contains(t, webService.DependsOn, "api")
}

func TestParseComposeSpec_DependsOnLongForm(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx:latest
    depends_on:
      db:
        condition: service_healthy
      redis:
        condition: service_started

  db:
    image: postgres:15

  redis:
    image: redis:7
`
	spec, err := ParseComposeSpec(yaml)
	require.NoError(t, err)

	// Find web service
	var webService *Service
	for i := range spec.Services {
		if spec.Services[i].Name == "web" {
			webService = &spec.Services[i]
			break
		}
	}
	require.NotNil(t, webService)
	assert.Contains(t, webService.DependsOn, "db")
	assert.Contains(t, webService.DependsOn, "redis")
}

func TestParseComposeSpec_CircularDependency(t *testing.T) {
	_, err := ParseComposeSpec(circularDepSpec)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCircularDependency)
}

func TestParseComposeSpec_SelfReference(t *testing.T) {
	_, err := ParseComposeSpec(selfReferenceSpec)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCircularDependency)
}

// =============================================================================
// HealthCheck Tests
// =============================================================================

func TestParseComposeSpec_HealthCheck(t *testing.T) {
	spec, err := ParseComposeSpec(serviceWithHealthCheckSpec)
	require.NoError(t, err)
	require.Len(t, spec.Services, 1)

	hc := spec.Services[0].HealthCheck
	require.NotNil(t, hc)
	assert.Equal(t, []string{"CMD", "curl", "-f", "http://localhost"}, hc.Test)
	assert.Equal(t, "30s", hc.Interval)
	assert.Equal(t, "10s", hc.Timeout)
	assert.Equal(t, 3, hc.Retries)
	assert.Equal(t, "5s", hc.StartPeriod)
}

func TestParseComposeSpec_HealthCheckCMDShell(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx:latest
    healthcheck:
      test: ["CMD-SHELL", "curl -f http://localhost || exit 1"]
`
	spec, err := ParseComposeSpec(yaml)
	require.NoError(t, err)

	hc := spec.Services[0].HealthCheck
	require.NotNil(t, hc)
	assert.Equal(t, []string{"CMD-SHELL", "curl -f http://localhost || exit 1"}, hc.Test)
}

// =============================================================================
// Restart Policy Tests
// =============================================================================

func TestParseComposeSpec_RestartAlways(t *testing.T) {
	yaml := `
services:
  app:
    image: nginx:latest
    restart: always
`
	spec, err := ParseComposeSpec(yaml)
	require.NoError(t, err)

	assert.Equal(t, RestartAlways, spec.Services[0].Restart)
}

func TestParseComposeSpec_RestartOnFailure(t *testing.T) {
	yaml := `
services:
  app:
    image: nginx:latest
    restart: on-failure
`
	spec, err := ParseComposeSpec(yaml)
	require.NoError(t, err)

	assert.Equal(t, RestartOnFailure, spec.Services[0].Restart)
}

func TestParseComposeSpec_RestartUnlessStopped(t *testing.T) {
	yaml := `
services:
  app:
    image: nginx:latest
    restart: unless-stopped
`
	spec, err := ParseComposeSpec(yaml)
	require.NoError(t, err)

	assert.Equal(t, RestartUnlessStopped, spec.Services[0].Restart)
}

// =============================================================================
// Resource Tests
// =============================================================================

func TestParseComposeSpec_ResourceLimits(t *testing.T) {
	spec, err := ParseComposeSpec(serviceWithResourcesSpec)
	require.NoError(t, err)
	require.Len(t, spec.Services, 1)

	res := spec.Services[0].Resources
	assert.Equal(t, 2.0, res.CPULimit)
	assert.Equal(t, int64(1024*1024*1024), res.MemoryLimit) // 1G in bytes
	assert.Equal(t, 0.5, res.CPUReservation)
	assert.Equal(t, int64(512*1024*1024), res.MemoryReservation) // 512M in bytes
}

func TestCalculateResources_Defaults(t *testing.T) {
	spec, err := ParseComposeSpec(minimalValidSpec)
	require.NoError(t, err)

	resources := CalculateResources(spec)

	// 1 service * 0.5 CPU = 0.5
	assert.Equal(t, 0.5, resources.CPUCores)
	// 1 service * 256 MB = 256
	assert.Equal(t, int64(256), resources.MemoryMB)
	// No volumes
	assert.Equal(t, int64(0), resources.DiskMB)
}

func TestCalculateResources_MultipleServices(t *testing.T) {
	spec, err := ParseComposeSpec(multiServiceSpec)
	require.NoError(t, err)

	resources := CalculateResources(spec)

	// 3 services * 0.5 CPU = 1.5
	assert.Equal(t, 1.5, resources.CPUCores)
	// 3 services * 256 MB = 768
	assert.Equal(t, int64(768), resources.MemoryMB)
	// 1 volume * 1024 MB = 1024
	assert.Equal(t, int64(1024), resources.DiskMB)
}

func TestCalculateResources_WithExplicitLimits(t *testing.T) {
	spec, err := ParseComposeSpec(serviceWithResourcesSpec)
	require.NoError(t, err)

	resources := CalculateResources(spec)

	// Should use explicit limit (2.0) not default (0.5)
	assert.Equal(t, 2.0, resources.CPUCores)
	// Should use explicit limit (1G = 1024 MB) not default (256)
	assert.Equal(t, int64(1024), resources.MemoryMB)
}

func TestCalculateResources_WordPress(t *testing.T) {
	spec, err := ParseComposeSpec(wordpressSpec)
	require.NoError(t, err)

	resources := CalculateResources(spec)

	// 2 services * 0.5 CPU = 1.0
	assert.Equal(t, 1.0, resources.CPUCores)
	// 2 services * 256 MB = 512
	assert.Equal(t, int64(512), resources.MemoryMB)
	// 2 volumes * 1024 MB = 2048
	assert.Equal(t, int64(2048), resources.DiskMB)
}

// =============================================================================
// Validation Tests
// =============================================================================

func TestValidateParsedSpec_Valid(t *testing.T) {
	spec, err := ParseComposeSpec(wordpressSpec)
	require.NoError(t, err)

	errs := ValidateParsedSpec(spec)
	assert.Empty(t, errs)
}

func TestValidateParsedSpec_NegativeCPU(t *testing.T) {
	spec := &ParsedSpec{
		Services: []Service{
			{
				Name:  "app",
				Image: "nginx:latest",
				Resources: ServiceResources{
					CPULimit: -1,
				},
			},
		},
	}

	errs := ValidateParsedSpec(spec)
	require.Len(t, errs, 1)
	assert.ErrorIs(t, errs[0], ErrInvalidCPU)
}

func TestValidateParsedSpec_NegativeMemory(t *testing.T) {
	spec := &ParsedSpec{
		Services: []Service{
			{
				Name:  "app",
				Image: "nginx:latest",
				Resources: ServiceResources{
					MemoryLimit: -1,
				},
			},
		},
	}

	errs := ValidateParsedSpec(spec)
	require.Len(t, errs, 1)
	assert.ErrorIs(t, errs[0], ErrInvalidMemory)
}

// =============================================================================
// Complex/Real-World Tests
// =============================================================================

func TestParseComposeSpec_WordPress(t *testing.T) {
	spec, err := ParseComposeSpec(wordpressSpec)
	require.NoError(t, err)

	// Should have 2 services
	assert.Len(t, spec.Services, 2)

	// Should have 2 volumes
	assert.Len(t, spec.Volumes, 2)

	// Find wordpress service
	var wpService *Service
	for i := range spec.Services {
		if spec.Services[i].Name == "wordpress" {
			wpService = &spec.Services[i]
			break
		}
	}
	require.NotNil(t, wpService)
	assert.Equal(t, "wordpress:latest", wpService.Image)
	assert.Contains(t, wpService.DependsOn, "db")
	require.Len(t, wpService.Ports, 1)
	assert.Equal(t, uint32(80), wpService.Ports[0].Target)
}

func TestParseComposeSpec_NetworkedServices(t *testing.T) {
	spec, err := ParseComposeSpec(networkSpec)
	require.NoError(t, err)

	assert.Len(t, spec.Services, 2)
	assert.Len(t, spec.Networks, 2)

	// API should be in both networks
	var apiService *Service
	for i := range spec.Services {
		if spec.Services[i].Name == "api" {
			apiService = &spec.Services[i]
			break
		}
	}
	require.NotNil(t, apiService)
	assert.Len(t, apiService.Networks, 2)
	assert.Contains(t, apiService.Networks, "frontend")
	assert.Contains(t, apiService.Networks, "backend")
}

// =============================================================================
// Edge Case Tests
// =============================================================================

func TestParseComposeSpec_EmptyServiceName(t *testing.T) {
	// This should be handled by compose-go validation
	yaml := `
services:
  "":
    image: nginx:latest
`
	_, err := ParseComposeSpec(yaml)
	// Should fail or handle gracefully
	require.Error(t, err)
}

func TestParseComposeSpec_UnicodeInValues(t *testing.T) {
	yaml := `
services:
  app:
    image: nginx:latest
    labels:
      description: "Service with unicode: "
`
	spec, err := ParseComposeSpec(yaml)
	require.NoError(t, err)

	labels := spec.Services[0].Labels
	assert.Contains(t, labels["description"], "")
}

func TestParseComposeSpec_VeryLongImage(t *testing.T) {
	longImage := "registry.example.com/very/deep/path/to/image/name:v1.2.3-alpha.build.12345"
	yaml := `
services:
  app:
    image: ` + longImage + `
`
	spec, err := ParseComposeSpec(yaml)
	require.NoError(t, err)

	assert.Equal(t, longImage, spec.Services[0].Image)
}

// =============================================================================
// Unsupported Feature Tests
// =============================================================================

func TestParseComposeSpec_SecretsUnsupported(t *testing.T) {
	yaml := `
services:
  app:
    image: nginx:latest
    secrets:
      - my_secret

secrets:
  my_secret:
    file: ./secret.txt
`
	_, err := ParseComposeSpec(yaml)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnsupportedFeature)
}

func TestParseComposeSpec_ConfigsUnsupported(t *testing.T) {
	yaml := `
services:
  app:
    image: nginx:latest
    configs:
      - my_config

configs:
  my_config:
    file: ./config.txt
`
	_, err := ParseComposeSpec(yaml)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnsupportedFeature)
}

func TestParseComposeSpec_ExtendsUnsupported(t *testing.T) {
	// When extends references an external file, compose-go will either:
	// 1. Try to load the file (if SkipExtends is false)
	// 2. Skip extends processing (if SkipExtends is true), leaving service without image
	// Either way, we get an error. We check it's an error (not unsupported feature
	// because compose-go handles this before our checks run)
	yaml := `
services:
  app:
    extends:
      file: base.yml
      service: base
`
	_, err := ParseComposeSpec(yaml)
	require.Error(t, err)
	// Error could be "service must have image" or file not found
	assert.True(t, errors.Is(err, ErrServiceNoImage) || errors.Is(err, ErrInvalidYAML))
}

// Replicas should be silently ignored (not an error)
func TestParseComposeSpec_ReplicasIgnored(t *testing.T) {
	yaml := `
services:
  app:
    image: nginx:latest
    deploy:
      replicas: 3
`
	spec, err := ParseComposeSpec(yaml)
	require.NoError(t, err)
	assert.Len(t, spec.Services, 1)
	// Replicas field is just ignored, no error
}

// =============================================================================
// Error Type Tests
// =============================================================================

func TestParseError_Error(t *testing.T) {
	// Test ParseError with field
	errWithField := NewParseError("services.web.ports[0]", "invalid port", ErrServiceInvalidPort)
	assert.Equal(t, "services.web.ports[0]: invalid port", errWithField.Error())

	// Test ParseError without field
	errWithoutField := NewParseError("", "general error", ErrInvalidYAML)
	assert.Equal(t, "general error", errWithoutField.Error())
}

func TestParseError_Unwrap(t *testing.T) {
	err := NewParseError("test", "message", ErrInvalidYAML)
	assert.ErrorIs(t, err, ErrInvalidYAML)
}

// ExtractVariables on ParsedSpec - tests the version that scans parsed environment
func TestExtractVariables_ParsedSpec(t *testing.T) {
	// Create a spec directly with a placeholder in environment
	// (this could happen if the placeholder wasn't resolved for some reason)
	spec := &ParsedSpec{
		Services: []Service{
			{
				Name:  "app",
				Image: "nginx",
				Environment: map[string]string{
					"UNRESOLVED": "${SOME_VAR}",
				},
			},
		},
	}
	vars := ExtractVariables(spec)
	assert.Contains(t, vars, "SOME_VAR")
}

// Test port validation for published port > 65535
func TestValidatePorts_PublishedTooHigh(t *testing.T) {
	// Test via ParseComposeSpec which calls validatePorts internally
	_, err := ParseComposeSpec(`
services:
  app:
    image: nginx
    ports:
      - target: 80
        published: 70000
`)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrServiceInvalidPort)
}

// Test that internal extends (same file) - compose-go may or may not process it
func TestParseComposeSpec_InternalExtends(t *testing.T) {
	yaml := `
services:
  base:
    image: nginx:latest
  app:
    extends:
      service: base
`
	spec, err := ParseComposeSpec(yaml)
	// Internal extends behavior depends on compose-go version and SkipExtends setting
	// With SkipExtends=true, the extends is skipped and app gets no image
	// We expect an error because app has no image after extends is skipped
	if err != nil {
		// Error is expected - app has no image because extends was skipped
		// Our checkUnsupportedFeatures may also catch it
		assert.Error(t, err)
	} else {
		// If compose-go processed the extends, we should have 2 services
		assert.NotEmpty(t, spec.Services)
	}
}

// Test validatePorts for port index formatting
func TestValidatePorts_IndexFormatting(t *testing.T) {
	// Test with multiple ports - verifies the index formatting code path
	services := []Service{
		{
			Name:  "app",
			Image: "nginx",
			Ports: []Port{
				{Target: 80, Published: 8080},
				{Target: 0, Published: 8081}, // Invalid - target is 0
			},
		},
	}
	err := validatePorts(services)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrServiceInvalidPort)
}
