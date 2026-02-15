package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/artpar/hoster/internal/core/crypto"
	"github.com/artpar/hoster/internal/core/domain"
	"github.com/artpar/hoster/internal/core/proxy"
	"github.com/artpar/hoster/internal/shell/billing"
	"github.com/artpar/hoster/internal/shell/docker"
	"github.com/artpar/hoster/internal/shell/provider"
)

// RegisterHandlers registers all command handlers on the bus.
func RegisterHandlers(bus *Bus) {
	// Deployment lifecycle
	bus.Register("ScheduleDeployment", scheduleDeployment)
	bus.Register("StartDeployment", startDeployment)
	bus.Register("StopDeployment", stopDeployment)
	bus.Register("DeleteDeployment", deleteDeployment)
	bus.Register("DeploymentRunning", deploymentRunning)
	bus.Register("DeploymentFailed", deploymentFailed)

	// Cloud provision lifecycle
	bus.Register("DestroyInstance", destroyProvision)
}

// =============================================================================
// Deployment Handlers
// =============================================================================

// scheduleDeployment validates the deployer's selected node and transitions to starting.
func scheduleDeployment(ctx context.Context, deps *Deps, data map[string]any) error {
	store := deps.Store
	logger := deps.Logger
	nodePool := getNodePool(deps)

	refID, _ := data["reference_id"].(string)

	// The deployer must have selected a node at deploy time
	selectedNodeRef, _ := data["node_id"].(string)
	if selectedNodeRef == "" {
		return failDeployment(ctx, store, refID, "no node selected — please select a node when deploying")
	}

	// Look up the selected node and verify it's online
	selectedNode, err := store.Get(ctx, "nodes", selectedNodeRef)
	if err != nil {
		return failDeployment(ctx, store, refID, fmt.Sprintf("selected node %s not found", selectedNodeRef))
	}

	nodeStatus, _ := selectedNode["status"].(string)
	if nodeStatus != "online" {
		return failDeployment(ctx, store, refID, fmt.Sprintf("selected node %s is %s, not online", selectedNodeRef, nodeStatus))
	}

	// Allocate proxy port if needed
	proxyPort := toInt(data["proxy_port"])
	if proxyPort == 0 {
		usedPorts, err := getUsedProxyPorts(ctx, store, selectedNodeRef)
		if err != nil {
			logger.Warn("failed to get used proxy ports", "error", err)
		}
		port, err := proxy.AllocatePort(usedPorts, proxy.DefaultPortRange())
		if err != nil {
			return fmt.Errorf("allocate proxy port: %w", err)
		}
		proxyPort = port
	}

	// Generate auto domain if none set
	var domains any
	if d, ok := data["domains"]; ok {
		domains = d
	}
	baseDomain, _ := deps.Extra["base_domain"].(string)
	if domains == nil && baseDomain != "" {
		name, _ := data["name"].(string)
		autoDomain := domain.GenerateDomain(name, baseDomain)
		domainsJSON, _ := json.Marshal([]domain.Domain{autoDomain})
		domains = string(domainsJSON)
	}

	// Update deployment with node assignment, proxy port, domains
	updates := map[string]any{
		"node_id":    selectedNodeRef,
		"proxy_port": proxyPort,
	}
	if domains != nil {
		updates["domains"] = domains
	}
	store.Update(ctx, "deployments", refID, updates)

	// Verify node pool connectivity
	if nodePool != nil {
		if _, err := nodePool.GetClient(ctx, selectedNodeRef); err != nil {
			logger.Warn("node pool client unavailable, will retry on start", "node_id", selectedNodeRef, "error", err)
		}
	}

	// Transition to starting
	_, cmd, err := store.Transition(ctx, "deployments", refID, "starting")
	if err != nil {
		return fmt.Errorf("transition to starting: %w", err)
	}

	// Dispatch the StartDeployment command
	if cmd != "" {
		row, _ := store.Get(ctx, "deployments", refID)
		if row != nil {
			deps.Logger.Debug("dispatching start command", "deployment", refID)
			return startDeployment(ctx, deps, row)
		}
	}

	return nil
}

