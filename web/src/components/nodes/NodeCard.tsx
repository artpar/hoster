import { useState, useRef, useEffect } from 'react';
import {
  Server,
  MapPin,
  Clock,
  MoreVertical,
  Wrench,
  Play,
  Trash2,
  AlertCircle,
} from 'lucide-react';
import type { Node } from '@/api/types';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { Badge } from '@/components/ui/Badge';
import { cn } from '@/lib/cn';

interface NodeCardProps {
  node: Node;
  onEnterMaintenance?: (id: string) => void;
  onExitMaintenance?: (id: string) => void;
  onDelete?: (id: string) => void;
  onDestroy?: (id: string) => void;
  isDeleting?: boolean;
  isUpdating?: boolean;
}

const providerLabels: Record<string, string> = {
  aws: 'AWS',
  digitalocean: 'DigitalOcean',
  hetzner: 'Hetzner',
  manual: 'Manual',
};

const statusStyles: Record<string, string> = {
  online: 'bg-green-100 text-green-800',
  offline: 'bg-red-100 text-red-800',
  maintenance: 'bg-yellow-100 text-yellow-800',
};

const statusLabels: Record<string, string> = {
  online: 'Online',
  offline: 'Offline',
  maintenance: 'Maintenance',
};

function formatBytes(mb: number): string {
  if (mb >= 1024) {
    return `${(mb / 1024).toFixed(1)} GB`;
  }
  return `${mb} MB`;
}

function formatPercent(used: number, total: number): number {
  if (total === 0) return 0;
  return Math.round((used / total) * 100);
}

function UsageBar({ used, total, label }: { used: number; total: number; label: string }) {
  const percent = formatPercent(used, total);
  return (
    <div className="space-y-1">
      <div className="flex justify-between text-xs">
        <span className="text-muted-foreground">{label}</span>
        <span className="font-medium">{percent}%</span>
      </div>
      <div className="h-2 w-full rounded-full bg-secondary">
        <div
          className={cn(
            'h-2 rounded-full transition-all',
            percent > 90 ? 'bg-red-500' : percent > 70 ? 'bg-yellow-500' : 'bg-green-500'
          )}
          style={{ width: `${percent}%` }}
        />
      </div>
    </div>
  );
}

