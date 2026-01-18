package compose

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/artpar/hoster/internal/core/domain"
	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/types"
	"gopkg.in/yaml.v3"
)

// silence unused import warnings for now
var (
	_ = context.Background
	_ = strconv.Atoi
	_ = yaml.Unmarshal
)

// =============================================================================
// Resource Defaults (per specs/domain/template.md)
// =============================================================================

const (
	// DefaultCPUPerService is the default CPU cores per service.
	DefaultCPUPerService = 0.5
	// DefaultMemoryPerService is the default memory per service in bytes.
	DefaultMemoryPerService = 256 * 1024 * 1024 // 256 MB
	// DefaultDiskPerVolume is the default disk per volume in MB.
	DefaultDiskPerVolume = 1024 // 1024 MB
)

// =============================================================================
// Parser Functions
// =============================================================================

// ParseComposeSpec parses Docker Compose YAML into a ParsedSpec.
// This is a pure function - no I/O, no side effects.
// Input: raw YAML string
// Output: ParsedSpec struct or error
func ParseComposeSpec(yamlContent string) (*ParsedSpec, error) {
	// Input validation
	if strings.TrimSpace(yamlContent) == "" {
		return nil, ErrEmptyInput
	}

	// Parse using compose-go
	project, err := loadComposeSpec(yamlContent)
	if err != nil {
		return nil, err
	}

	// Check for unsupported features first
	if err := checkUnsupportedFeatures(project); err != nil {
		return nil, err
	}

	// Validate required fields
	if len(project.Services) == 0 {
		return nil, ErrNoServices
	}

	// Convert to Hoster types
	spec := &ParsedSpec{
		Services: make([]Service, 0, len(project.Services)),
		Networks: make([]Network, 0, len(project.Networks)),
		Volumes:  make([]Volume, 0, len(project.Volumes)),
	}

	// Convert services
	for _, svc := range project.Services {
		converted, err := convertService(svc)
		if err != nil {
			return nil, err
		}
		spec.Services = append(spec.Services, converted)
	}

	// Validate no circular dependencies
	if err := detectCircularDependencies(spec.Services); err != nil {
		return nil, err
	}

	// Validate ports
	if err := validatePorts(spec.Services); err != nil {
		return nil, err
	}

	// Convert networks
	for name, net := range project.Networks {
		spec.Networks = append(spec.Networks, convertNetwork(name, net))
	}

	// Convert volumes
	for name, vol := range project.Volumes {
		spec.Volumes = append(spec.Volumes, convertVolume(name, vol))
	}

	return spec, nil
}

// loadComposeSpec loads a compose spec using compose-go
func loadComposeSpec(yamlContent string) (*types.Project, error) {
	// Parse YAML into a map first
	var dict map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlContent), &dict); err != nil {
		return nil, NewParseError("", "invalid YAML syntax", ErrInvalidYAML)
	}

	// Check if it's a valid object
	if dict == nil {
		return nil, NewParseError("", "invalid YAML syntax", ErrInvalidYAML)
	}

	// Load the project
	project, err := loader.LoadWithContext(context.Background(), types.ConfigDetails{
		ConfigFiles: []types.ConfigFile{
			{
				Content: []byte(yamlContent),
				Config:  dict,
			},
		},
	}, func(opts *loader.Options) {
		opts.SetProjectName("hoster-temp", false)
		opts.SkipValidation = false
		opts.SkipInterpolation = false // Enable interpolation for proper type parsing
		// Don't resolve paths since we're in-memory
		opts.SkipNormalization = true
		opts.SkipExtends = true // Don't try to load external files
	})
	if err != nil {
		errStr := err.Error()
		// Check for circular dependency
		if strings.Contains(errStr, "dependency cycle detected") {
			return nil, NewParseError("", "circular dependency detected", ErrCircularDependency)
		}
		// Check if it's a service validation error
		if strings.Contains(errStr, "image") && strings.Contains(errStr, "build") {
			return nil, NewParseError("", "service must have image or build", ErrServiceNoImage)
		}
		return nil, NewParseError("", errStr, ErrInvalidYAML)
	}

	return project, nil
}

// checkUnsupportedFeatures checks for features we don't support
func checkUnsupportedFeatures(project *types.Project) error {
	// Check for secrets
	if len(project.Secrets) > 0 {
		return NewParseError("secrets", "secrets are not supported", ErrUnsupportedFeature)
	}

	// Check for configs
	if len(project.Configs) > 0 {
		return NewParseError("configs", "configs are not supported", ErrUnsupportedFeature)
	}

	// Check for extends in services
	for _, svc := range project.Services {
		if svc.Extends != nil && svc.Extends.File != "" {
			return NewParseError("services."+svc.Name+".extends", "extends is not supported", ErrUnsupportedFeature)
		}
	}

	return nil
}