// startDeployment starts containers on the assigned node.
func startDeployment(ctx context.Context, deps *Deps, data map[string]any) error {
	store := deps.Store
	logger := deps.Logger
	nodePool := getNodePool(deps)

	refID, _ := data["reference_id"].(string)
	nodeID, _ := data["node_id"].(string)
	templateID := toInt(data["template_id"])
	configDir, _ := deps.Extra["config_dir"].(string)

	if nodePool == nil {
		return failDeployment(ctx, store, refID, "node pool not configured")
	}

	client, err := nodePool.GetClient(ctx, nodeID)
	if err != nil {
		return failDeployment(ctx, store, refID, fmt.Sprintf("failed to get docker client for node %s: %v", nodeID, err))
	}

	// Get template for compose spec
	tmpl, err := store.GetByID(ctx, "templates", templateID)
	if err != nil {
		return failDeployment(ctx, store, refID, fmt.Sprintf("template not found: %v", err))
	}

	composeSpec, _ := tmpl["compose_spec"].(string)
	if composeSpec == "" {
		return failDeployment(ctx, store, refID, "template has no compose spec")
	}

	// Build domain.Deployment for orchestrator
	depl := mapToDeployment(data)

	// Parse config files from template
	var configFiles []domain.ConfigFile
	if cfRaw, ok := tmpl["config_files"]; ok {
		if cfStr, ok := cfRaw.(string); ok && cfStr != "" {
			json.Unmarshal([]byte(cfStr), &configFiles)
		} else if cfParsed, ok := cfRaw.([]any); ok {
			b, _ := json.Marshal(cfParsed)
			json.Unmarshal(b, &configFiles)
		}
	}

	// Start via orchestrator
	orchestrator := docker.NewOrchestrator(client, logger, configDir, store)
	containers, err := orchestrator.StartDeployment(ctx, depl, composeSpec, configFiles)
	if err != nil {
		return failDeployment(ctx, store, refID, fmt.Sprintf("failed to start containers: %v", err))
	}

	// Transition to running
	containersJSON, _ := json.Marshal(containers)
	now := time.Now().UTC().Format(time.RFC3339)
	store.Update(ctx, "deployments", refID, map[string]any{
		"containers": string(containersJSON),
		"started_at": now,
	})

	_, _, err = store.Transition(ctx, "deployments", refID, "running")
	if err != nil {
		logger.Error("failed to transition to running", "deployment", refID, "error", err)
	} else {
		recordBillingEvent(ctx, store, data, domain.EventDeploymentStarted)
	}

	logger.Info("deployment started", "deployment", refID, "containers", len(containers))
	return nil
}

// stopDeployment stops containers on the assigned node.
func stopDeployment(ctx context.Context, deps *Deps, data map[string]any) error {
	store := deps.Store
	logger := deps.Logger
	nodePool := getNodePool(deps)

	refID, _ := data["reference_id"].(string)
	nodeID, _ := data["node_id"].(string)
	configDir, _ := deps.Extra["config_dir"].(string)

	if nodePool == nil {
		logger.Warn("node pool not configured, skipping container stop", "deployment", refID)
	} else if nodeID != "" {
		client, err := nodePool.GetClient(ctx, nodeID)
		if err != nil {
			logger.Warn("failed to get docker client, skipping container stop", "node_id", nodeID, "error", err)
		} else {
			depl := mapToDeployment(data)
			orchestrator := docker.NewOrchestrator(client, logger, configDir, nil)
			if err := orchestrator.StopDeployment(ctx, depl); err != nil {
				logger.Error("failed to stop containers", "deployment", refID, "error", err)
			}
		}
	}

	// Transition to stopped
	now := time.Now().UTC().Format(time.RFC3339)
	store.Update(ctx, "deployments", refID, map[string]any{
		"stopped_at": now,
	})
	_, _, err := store.Transition(ctx, "deployments", refID, "stopped")
	if err != nil {
		logger.Error("failed to transition to stopped", "deployment", refID, "error", err)
	} else {
		recordBillingEvent(ctx, store, data, domain.EventDeploymentStopped)
	}

	logger.Info("deployment stopped", "deployment", refID)
	return nil
}

// deleteDeployment removes all containers and transitions to deleted.
func deleteDeployment(ctx context.Context, deps *Deps, data map[string]any) error {
	store := deps.Store
	logger := deps.Logger
	nodePool := getNodePool(deps)

	refID, _ := data["reference_id"].(string)
	nodeID, _ := data["node_id"].(string)
	configDir, _ := deps.Extra["config_dir"].(string)

	if nodePool != nil && nodeID != "" {
		client, err := nodePool.GetClient(ctx, nodeID)
		if err != nil {
			logger.Warn("failed to get docker client, skipping container removal", "node_id", nodeID, "error", err)
		} else {
			depl := mapToDeployment(data)
			orchestrator := docker.NewOrchestrator(client, logger, configDir, nil)
			if err := orchestrator.RemoveDeployment(ctx, depl); err != nil {
				logger.Warn("failed to remove deployment containers", "deployment", refID, "error", err)
			}
		}
	}

	// Transition to deleted
	_, _, err := store.Transition(ctx, "deployments", refID, "deleted")
	if err != nil {
		logger.Error("failed to transition to deleted", "deployment", refID, "error", err)
	} else {
		recordBillingEvent(ctx, store, data, domain.EventDeploymentDeleted)
	}

	logger.Info("deployment deleted", "deployment", refID)
	return nil
}

// deploymentRunning is called when a deployment enters the running state.
func deploymentRunning(ctx context.Context, deps *Deps, data map[string]any) error {
	refID, _ := data["reference_id"].(string)
	deps.Logger.Info("deployment is running", "deployment", refID)
	return nil
}

