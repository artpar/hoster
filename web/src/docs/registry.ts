import type { StatusDoc, MetricDoc, PageDoc, EventDoc, DocEntry } from './types';

// --- DEPLOYMENT STATUSES ---
export const deploymentStatuses: Record<
  'pending' | 'scheduled' | 'starting' | 'running' | 'stopping' | 'stopped' | 'deleting' | 'deleted' | 'failed',
  StatusDoc
> = {
  pending:   { label: 'Pending',   style: 'bg-blue-100 text-blue-800',     description: 'Waiting to be assigned to an available server node.' },
  scheduled: { label: 'Scheduled', style: 'bg-blue-100 text-blue-800',     description: 'A server has been assigned. Container images are being pulled.' },
  starting:  { label: 'Starting',  style: 'bg-yellow-100 text-yellow-800', description: 'Containers are being created and started. Usually takes 10-30 seconds.' },
  running:   { label: 'Running',   style: 'bg-green-100 text-green-800',   description: 'Application is live and accessible at its domain URL.' },
  stopping:  { label: 'Stopping',  style: 'bg-yellow-100 text-yellow-800', description: 'Containers are gracefully shutting down. Data is preserved.' },
  stopped:   { label: 'Stopped',   style: 'bg-gray-100 text-gray-800',     description: 'Paused. No resources consumed. Can be restarted anytime.' },
  deleting:  { label: 'Deleting',  style: 'bg-red-100 text-red-800',       description: 'All containers and data are being permanently removed.' },
  deleted:   { label: 'Deleted',   style: 'bg-gray-100 text-gray-800',     description: 'Permanently removed. Cannot be recovered.' },
  failed:    { label: 'Failed',    style: 'bg-red-100 text-red-800',       description: 'Something went wrong. Check the error message for details.' },
};

// --- HEALTH STATUSES ---
export const healthStatuses: Record<'healthy' | 'unhealthy' | 'degraded' | 'unknown', StatusDoc> = {
  healthy:   { label: 'Healthy',   style: 'bg-green-100 text-green-800',   description: 'All health checks passing.' },
  unhealthy: { label: 'Unhealthy', style: 'bg-red-100 text-red-800',       description: 'One or more health checks failing.' },
  degraded:  { label: 'Degraded',  style: 'bg-yellow-100 text-yellow-800', description: 'Partially operational. Some services may be impaired.' },
  unknown:   { label: 'Unknown',   style: 'bg-gray-100 text-gray-800',     description: 'Health status could not be determined.' },
};

// --- TEMPLATE STATUSES ---
export const templateStatuses: Record<'draft' | 'published' | 'deprecated', StatusDoc> = {
  draft:      { label: 'Draft',      style: 'bg-gray-100 text-gray-800',     description: 'Only visible to the creator. Not available in marketplace.' },
  published:  { label: 'Published',  style: 'bg-green-100 text-green-800',   description: 'Visible in marketplace. Users can deploy this template.' },
  deprecated: { label: 'Deprecated', style: 'bg-yellow-100 text-yellow-800', description: 'Still accessible but no longer recommended for new deployments.' },
};

// --- NODE STATUSES ---
export const nodeStatuses: Record<'online' | 'offline' | 'maintenance', StatusDoc> = {
  online:      { label: 'Online',      style: 'bg-green-100 text-green-800',   description: 'Connected and accepting new deployments.' },
  offline:     { label: 'Offline',     style: 'bg-red-100 text-red-800',       description: 'Cannot be reached. Existing deployments may be affected.' },
  maintenance: { label: 'Maintenance', style: 'bg-yellow-100 text-yellow-800', description: 'No new deployments will be scheduled. Existing deployments keep running.' },
};

// Merged lookup for StatusBadge (all status types in one map)
export const allStatuses: Record<string, StatusDoc> = {
  ...deploymentStatuses,
  ...healthStatuses,
  ...templateStatuses,
  ...nodeStatuses,
};

// --- CONTAINER METRICS ---
export const containerMetrics: Record<
  'cpu_percent' | 'memory_percent' | 'memory_usage' | 'network_rx' | 'network_tx' | 'block_read' | 'block_write' | 'pids',
  MetricDoc