// convertService converts a compose-go service to our Service type
func convertService(svc types.ServiceConfig) (Service, error) {
	service := Service{
		Name:        svc.Name,
		Image:       svc.Image,
		Command:     svc.Command,
		Entrypoint:  svc.Entrypoint,
		Environment: make(map[string]string),
		Labels:      make(map[string]string),
		Networks:    make([]string, 0),
		DependsOn:   make([]string, 0),
	}

	// Build config
	if svc.Build != nil {
		service.Build = &BuildConfig{
			Context:    svc.Build.Context,
			Dockerfile: svc.Build.Dockerfile,
		}
	}

	// Validate image or build
	if service.Image == "" && service.Build == nil {
		return Service{}, NewParseError("services."+svc.Name, "service must have image or build", ErrServiceNoImage)
	}

	// Ports
	for _, p := range svc.Ports {
		var published uint32
		if p.Published != "" {
			pub, err := strconv.ParseUint(p.Published, 10, 32)
			if err == nil {
				published = uint32(pub)
			}
		}
		port := Port{
			Target:    p.Target,
			Published: published,
			Protocol:  p.Protocol,
			HostIP:    p.HostIP,
		}
		service.Ports = append(service.Ports, port)
	}

	// Environment
	for k, v := range svc.Environment {
		if v != nil {
			service.Environment[k] = *v
		}
	}

	// Volumes
	for _, v := range svc.Volumes {
		mount := VolumeMount{
			Source:   v.Source,
			Target:   v.Target,
			ReadOnly: v.ReadOnly,
		}
		switch v.Type {
		case "bind":
			mount.Type = VolumeMountTypeBind
		case "volume":
			mount.Type = VolumeMountTypeVolume
		case "tmpfs":
			mount.Type = VolumeMountTypeTmpfs
		default:
			// Infer type from source
			if strings.HasPrefix(v.Source, "./") || strings.HasPrefix(v.Source, "/") || strings.HasPrefix(v.Source, "~") {
				mount.Type = VolumeMountTypeBind
			} else {
				mount.Type = VolumeMountTypeVolume
			}
		}
		service.Volumes = append(service.Volumes, mount)
	}

	// Networks
	for net := range svc.Networks {
		service.Networks = append(service.Networks, net)
	}

	// DependsOn
	for dep := range svc.DependsOn {
		service.DependsOn = append(service.DependsOn, dep)
	}

	// Restart policy
	service.Restart = RestartPolicy(svc.Restart)

	// Labels
	for k, v := range svc.Labels {
		service.Labels[k] = v
	}

	// HealthCheck
	if svc.HealthCheck != nil && !svc.HealthCheck.Disable {
		service.HealthCheck = &HealthCheck{
			Test: svc.HealthCheck.Test,
		}
		if svc.HealthCheck.Retries != nil {
			service.HealthCheck.Retries = int(*svc.HealthCheck.Retries)
		}
		if svc.HealthCheck.Interval != nil {
			service.HealthCheck.Interval = svc.HealthCheck.Interval.String()
		}
		if svc.HealthCheck.Timeout != nil {
			service.HealthCheck.Timeout = svc.HealthCheck.Timeout.String()
		}
		if svc.HealthCheck.StartPeriod != nil {
			service.HealthCheck.StartPeriod = svc.HealthCheck.StartPeriod.String()
		}
	}

	// Resources
	// Note: compose-go's NanoCPUs is misnamed - it's actually the CPU count as float32
	if svc.Deploy != nil && svc.Deploy.Resources.Limits != nil {
		limits := svc.Deploy.Resources.Limits
		service.Resources.CPULimit = float64(limits.NanoCPUs)
		service.Resources.MemoryLimit = int64(limits.MemoryBytes)
	}
	if svc.Deploy != nil && svc.Deploy.Resources.Reservations != nil {
		reservations := svc.Deploy.Resources.Reservations
		service.Resources.CPUReservation = float64(reservations.NanoCPUs)
		service.Resources.MemoryReservation = int64(reservations.MemoryBytes)
	}

	return service, nil
}

// convertNetwork converts a compose-go network to our Network type
func convertNetwork(name string, net types.NetworkConfig) Network {
	return Network{
		Name:       name,
		Driver:     net.Driver,
		External:   bool(net.External),
		Internal:   net.Internal,
		Attachable: net.Attachable,
		Labels:     net.Labels,
	}
}

// convertVolume converts a compose-go volume to our Volume type
func convertVolume(name string, vol types.VolumeConfig) Volume {
	return Volume{
		Name:     name,
		Driver:   vol.Driver,
		External: bool(vol.External),
		Labels:   vol.Labels,
	}
}

// detectCircularDependencies detects circular dependencies in service dependencies
func detectCircularDependencies(services []Service) error {
	// Build adjacency list
	deps := make(map[string][]string)
	for _, svc := range services {
		deps[svc.Name] = svc.DependsOn
	}

	// Track visited and recursion stack for DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(node string) bool
	hasCycle = func(node string) bool {
		visited[node] = true
		recStack[node] = true

		for _, dep := range deps[node] {
			// Self-reference
			if dep == node {
				return true
			}
			if !visited[dep] {
				if hasCycle(dep) {
					return true
				}
			} else if recStack[dep] {
				return true
			}
		}

		recStack[node] = false
		return false
	}

	for _, svc := range services {
		if !visited[svc.Name] {
			if hasCycle(svc.Name) {
				return ErrCircularDependency
			}
		}
	}

	return nil
}

