// JSON:API Response Types
// Following JSON:API specification (https://jsonapi.org)

export interface JsonApiResource<T extends string, A> {
  type: T;
  id: string;
  attributes: A;
  relationships?: Record<string, JsonApiRelationship>;
  links?: JsonApiLinks;
}

export interface JsonApiRelationship {
  data: JsonApiResourceIdentifier | JsonApiResourceIdentifier[] | null;
  links?: JsonApiLinks;
}

export interface JsonApiResourceIdentifier {
  type: string;
  id: string;
}

export interface JsonApiLinks {
  self?: string;
  related?: string;
  first?: string;
  last?: string;
  prev?: string;
  next?: string;
}

export interface JsonApiError {
  id?: string;
  status?: string;
  code?: string;
  title?: string;
  detail?: string;
  source?: {
    pointer?: string;
    parameter?: string;
  };
}

export interface JsonApiResponse<T> {
  data: T;
  included?: JsonApiResource<string, unknown>[];
  links?: JsonApiLinks;
  meta?: Record<string, unknown>;
}

export interface JsonApiErrorResponse {
  errors: JsonApiError[];
}

// Template Types
export interface TemplateAttributes {
  name: string;
  slug: string;
  description?: string;
  version: string;
  compose_spec: string;
  variables?: Variable[];
  config_files?: ConfigFile[];
  resource_requirements: ResourceRequirements;
  price_monthly_cents: number;
  category?: string;
  tags?: string[];
  published: boolean;
  creator_id: string;
  created_at: string;
  updated_at: string;
}

export interface Variable {
  name: string;
  description?: string;
  label?: string;
  type: string;
  default?: string;
  required?: boolean;
  options?: string[];
  validation?: string;
}

export interface ConfigFile {
  name: string;
  path: string;
  content: string;
  mode?: string;
}

export interface ResourceRequirements {
  cpu_cores: number;
  memory_mb: number;
  disk_mb: number;
}

export type Template = JsonApiResource<'templates', TemplateAttributes>;

// Deployment Types
export interface DeploymentAttributes {
  name: string;
  status: 'pending' | 'scheduled' | 'starting' | 'running' | 'stopping' | 'stopped' | 'deleting' | 'deleted' | 'failed';
  domain?: string;
  environment_variables?: Record<string, string>;
  customer_id: string;
  template_id: string;
  created_at: string;
  updated_at: string;
  started_at?: string;
  stopped_at?: string;
  error_message?: string;
  containers?: ContainerInfo[];
}

export interface ContainerInfo {
  id: string;
  service_name: string;
  image: string;
  status: string;
  ports?: PortMapping[];
}

export interface PortMapping {
  host_port: number;
  container_port: number;
  protocol: string;
}

export type Deployment = JsonApiResource<'deployments', DeploymentAttributes>;

// Monitoring Types
export interface HealthAttributes {
  status: 'healthy' | 'unhealthy' | 'degraded' | 'unknown';
  containers: ContainerHealth[];
  checked_at: string;
}

export interface ContainerHealth {
  name: string;
  status: string;
  health: 'healthy' | 'unhealthy' | 'degraded' | 'unknown';
  started_at?: string;
  restarts: number;
}

export interface StatsAttributes {
  containers: ContainerStats[];
  collected_at: string;
}

export interface ContainerStats {
  name: string;
  cpu_percent: number;
  memory_usage_bytes: number;
  memory_limit_bytes: number;
  memory_percent: number;
  network_rx_bytes: number;
  network_tx_bytes: number;
  block_read_bytes: number;
  block_write_bytes: number;
  pids: number;
}

export interface LogsAttributes {
  logs: LogEntry[];
}

export interface LogEntry {
  container: string;
  timestamp: string;
  stream: 'stdout' | 'stderr';
  message: string;
}

export interface EventsAttributes {
  events: ContainerEvent[];
}

