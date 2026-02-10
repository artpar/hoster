import { Cloud, RefreshCw, Trash2, AlertCircle, Loader2 } from 'lucide-react';
import type { CloudProvision } from '@/api/types';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { Badge } from '@/components/ui/Badge';
import { provisionStatusBadge } from '@/components/cloud';

interface ProvisionCardProps {
  provision: CloudProvision;
  onRetry?: (id: string) => void;
  onDestroy?: (id: string) => void;
  isRetrying?: boolean;
}

const providerLabels: Record<string, string> = {
  aws: 'AWS',
  digitalocean: 'DigitalOcean',
  hetzner: 'Hetzner',
};

export function ProvisionCard({ provision, onRetry, onDestroy, isRetrying }: ProvisionCardProps) {
  const attrs = provision.attributes;
  const isActive = attrs.status === 'pending' || attrs.status === 'creating' || attrs.status === 'configuring';

  return (
    <Card className="relative border-dashed">
      <CardHeader className="pb-2">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-2">
            <Cloud className="h-5 w-5 text-muted-foreground" />
            <CardTitle className="text-lg">{attrs.instance_name}</CardTitle>
          </div>
          <div className="flex items-center gap-2">
            {provisionStatusBadge(attrs.status)}
          </div>
        </div>
        <div className="mt-1 flex items-center gap-2">
          <Badge variant="outline">{providerLabels[attrs.provider] || attrs.provider}</Badge>
          <span className="text-xs text-muted-foreground">{attrs.region}</span>
          <span className="text-xs text-muted-foreground">{attrs.size}</span>
        </div>
      </CardHeader>

      <CardContent className="space-y-3">
        {/* Progress / step */}
        {isActive && attrs.current_step && (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Loader2 className="h-3 w-3 animate-spin" />
            <span>{attrs.current_step}</span>
          </div>
        )}

        {/* Public IP if available */}
        {attrs.public_ip && (
          <p className="text-sm text-muted-foreground">
            IP: <span className="font-mono">{attrs.public_ip}</span>
          </p>
        )}

        {/* Error message */}
        {attrs.error_message && (
          <div className="flex items-start gap-2 rounded-md bg-destructive/10 p-2 text-sm text-destructive">
            <AlertCircle className="h-4 w-4 mt-0.5 shrink-0" />
            <span>{attrs.error_message}</span>
          </div>
        )}

        {/* Actions */}
        <div className="flex items-center gap-2">
          {attrs.status === 'failed' && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => onRetry?.(provision.id)}
              disabled={isRetrying}
            >
              <RefreshCw className="mr-1 h-3 w-3" />
              Retry
            </Button>
          )}
          {attrs.status !== 'destroying' && attrs.status !== 'destroyed' && (
            <Button
              variant="outline"
              size="sm"
              className="text-destructive hover:text-destructive"
              onClick={() => onDestroy?.(provision.id)}
            >
              <Trash2 className="mr-1 h-3 w-3" />
              Destroy
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