// validatePorts validates all port configurations
func validatePorts(services []Service) error {
	for _, svc := range services {
		for i, port := range svc.Ports {
			if port.Target == 0 {
				return NewParseError(
					"services."+svc.Name+".ports["+string(rune('0'+i))+"]",
					"target port cannot be 0",
					ErrServiceInvalidPort,
				)
			}
			if port.Target > 65535 {
				return NewParseError(
					"services."+svc.Name+".ports["+string(rune('0'+i))+"]",
					"target port must be <= 65535",
					ErrServiceInvalidPort,
				)
			}
			if port.Published > 65535 {
				return NewParseError(
					"services."+svc.Name+".ports["+string(rune('0'+i))+"]",
					"published port must be <= 65535",
					ErrServiceInvalidPort,
				)
			}
		}
	}
	return nil
}

// =============================================================================
// Resource Calculation
// =============================================================================

// CalculateResources calculates total resource requirements from a parsed spec.
// Uses defaults when resources are not explicitly specified.
// Per service: 0.5 CPU cores, 256MB memory
// Per volume: 1024MB disk
func CalculateResources(spec *ParsedSpec) domain.Resources {
	var totalCPU float64
	var totalMemoryBytes int64
	var totalDiskMB int64

	for _, svc := range spec.Services {
		// Use explicit limits if set, otherwise defaults
		if svc.Resources.CPULimit > 0 {
			totalCPU += svc.Resources.CPULimit
		} else {
			totalCPU += DefaultCPUPerService
		}

		if svc.Resources.MemoryLimit > 0 {
			totalMemoryBytes += svc.Resources.MemoryLimit
		} else {
			totalMemoryBytes += DefaultMemoryPerService
		}
	}

	// Add disk for each named volume
	totalDiskMB = int64(len(spec.Volumes)) * DefaultDiskPerVolume

	return domain.Resources{
		CPUCores: totalCPU,
		MemoryMB: totalMemoryBytes / (1024 * 1024),
		DiskMB:   totalDiskMB,
	}
}

// =============================================================================
// Variable Extraction
// =============================================================================

// variablePlaceholderRegex matches ${VAR_NAME} or ${VAR_NAME:-default}
var variablePlaceholderRegex = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(?::-[^}]*)?\}`)

// ExtractVariables extracts environment variable placeholders (${VAR_NAME}) from spec.
// Returns unique variable names without the ${} wrapper.
// Note: This works on the resolved environment values in ParsedSpec.
func ExtractVariables(spec *ParsedSpec) []string {
	seen := make(map[string]bool)
	var vars []string

	for _, svc := range spec.Services {
		for _, val := range svc.Environment {
			matches := variablePlaceholderRegex.FindAllStringSubmatch(val, -1)
			for _, match := range matches {
				if len(match) >= 2 {
					varName := match[1]
					if !seen[varName] {
						seen[varName] = true
						vars = append(vars, varName)
					}
				}
			}
		}
	}

	return vars
}

// ExtractVariablesFromYAML extracts environment variable placeholders from raw YAML content.
// This extracts variable names before compose-go interpolates them.
// Returns unique variable names without the ${} wrapper.
func ExtractVariablesFromYAML(yamlContent string) []string {
	seen := make(map[string]bool)
	var vars []string

	matches := variablePlaceholderRegex.FindAllStringSubmatch(yamlContent, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			varName := match[1]
			if !seen[varName] {
				seen[varName] = true
				vars = append(vars, varName)
			}
		}
	}

	return vars
}

// =============================================================================
// Validation
// =============================================================================

// ValidateParsedSpec performs semantic validation on a parsed spec.
// Returns all validation errors found.
func ValidateParsedSpec(spec *ParsedSpec) []error {
	var errs []error

	for _, svc := range spec.Services {
		// Validate CPU
		if svc.Resources.CPULimit < 0 {
			errs = append(errs, NewParseError(
				"services."+svc.Name+".resources.cpu_limit",
				"CPU limit cannot be negative",
				ErrInvalidCPU,
			))
		}
		if svc.Resources.CPUReservation < 0 {
			errs = append(errs, NewParseError(
				"services."+svc.Name+".resources.cpu_reservation",
				"CPU reservation cannot be negative",
				ErrInvalidCPU,
			))
		}

		// Validate memory
		if svc.Resources.MemoryLimit < 0 {
			errs = append(errs, NewParseError(
				"services."+svc.Name+".resources.memory_limit",
				"Memory limit cannot be negative",
				ErrInvalidMemory,
			))
		}
		if svc.Resources.MemoryReservation < 0 {
			errs = append(errs, NewParseError(
				"services."+svc.Name+".resources.memory_reservation",
				"Memory reservation cannot be negative",
				ErrInvalidMemory,
			))
		}
	}

	return errs
}