export function NodeCard({
  node,
  onEnterMaintenance,
  onExitMaintenance,
  onDelete,
  onDestroy,
  isDeleting,
  isUpdating,
}: NodeCardProps) {
  const [showActions, setShowActions] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  const attrs = node.attributes;

  // Close dropdown on click outside
  useEffect(() => {
    if (!showActions) return;
    const handleClick = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as globalThis.Node)) {
        setShowActions(false);
      }
    };
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, [showActions]);
  // API returns flat capacity_* fields; build capacity object with safe defaults
  const capacity = attrs.capacity ?? {
    cpu_cores: (attrs as any).capacity_cpu_cores ?? 0,
    memory_mb: (attrs as any).capacity_memory_mb ?? 0,
    disk_mb: (attrs as any).capacity_disk_mb ?? 0,
    cpu_used: (attrs as any).capacity_cpu_used ?? 0,
    memory_used_mb: (attrs as any).capacity_memory_used_mb ?? 0,
    disk_used_mb: (attrs as any).capacity_disk_used_mb ?? 0,
  };

  const isCloudNode = attrs.provider_type && attrs.provider_type !== '' && attrs.provider_type !== 'manual';
  const originLabel = providerLabels[attrs.provider_type || ''] || 'Manual';

  const lastHealthCheck = attrs.last_health_check
    ? new Date(attrs.last_health_check).toLocaleString()
    : 'Never';

  return (
    <Card className="relative">
      <CardHeader className="pb-2">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-2">
            <Server className="h-5 w-5 text-muted-foreground" />
            <CardTitle className="text-lg">{attrs.name}</CardTitle>
            <Badge variant="outline" className="text-xs">{originLabel}</Badge>
            {attrs.public && (
              <Badge variant="secondary" className="text-xs">Public</Badge>
            )}
          </div>
          <div className="flex items-center gap-2">
            <span
              className={cn(
                'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium',
                statusStyles[attrs.status] || 'bg-gray-100 text-gray-800'
              )}
            >
              {statusLabels[attrs.status] || attrs.status}
            </span>
            <div className="relative" ref={menuRef}>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setShowActions(!showActions)}
                className="h-8 w-8 p-0"
              >
                <MoreVertical className="h-4 w-4" />
              </Button>
              {showActions && (
                <div className="absolute right-0 top-full z-50 mt-1 w-48 rounded-md border bg-background p-1 shadow-md">
                  {attrs.status === 'maintenance' ? (
                    <button
                      onClick={() => {
                        onExitMaintenance?.(node.id);
                        setShowActions(false);
                      }}
                      disabled={isUpdating}
                      className="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm hover:bg-accent"
                    >
                      <Play className="h-4 w-4" />
                      Exit Maintenance
                    </button>
                  ) : (
                    <button
                      onClick={() => {
                        onEnterMaintenance?.(node.id);
                        setShowActions(false);
                      }}
                      disabled={isUpdating}
                      className="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm hover:bg-accent"
                    >
                      <Wrench className="h-4 w-4" />
                      Enter Maintenance
                    </button>
                  )}
                  {isCloudNode ? (
                    <button
                      onClick={() => {
                        onDestroy?.(node.id);
                        setShowActions(false);
                      }}
                      disabled={isDeleting}
                      className="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm text-destructive hover:bg-destructive/10"
                    >
                      <Trash2 className="h-4 w-4" />
                      Destroy Node
                    </button>
                  ) : (
                    <button
                      onClick={() => {
                        onDelete?.(node.id);
                        setShowActions(false);
                      }}
                      disabled={isDeleting}
                      className="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm text-destructive hover:bg-destructive/10"
                    >
                      <Trash2 className="h-4 w-4" />
                      Delete Node
                    </button>
                  )}
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Connection Info */}
        <p className="mt-1 text-sm text-muted-foreground">
          {attrs.ssh_user}@{attrs.ssh_host}:{attrs.ssh_port}
        </p>
      </CardHeader>

      <CardContent className="space-y-4">
        {/* Error Message */}
        {attrs.error_message && (
          <div className="flex items-start gap-2 rounded-md bg-destructive/10 p-2 text-sm text-destructive">
            <AlertCircle className="h-4 w-4 mt-0.5 shrink-0" />
            <span>{attrs.error_message}</span>
          </div>
        )}

        {/* Capabilities */}
        <div className="flex flex-wrap gap-1">
          {(attrs.capabilities ?? []).map((cap) => (
            <Badge key={cap} variant="secondary" className="text-xs">
              {cap}
            </Badge>
          ))}
        </div>

        {/* Resource Usage */}
        <div className="space-y-2">
          <UsageBar
            used={capacity.cpu_used}
            total={capacity.cpu_cores}
            label={`CPU (${capacity.cpu_used.toFixed(1)} / ${capacity.cpu_cores} cores)`}
          />
          <UsageBar
            used={capacity.memory_used_mb}
            total={capacity.memory_mb}
            label={`Memory (${formatBytes(capacity.memory_used_mb)} / ${formatBytes(capacity.memory_mb)})`}
          />
          <UsageBar
            used={capacity.disk_used_mb}
            total={capacity.disk_mb}
            label={`Disk (${formatBytes(capacity.disk_used_mb)} / ${formatBytes(capacity.disk_mb)})`}
          />
        </div>

        {/* Footer Info */}
        <div className="flex flex-wrap gap-4 text-xs text-muted-foreground">
          {attrs.location && (
            <span className="flex items-center gap-1">
              <MapPin className="h-3 w-3" />
              {attrs.location}
            </span>
          )}
          <span className="flex items-center gap-1">
            <Clock className="h-3 w-3" />
            Last check: {lastHealthCheck}
          </span>
        </div>
      </CardContent>
    </Card>
  );
}