// deploymentFailed is called when a deployment enters the failed state.
func deploymentFailed(ctx context.Context, deps *Deps, data map[string]any) error {
	refID, _ := data["reference_id"].(string)
	errMsg, _ := data["error_message"].(string)
	deps.Logger.Error("deployment failed", "deployment", refID, "error", errMsg)
	return nil
}

// =============================================================================
// Cloud Provision Handlers
// =============================================================================

// destroyProvision destroys the cloud instance and transitions to destroyed.
func destroyProvision(ctx context.Context, deps *Deps, data map[string]any) error {
	store := deps.Store
	logger := deps.Logger

	refID := strVal(data["reference_id"])
	instanceID := strVal(data["provider_instance_id"])

	if instanceID == "" {
		// No instance was ever created — just transition to destroyed
		_, _, err := store.Transition(ctx, "cloud_provisions", refID, "destroyed")
		if err != nil {
			logger.Error("failed to transition to destroyed", "provision", refID, "error", err)
		}
		return nil
	}

	providerType := strVal(data["provider"])

	// Look up credential by FK integer ID
	credID := toInt(data["credential_id"])
	if credID == 0 {
		return failProvision(ctx, store, refID, "no credential_id on provision, cannot destroy cloud resource")
	}

	cred, err := store.GetByID(ctx, "cloud_credentials", credID)
	if err != nil {
		return failProvision(ctx, store, refID, fmt.Sprintf("failed to look up credential %d: %v", credID, err))
	}

	// Decrypt credentials
	credEncrypted := cred["credentials"]
	var credBytes []byte
	switch v := credEncrypted.(type) {
	case []byte:
		credBytes = v
	case string:
		credBytes = []byte(v)
	}

	encryptionKey, _ := deps.Extra["encryption_key"].([]byte)
	decrypted, err := crypto.Decrypt(credBytes, encryptionKey)
	if err != nil {
		return failProvision(ctx, store, refID, fmt.Sprintf("failed to decrypt credentials: %v", err))
	}

	prov, err := provider.NewProvider(providerType, decrypted, logger)
	if err != nil {
		return failProvision(ctx, store, refID, fmt.Sprintf("failed to create provider: %v", err))
	}

	destroyReq := provider.DestroyRequest{
		ProviderInstanceID: instanceID,
		InstanceName:       strVal(data["instance_name"]),
		Region:             strVal(data["region"]),
	}
	if err := prov.DestroyInstance(ctx, destroyReq); err != nil {
		return failProvision(ctx, store, refID, fmt.Sprintf("destroy instance failed: %v", err))
	}

	// Transition to destroyed — only reached when the cloud API call succeeded
	_, _, err = store.Transition(ctx, "cloud_provisions", refID, "destroyed")
	if err != nil {
		logger.Error("failed to transition to destroyed", "provision", refID, "error", err)
	}

	// Delete associated node if one was created
	nodeRefID := strVal(data["node_id"])
	if nodeRefID != "" {
		if err := store.Delete(ctx, "nodes", nodeRefID); err != nil {
			logger.Warn("failed to delete associated node", "provision", refID, "node", nodeRefID, "error", err)
		}
	}

	logger.Info("provision destroyed", "provision", refID, "instance_id", instanceID)
	return nil
}

// =============================================================================
// Helpers
// =============================================================================

func failDeployment(ctx context.Context, store *Store, refID, reason string) error {
	store.Update(ctx, "deployments", refID, map[string]any{
		"error_message": reason,
	})
	store.Transition(ctx, "deployments", refID, "failed")
	return fmt.Errorf("%s: %s", refID, reason)
}

func failProvision(ctx context.Context, store *Store, refID, reason string) error {
	store.Update(ctx, "cloud_provisions", refID, map[string]any{
		"error_message": reason,
	})
	store.Transition(ctx, "cloud_provisions", refID, "failed")
	return fmt.Errorf("%s: %s", refID, reason)
}

func getNodePool(deps *Deps) *docker.NodePool {
	if np, ok := deps.Extra["node_pool"].(*docker.NodePool); ok {
		return np
	}
	return nil
}

func getUsedProxyPorts(ctx context.Context, store *Store, nodeID string) ([]int, error) {
	rows, err := store.RawQuery(ctx,
		"SELECT proxy_port FROM deployments WHERE node_id = ? AND status NOT IN ('deleted', 'stopped') AND proxy_port IS NOT NULL",
		nodeID)
	if err != nil {
		return nil, err
	}
	var ports []int
	for _, row := range rows {
		if p := toInt(row["proxy_port"]); p > 0 {
			ports = append(ports, p)
		}
	}
	return ports, nil
}

func recordBillingEvent(ctx context.Context, store *Store, data map[string]any, eventType domain.EventType) {
	refID, _ := data["reference_id"].(string)
	customerID := toInt(data["customer_id"])
	if customerID == 0 || refID == "" {
		return
	}
	billing.RecordEvent(ctx, store, customerID, eventType, refID, "deployment", nil)
}

func toInt(v any) int {
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case json.Number:
		if i, err := val.Int64(); err == nil {
			return int(i)
		}
	}
	return 0
}

