import { Badge } from '@/components/ui/Badge';
import type { ProvisionStatus } from '@/api/types';

export function provisionStatusBadge(status: ProvisionStatus) {
  switch (status) {
    case 'pending':
    case 'creating':
    case 'configuring':
      return <Badge variant="warning">{status}</Badge>;
    case 'ready':
      return <Badge variant="success">{status}</Badge>;
    case 'failed':
      return <Badge variant="destructive">{status}</Badge>;
    case 'destroying':
    case 'destroyed':
      return <Badge variant="secondary">{status}</Badge>;
  }
}