export interface ContainerEvent {
  id: string;
  type: string;
  container: string;
  message: string;
  timestamp: string;
}

// Request Types
export interface CreateTemplateRequest {
  name: string;
  description: string;
  version: string;
  compose_spec: string;
  price_monthly_cents: number;
  icon_url?: string;
  documentation_url?: string;
}

export interface CreateDeploymentRequest {
  name: string;
  template_id: string;
  environment_variables?: Record<string, string>;
  custom_domain?: string;
  config_overrides?: Record<string, string>;
  node_id?: string;
}

// Node Types
export type NodeStatus = 'online' | 'offline' | 'maintenance';

export interface NodeCapacity {
  cpu_cores: number;
  memory_mb: number;
  disk_mb: number;
  cpu_used: number;
  memory_used_mb: number;
  disk_used_mb: number;
}

export interface NodeAttributes {
  name: string;
  ssh_host: string;
  ssh_port: number;
  ssh_user: string;
  ssh_key_id?: string;
  docker_socket: string;
  status: NodeStatus;
  public: boolean;
  capabilities: string[];
  capacity: NodeCapacity;
  location?: string;
  last_health_check?: string;
  error_message?: string;
  provider_type?: string;
  provision_id?: string;
  creator_id: string;
  created_at: string;
  updated_at: string;
}

export type Node = JsonApiResource<'nodes', NodeAttributes>;

export interface CreateNodeRequest {
  name: string;
  ssh_host: string;
  ssh_port?: number;
  ssh_user: string;
  ssh_key_id?: string;
  docker_socket?: string;
  public?: boolean;
  capabilities?: string[];
  location?: string;
  base_domain?: string;
}

export interface UpdateNodeRequest {
  name?: string;
  ssh_host?: string;
  ssh_port?: number;
  ssh_user?: string;
  ssh_key_id?: string;
  docker_socket?: string;
  public?: boolean;
  capabilities?: string[];
  location?: string;
  base_domain?: string;
}

// SSH Key Types
export interface SSHKeyAttributes {
  name: string;
  fingerprint: string;
  creator_id: string;
  created_at: string;
}

export type SSHKey = JsonApiResource<'ssh_keys', SSHKeyAttributes>;

export interface CreateSSHKeyRequest {
  name: string;
  private_key: string;
}

// Cloud Credential Types
export interface CloudCredentialAttributes {
  name: string;
  provider: 'aws' | 'digitalocean' | 'hetzner';
  default_region?: string;
  creator_id: string;
  created_at: string;
  updated_at: string;
  credentials?: string; // write-only
}

export type CloudCredential = JsonApiResource<'cloud_credentials', CloudCredentialAttributes>;

export interface CreateCloudCredentialRequest {
  name: string;
  provider: 'aws' | 'digitalocean' | 'hetzner';
  credentials: string;
  default_region?: string;
}

// Cloud Provision Types
export type ProvisionStatus = 'pending' | 'creating' | 'configuring' | 'ready' | 'failed' | 'destroying' | 'destroyed';

export interface CloudProvisionAttributes {
  creator_id: string;
  credential_id: string;
  provider: string;
  status: ProvisionStatus;
  instance_name: string;
  region: string;
  size: string;
  provider_instance_id?: string;
  public_ip?: string;
  node_id?: string;
  ssh_key_id?: string;
  current_step?: string;
  error_message?: string;
  created_at: string;
  updated_at: string;
  completed_at?: string;
}

export type CloudProvision = JsonApiResource<'cloud_provisions', CloudProvisionAttributes>;

export interface CreateCloudProvisionRequest {
  credential_id: string;
  instance_name: string;
  region: string;
  size: string;
}

// Provider catalog types (returned by custom actions)
export interface ProviderRegion {
  id: string;
  name: string;
  available: boolean;
}

export interface ProviderInstanceSize {
  id: string;
  name: string;
  cpu_cores: number;
  memory_mb: number;
  disk_gb: number;
  price_hourly: number;
}
