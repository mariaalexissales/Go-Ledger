import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Stack,
  TextField,
  Typography,
} from '@mui/material'
import { useCreateTransaction } from '../transactions.queries'
import { ErrorState } from '@/components/feedback/States'

const schema = z.object({
  account_id: z
    .string()
    .min(1, 'Required')
    .refine((v) => Number.isInteger(Number(v)) && Number(v) > 0, 'Must be a positive whole number'),
  amount: z
    .string()
    .min(1, 'Required')
    .refine((v) => Number.isFinite(Number(v)), 'Enter a number')
    .refine((v) => Number(v) !== 0, 'Amount cannot be zero'),
})

type FormValues = z.infer<typeof schema>

export function CreateTransactionDialog({
  open,
  onClose,
  accountId,
}: {
  open: boolean
  onClose: () => void
  accountId?: number
}) {
  const createTransaction = useCreateTransaction()

  const { register, handleSubmit, formState, reset } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { account_id: accountId ? String(accountId) : '', amount: '' },
  })

  const close = () => {
    reset()
    createTransaction.reset()
    onClose()
  }

  const onSubmit = handleSubmit((values) => {
    createTransaction.mutate(
      { account_id: Number(values.account_id), amount: Number(values.amount) },
      { onSuccess: close },
    )
  })

  return (
    <Dialog open={open} onClose={close} fullWidth maxWidth="xs">
      <form onSubmit={onSubmit} noValidate>
        <DialogTitle>New transaction</DialogTitle>

        <DialogContent>
          <Stack spacing={2} sx={{ pt: 1 }}>
            {createTransaction.isError && <ErrorState error={createTransaction.error} />}

            <TextField
              label="Account ID"
              type="number"
              fullWidth
              disabled={accountId !== undefined}
              error={Boolean(formState.errors.account_id)}
              helperText={formState.errors.account_id?.message}
              {...register('account_id')}
            />

            <TextField
              label="Amount"
              type="number"
              fullWidth
              slotProps={{ htmlInput: { step: '0.01' } }}
              error={Boolean(formState.errors.amount)}
              helperText={formState.errors.amount?.message}
              {...register('amount')}
            />

            <Typography variant="caption" color="text.secondary">
              Negative amounts are debits. The account balance updates in the same transaction.
            </Typography>
          </Stack>
        </DialogContent>

        <DialogActions>
          <Button onClick={close}>Cancel</Button>
          <Button type="submit" variant="contained" disabled={createTransaction.isPending}>
            {createTransaction.isPending ? 'Posting…' : 'Post'}
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  )
}
