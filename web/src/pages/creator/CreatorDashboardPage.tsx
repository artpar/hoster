import { useState, useMemo } from 'react';
import {
  LayoutDashboard,
  Plus,
  Package,
  Users,
  TrendingUp,
  Search,
  Filter,
  Server,
  Key,
} from 'lucide-react';
import { useTemplates } from '@/hooks/useTemplates';
import { useDeployments } from '@/hooks/useDeployments';
import { useNodes, useDeleteNode, useEnterMaintenanceMode, useExitMaintenanceMode } from '@/hooks/useNodes';
import { useSSHKeys, useDeleteSSHKey } from '@/hooks/useSSHKeys';
import { useIsAuthenticated, useUserId } from '@/stores/authStore';
import { LoadingPage } from '@/components/common/LoadingSpinner';
import { EmptyState } from '@/components/common/EmptyState';
import { TemplateCard } from '@/components/templates/TemplateCard';
import { CreateTemplateDialog } from '@/components/templates/CreateTemplateDialog';
import { NodeCard, AddNodeDialog, AddSSHKeyDialog } from '@/components/nodes';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Select } from '@/components/ui/Select';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/Tabs';
import { Badge } from '@/components/ui/Badge';

type StatusFilter = 'all' | 'draft' | 'published' | 'deprecated';

