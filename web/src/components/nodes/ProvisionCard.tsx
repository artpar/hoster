import { Cloud, RefreshCw, Trash2, AlertCircle, Loader2, Check, Circle } from 'lucide-react';
import type { CloudProvision, ProvisionStatus } from '@/api/types';
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

const PROVISION_STEPS = ['pending', 'creating', 'configuring', 'ready'] as const;

function stepState(step: string, currentStatus: ProvisionStatus): 'completed' | 'active' | 'pending' {
  const stepIdx = PROVISION_STEPS.indexOf(step as typeof PROVISION_STEPS[number]);
  const statusIdx = PROVISION_STEPS.indexOf(currentStatus as typeof PROVISION_STEPS[number]);

  if (statusIdx < 0) {
    // Status is failed/destroying/destroyed — show steps up to where it failed
    return 'pending';
  }
  if (stepIdx < statusIdx) return 'completed';
  if (stepIdx === statusIdx) return 'active';
  return 'pending';
}

const stepLabels: Record<string, string> = {
  pending: 'Queued',
  creating: 'Creating',
  configuring: 'Configuring',
  ready: 'Ready',
};

function StepTimeline({ status }: { status: ProvisionStatus }) {
  const isDestructive = status === 'failed' || status === 'destroying' || status === 'destroyed';
  if (isDestructive) return null;

  return (
    <div className="flex items-center gap-1">
      {PROVISION_STEPS.map((step, i) => {
        const state = stepState(step, status);
        return (
          <div key={step} className="flex items-center gap-1">
            <div className="flex flex-col items-center">
              <div className="flex items-center gap-1">
                {state === 'completed' && (
                  <div className="flex h-5 w-5 items-center justify-center rounded-full bg-primary text-primary-foreground">
                    <Check className="h-3 w-3" />
                  </div>
                )}
                {state === 'active' && (
                  <div className="flex h-5 w-5 items-center justify-center rounded-full border-2 border-primary">
                    <Loader2 className="h-3 w-3 animate-spin text-primary" />
                  </div>
                )}
                {state === 'pending' && (
                  <div className="flex h-5 w-5 items-center justify-center rounded-full border-2 border-muted-foreground/30">
                    <Circle className="h-2 w-2 text-muted-foreground/30" />
                  </div>
                )}
              </div>
              <span className={`mt-1 text-[10px] leading-tight ${
                state === 'active' ? 'font-medium text-primary' :
                state === 'completed' ? 'text-muted-foreground' :
                'text-muted-foreground/50'
              }`}>
                {stepLabels[step]}
              </span>
            </div>
            {i < PROVISION_STEPS.length - 1 && (
              <div className={`mb-4 h-px w-6 ${
                state === 'completed' ? 'bg-primary' : 'bg-muted-foreground/20'
              }`} />
            )}
          </div>
        );
      })}
    </div>
  );
}

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
        {/* Step timeline for active provisions */}
        <StepTimeline status={attrs.status} />

        {/* Current step detail text */}
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