> = {
  cpu_percent:    { label: 'CPU %',       unit: '%',     description: 'Percentage of host CPU cores being used by this container.' },
  memory_percent: { label: 'Memory %',    unit: '%',     description: 'Percentage of the container memory limit currently in use.' },
  memory_usage:   { label: 'Memory',      unit: 'bytes', description: 'Current memory usage vs. the container limit.' },
  network_rx:     { label: 'Network RX',  unit: 'bytes', description: 'Total bytes received since container started.' },
  network_tx:     { label: 'Network TX',  unit: 'bytes', description: 'Total bytes sent since container started.' },
  block_read:     { label: 'Block Read',  unit: 'bytes', description: 'Total bytes read from disk since container started.' },
  block_write:    { label: 'Block Write', unit: 'bytes', description: 'Total bytes written to disk since container started.' },
  pids:           { label: 'PIDs',        unit: 'count', description: 'Number of active processes inside the container.' },
};

// --- EVENT TYPES ---
export const eventTypes: Record<
  'image_pulling' | 'image_pulled' | 'container_created' | 'container_started' | 'container_stopped' | 'container_restarted' | 'container_died' | 'container_oom' | 'health_unhealthy' | 'health_healthy',
  EventDoc
> = {
  image_pulling:       { label: 'Pulling',   severity: 'info',    description: 'Docker image is being downloaded from the registry.' },
  image_pulled:        { label: 'Pulled',    severity: 'success', description: 'Docker image downloaded successfully.' },
  container_created:   { label: 'Created',   severity: 'info',    description: 'Container was built from its Docker image.' },
  container_started:   { label: 'Started',   severity: 'success', description: 'Container began running.' },
  container_stopped:   { label: 'Stopped',   severity: 'info',    description: 'Graceful shutdown completed.' },
  container_restarted: { label: 'Restarted', severity: 'warning', description: 'Container was restarted (may indicate instability).' },
  container_died:      { label: 'Died',      severity: 'error',   description: 'Container crashed unexpectedly.' },
  container_oom:       { label: 'OOM Kill',  severity: 'error',   description: 'Killed by the system — ran out of memory.' },
  health_unhealthy:    { label: 'Unhealthy', severity: 'error',   description: 'Health check failed.' },
  health_healthy:      { label: 'Healthy',   severity: 'success', description: 'Health check passed.' },
};

// --- LOG STREAMS ---
export const logStreams: Record<'stdout' | 'stderr', DocEntry> = {
  stdout: { label: 'stdout', description: 'Standard output — normal application messages.' },
  stderr: { label: 'stderr', description: 'Error output — not always errors, many apps write info here too.' },
};

// --- PAGE DOCUMENTATION ---
export const pages: Record<
  'home' | 'templates' | 'templateDetail' | 'deployments' | 'deploymentDetail' | 'nodes' | 'sshKeys',
  PageDoc
