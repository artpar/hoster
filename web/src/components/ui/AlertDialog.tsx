import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from './Dialog';
import { Button } from './Button';

interface AlertDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description?: string;
  buttonLabel?: string;
}

export function AlertDialog({
  open,
  onOpenChange,
  title,
  description,
  buttonLabel = 'OK',
}: AlertDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          {description && <DialogDescription>{description}</DialogDescription>}
        </DialogHeader>
        <DialogFooter>
          <Button onClick={() => onOpenChange(false)}>{buttonLabel}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
