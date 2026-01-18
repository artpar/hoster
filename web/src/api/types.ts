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
  price_cents: number;
  icon_url?: string;
  documentation_url?: string;
}

export interface CreateDeploymentRequest {
  name: string;
  template_id: string;
  environment_variables?: Record<string, string>;
  custom_domain?: string;
  config_overrides?: Record<string, string>;
}