> = {
  home: {
    title: 'Dashboard',
    subtitle: 'Overview of your deployments, templates, and infrastructure at a glance.',
    sections: {
      totalDeployments: {
        label: 'Total Deployments',
        description: 'All deployment instances you have created, across all templates.',
      },
      appTemplates: {
        label: 'App Templates',
        description: 'Templates you have created for the marketplace.',
      },
      nodes: {
        label: 'Nodes',
        description: 'Server nodes registered for running deployments.',
      },
      monthlyRevenue: {
        label: 'Monthly Revenue',
        description: 'Sum of monthly prices across all currently running deployments of your templates.',
      },
    },
    emptyState: {
      label: 'Nothing here yet',
      description: 'Deploy an app from the marketplace or create a template to get started.',
    },
  },
  templates: {
    title: 'Templates',
    subtitle: 'Browse and deploy ready-to-run apps, or create and manage your own.',
    sections: {
      howItWorks: {
        label: 'How It Works',
        description: 'Browse templates, click Deploy, and your app gets a URL, monitoring, and lifecycle controls.',
      },
      pricing: {
        label: 'Pricing',
        description: 'The monthly price covers hosting for as long as the deployment runs. Stop it anytime to pause billing. Free templates cost nothing.',
      },
    },
    emptyState: {
      label: 'No templates available',
      description: 'Check back later for new templates, or create your own.',
    },
  },
  templateDetail: {
    title: 'Template Details',
    subtitle: 'Everything about this template before you deploy.',
    sections: {
      services: {
        label: 'Included Services',
        description: 'Services are the individual containers that make up this application. For example, WordPress includes both a web server and a MySQL database.',
      },
      composeSpec: {
        label: 'Docker Compose Specification',
        description: 'The technical definition — which Docker images to run, how services connect, and what ports are exposed. You do not need to understand this to deploy.',
      },
      deploy: {
        label: 'What Happens When You Deploy',
        description: 'A new isolated instance is created on available infrastructure. You get a unique URL. The monthly price covers hosting as long as it runs.',
      },
      variables: {
        label: 'Configuration Variables',
        description: 'Values you can customize before deploying, like passwords or database names. Defaults are used if you leave optional ones blank.',
      },
    },
    emptyState: { label: 'Template not found', description: 'This template may have been removed.' },
  },
  deployments: {
    title: 'My Deployments',
    subtitle: 'Each card is a running (or stopped) application instance you created from the marketplace.',
    sections: {},
    emptyState: {
      label: 'No deployments yet',
      description: 'Visit the marketplace to browse templates, then deploy one with a single click.',
    },
  },
  deploymentDetail: {
    title: 'Deployment Details',
    subtitle: 'Monitor and manage this deployment.',
    sections: {
      domain: {
        label: 'Domain',
        description: 'The public URL where your application is accessible. Automatically generated when you deploy.',
      },
      containerHealth: {
        label: 'Container Health',
        description: 'Each service runs as a separate container. Health is determined by whether the container is running and passing configured health checks.',
      },
      resourceUsage: {
        label: 'Resource Usage',
        description: 'Real-time CPU, memory, and network snapshot. CPU % is relative to total host cores. Memory % is relative to the container limit.',
      },
      logs: {
        label: 'Container Logs',
        description: 'Application output from each container. Lines in red are from stderr. Not all stderr indicates a problem — many apps write info here too.',
      },
      stats: {
        label: 'Resource Statistics',
        description: 'Point-in-time resource usage for each running container. Network and Block I/O are cumulative totals since the container started.',
      },
      events: {
        label: 'Deployment Events',
        description: 'Timeline of lifecycle events, newest first. Shows when containers were created, started, stopped, or crashed.',
      },
    },
    emptyState: { label: 'Deployment not found', description: 'This deployment may have been deleted.' },
  },
  nodes: {
    title: 'My Nodes',
    subtitle: 'Servers where your deployments run. Add existing servers or provision cloud instances.',
    sections: {
      maintenance: {
        label: 'Maintenance Mode',
        description: 'Pauses new deployment scheduling to this node. Existing deployments keep running. Use when updating or rebooting a server.',
      },
      cloudProvisioning: {
        label: 'Cloud Provisioning',
        description: 'Create a cloud server instance and automatically register it as a deployment node. Includes SSH setup and Docker installation.',
      },
    },
    emptyState: {
      label: 'No worker nodes',
      description: 'Add an existing server or provision a cloud server to start deploying applications.',
    },
  },
  sshKeys: {
    title: 'SSH Keys',
    subtitle: 'SSH keys let Hoster securely connect to your worker nodes. Each key is encrypted with AES-256 before storage.',
    sections: {
      fingerprint: {
        label: 'Fingerprint',
        description: 'A unique identifier derived from your key. Use it to verify you uploaded the correct key without exposing the key itself.',
      },
      usedBy: {
        label: 'Used By',
        description: 'Shows which nodes reference this key. Deleting a key in use will break connectivity to those nodes.',
      },
      bestPractices: {
        label: 'Best Practices',
        description: 'Generate a dedicated key pair for Hoster (do not reuse personal keys). Use Ed25519 for best security. Rotate periodically.',
      },
    },
    emptyState: {
      label: 'No SSH keys',
      description: 'SSH keys authenticate Hoster to your servers without passwords. Generate a key pair with ssh-keygen, add the private key here, and place the public key on your server.',
    },
  },
};