export function CreatorDashboardPage() {
  const isAuthenticated = useIsAuthenticated();
  const userId = useUserId();
  const { data: templates, isLoading, error } = useTemplates();
  const { data: deployments } = useDeployments();
  const { data: nodes, isLoading: nodesLoading } = useNodes();
  const { data: sshKeys } = useSSHKeys();

  const deleteNode = useDeleteNode();
  const enterMaintenance = useEnterMaintenanceMode();
  const exitMaintenance = useExitMaintenanceMode();
  const deleteSSHKey = useDeleteSSHKey();

  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [addNodeDialogOpen, setAddNodeDialogOpen] = useState(false);
  const [addSSHKeyDialogOpen, setAddSSHKeyDialogOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');

  // Filter templates created by this user
  const myTemplates = useMemo(() => {
    if (!templates) return [];
    let result = templates.filter((t) => t.attributes.creator_id === userId);

    // Apply search filter
    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase();
      result = result.filter(
        (t) =>
          t.attributes.name.toLowerCase().includes(query) ||
          (t.attributes.description?.toLowerCase().includes(query) ?? false)
      );
    }

    // Apply status filter
    if (statusFilter !== 'all') {
      if (statusFilter === 'published') {
        result = result.filter((t) => t.attributes.published);
      } else if (statusFilter === 'draft') {
        result = result.filter((t) => !t.attributes.published);
      }
      // 'deprecated' status not supported in current model
    }

    return result;
  }, [templates, userId, searchQuery, statusFilter]);

  // Calculate stats
  const stats = useMemo(() => {
    if (!templates || !deployments) {
      return {
        totalTemplates: 0,
        publishedTemplates: 0,
        draftTemplates: 0,
        totalDeployments: 0,
        activeDeployments: 0,
        monthlyRevenue: 0,
      };
    }

    const userTemplates = templates.filter((t) => t.attributes.creator_id === userId);
    const templateIds = new Set(userTemplates.map((t) => t.id));
    const templateDeployments = deployments.filter((d) =>
      templateIds.has(d.attributes.template_id)
    );

    // Calculate monthly revenue (simplified)
    const activeTemplateDeployments = templateDeployments.filter(
      (d) => d.attributes.status === 'running'
    );
    const monthlyRevenue = activeTemplateDeployments.reduce((sum, d) => {
      const template = userTemplates.find((t) => t.id === d.attributes.template_id);
      return sum + (template?.attributes.price_monthly_cents ?? 0);
    }, 0);

    return {
      totalTemplates: userTemplates.length,
      publishedTemplates: userTemplates.filter((t) => t.attributes.published)
        .length,
      draftTemplates: userTemplates.filter((t) => !t.attributes.published).length,
      totalDeployments: templateDeployments.length,
      activeDeployments: activeTemplateDeployments.length,
      monthlyRevenue,
    };
  }, [templates, deployments, userId]);

  const handleCreateSuccess = (templateId: string) => {
    // Could navigate to edit page or just stay on dashboard
    console.log('Template created:', templateId);
  };

  if (!isAuthenticated) {
    return (
      <EmptyState
        icon={LayoutDashboard}
        title="Sign in required"
        description="Please sign in to access the creator dashboard"
      />
    );
  }

  if (isLoading) {
    return <LoadingPage />;
  }

  if (error) {
    return (
      <div className="rounded-md bg-destructive/10 p-4 text-destructive">
        Failed to load templates: {error.message}
      </div>
    );
  }

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold">Creator Dashboard</h1>
          <p className="text-muted-foreground">
            Manage your templates and track deployments
          </p>
        </div>
        <Button onClick={() => setCreateDialogOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Create Template
        </Button>
      </div>

      {/* Stats Cards */}
      <div className="mb-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Templates</CardTitle>
            <Package className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.totalTemplates}</div>
            <p className="text-xs text-muted-foreground">
              {stats.publishedTemplates} published, {stats.draftTemplates} drafts
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Deployments</CardTitle>
            <Users className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.totalDeployments}</div>
            <p className="text-xs text-muted-foreground">
              {stats.activeDeployments} currently active
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Monthly Revenue</CardTitle>
            <TrendingUp className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              ${(stats.monthlyRevenue / 100).toFixed(2)}
            </div>
            <p className="text-xs text-muted-foreground">From active deployments</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Published</CardTitle>
            <Package className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.publishedTemplates}</div>
            <p className="text-xs text-muted-foreground">Available in marketplace</p>
          </CardContent>
        </Card>
      </div>

      {/* Tabs for Templates, Nodes, and Analytics */}
      <Tabs defaultValue="templates">
        <TabsList>
          <TabsTrigger value="templates">
            <Package className="mr-1 h-4 w-4" />
            My Templates
          </TabsTrigger>
          <TabsTrigger value="nodes">
            <Server className="mr-1 h-4 w-4" />
            Nodes
            {nodes && nodes.length > 0 && (
              <Badge variant="secondary" className="ml-1.5">
                {nodes.length}
              </Badge>
            )}
          </TabsTrigger>
          <TabsTrigger value="analytics">
            <TrendingUp className="mr-1 h-4 w-4" />
            Analytics
          </TabsTrigger>
        </TabsList>

        {/* Templates Tab */}
        <TabsContent value="templates">
          {/* Search and Filters */}
          <div className="mb-4 flex flex-col gap-4 sm:flex-row sm:items-center">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                type="text"
                placeholder="Search your templates..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-9"
              />
            </div>
            <div className="flex items-center gap-2">
              <Filter className="h-4 w-4 text-muted-foreground" />
              <Select
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value as StatusFilter)}
                options={[
                  { value: 'all', label: 'All Status' },
                  { value: 'draft', label: 'Drafts' },
                  { value: 'published', label: 'Published' },
                  { value: 'deprecated', label: 'Deprecated' },
                ]}
                className="w-36"
              />
            </div>
          </div>

          {/* Templates Grid */}
          {myTemplates.length === 0 ? (
            <EmptyState
              icon={LayoutDashboard}
              title={searchQuery || statusFilter !== 'all' ? 'No matching templates' : 'No templates yet'}
              description={
                searchQuery || statusFilter !== 'all'
                  ? 'Try adjusting your search or filters'
                  : 'Create your first template to start earning'
              }
              action={
                !searchQuery && statusFilter === 'all'
                  ? {
                      label: 'Create Template',
                      onClick: () => setCreateDialogOpen(true),
                    }
                  : undefined
              }
            />
          ) : (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {myTemplates.map((template) => (
                <TemplateCard key={template.id} template={template} showActions />
              ))}
            </div>
          )}
        </TabsContent>

        {/* Nodes Tab */}
        <TabsContent value="nodes">
          {/* Actions Bar */}
          <div className="mb-4 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h3 className="text-lg font-semibold">Worker Nodes</h3>
              <p className="text-sm text-muted-foreground">
                Manage your VPS servers for running deployments
              </p>
            </div>
            <div className="flex gap-2">
              <Button variant="outline" onClick={() => setAddSSHKeyDialogOpen(true)}>
                <Key className="mr-2 h-4 w-4" />
                Manage SSH Keys
              </Button>
              <Button onClick={() => setAddNodeDialogOpen(true)}>
                <Plus className="mr-2 h-4 w-4" />
                Add Node
              </Button>
            </div>
          </div>

          {/* SSH Keys Summary */}
          {sshKeys && sshKeys.length > 0 && (
            <div className="mb-4 rounded-lg border bg-card p-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Key className="h-4 w-4 text-muted-foreground" />
                  <span className="text-sm font-medium">SSH Keys</span>
                </div>
                <div className="flex flex-wrap gap-2">
                  {sshKeys.map((key) => (
                    <div
                      key={key.id}
                      className="flex items-center gap-1 rounded-full bg-secondary px-2 py-1 text-xs"
                    >
                      <span className="font-medium">{key.attributes.name}</span>
                      <span className="text-muted-foreground">
                        ({key.attributes.fingerprint.substring(0, 12)}...)
                      </span>
                      <button
                        onClick={() => {
                          if (confirm(`Delete SSH key "${key.attributes.name}"?`)) {
                            deleteSSHKey.mutate(key.id);
                          }
                        }}
                        className="ml-1 text-muted-foreground hover:text-destructive"
                      >
                        &times;
                      </button>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}

          {/* Nodes Grid */}
          {nodesLoading ? (
            <div className="flex items-center justify-center py-12">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
            </div>
          ) : !nodes || nodes.length === 0 ? (
            <EmptyState
              icon={Server}
              title="No worker nodes"
              description="Add your first VPS server to start running deployments on your own infrastructure"
              action={{
                label: 'Add Node',
                onClick: () => setAddNodeDialogOpen(true),
              }}
            />
          ) : (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {nodes.map((node) => (
                <NodeCard
                  key={node.id}
                  node={node}
                  onEnterMaintenance={(id) => enterMaintenance.mutate(id)}
                  onExitMaintenance={(id) => exitMaintenance.mutate(id)}
                  onDelete={(id) => {
                    if (confirm(`Delete node "${node.attributes.name}"? This cannot be undone.`)) {
                      deleteNode.mutate(id);
                    }
                  }}
                  isDeleting={deleteNode.isPending}
                  isUpdating={enterMaintenance.isPending || exitMaintenance.isPending}
                />
              ))}
            </div>
          )}

          {/* Node Setup Tips */}
          <Card className="mt-6">
            <CardHeader>
              <CardTitle className="text-lg">Node Setup Guide</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <p className="text-sm text-muted-foreground">
                Before adding a node, ensure your VPS is properly configured:
              </p>
              <div className="rounded-md bg-secondary/50 p-4 font-mono text-xs">
                <p className="text-muted-foreground mb-2"># 1. Create deploy user with Docker access</p>
                <p>sudo useradd -m -s /bin/bash deploy</p>
                <p>sudo usermod -aG docker deploy</p>
                <p className="text-muted-foreground mt-4 mb-2"># 2. Set up SSH key authentication</p>
                <p>sudo mkdir -p /home/deploy/.ssh</p>
                <p>echo "YOUR_PUBLIC_KEY" | sudo tee /home/deploy/.ssh/authorized_keys</p>
                <p>sudo chmod 700 /home/deploy/.ssh</p>
                <p>sudo chmod 600 /home/deploy/.ssh/authorized_keys</p>
                <p>sudo chown -R deploy:deploy /home/deploy/.ssh</p>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Analytics Tab */}
        <TabsContent value="analytics">
          <div className="grid gap-4 md:grid-cols-2">
            {/* Deployment Stats by Template */}
            <Card>
              <CardHeader>
                <CardTitle className="text-lg">Deployments by Template</CardTitle>
              </CardHeader>
              <CardContent>
                {myTemplates.length === 0 ? (
                  <p className="text-sm text-muted-foreground">No templates yet</p>
                ) : (
                  <div className="space-y-3">
                    {myTemplates.map((template) => {
                      const templateDeployments =
                        deployments?.filter(
                          (d) => d.attributes.template_id === template.id
                        ).length ?? 0;
                      return (
                        <div
                          key={template.id}
                          className="flex items-center justify-between"
                        >
                          <span className="text-sm">{template.attributes.name}</span>
                          <span className="font-medium">{templateDeployments}</span>
                        </div>
                      );
                    })}
                  </div>
                )}
              </CardContent>
            </Card>

            {/* Revenue by Template */}
            <Card>
              <CardHeader>
                <CardTitle className="text-lg">Revenue by Template</CardTitle>
              </CardHeader>
              <CardContent>
                {myTemplates.length === 0 ? (
                  <p className="text-sm text-muted-foreground">No templates yet</p>
                ) : (
                  <div className="space-y-3">
                    {myTemplates.map((template) => {
                      const activeDeployments =
                        deployments?.filter(
                          (d) =>
                            d.attributes.template_id === template.id &&
                            d.attributes.status === 'running'
                        ).length ?? 0;
                      const revenue =
                        (activeDeployments * (template.attributes.price_monthly_cents || 0)) / 100;
                      return (
                        <div
                          key={template.id}
                          className="flex items-center justify-between"
                        >
                          <span className="text-sm">{template.attributes.name}</span>
                          <span className="font-medium">${revenue.toFixed(2)}/mo</span>
                        </div>
                      );
                    })}
                  </div>
                )}
              </CardContent>
            </Card>

            {/* Quick Tips */}
            <Card className="md:col-span-2">
              <CardHeader>
                <CardTitle className="text-lg">Creator Tips</CardTitle>
              </CardHeader>
              <CardContent>
                <ul className="space-y-2 text-sm text-muted-foreground">
                  <li>
                    <strong>Write clear descriptions</strong> - Help users understand what
                    your template does and what problems it solves.
                  </li>
                  <li>
                    <strong>Use semantic versioning</strong> - Follow X.Y.Z format to help
                    users track updates.
                  </li>
                  <li>
                    <strong>Test thoroughly</strong> - Make sure your compose spec works
                    before publishing.
                  </li>
                  <li>
                    <strong>Keep images public</strong> - All Docker images must be
                    publicly accessible.
                  </li>
                  <li>
                    <strong>Set fair pricing</strong> - Consider the value your template
                    provides when setting prices.
                  </li>
                </ul>
              </CardContent>
            </Card>
          </div>
        </TabsContent>
      </Tabs>

      {/* Create Template Dialog */}
      <CreateTemplateDialog
        open={createDialogOpen}
        onOpenChange={setCreateDialogOpen}
        onSuccess={handleCreateSuccess}
      />

      {/* Add Node Dialog */}
      <AddNodeDialog
        open={addNodeDialogOpen}
        onOpenChange={setAddNodeDialogOpen}
        onSuccess={(nodeId) => {
          console.log('Node created:', nodeId);
        }}
        onAddSSHKey={() => {
          setAddNodeDialogOpen(false);
          setAddSSHKeyDialogOpen(true);
        }}
      />

      {/* Add SSH Key Dialog */}
      <AddSSHKeyDialog
        open={addSSHKeyDialogOpen}
        onOpenChange={setAddSSHKeyDialogOpen}
        onSuccess={(keyId) => {
          console.log('SSH key created:', keyId);
          // Optionally re-open the Add Node dialog
          setAddNodeDialogOpen(true);
        }}
      />
    </div>
  );
}
