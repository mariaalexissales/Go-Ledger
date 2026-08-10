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
} from '@mui/material'
import { useCreateAccount } from '../accounts.queries'
import { ErrorState } from '@/components/feedback/States'

const schema = z.object({
  name: z.string().trim().min(1, 'A name is required').max(255, 'Name is too long'),
})

type FormValues = z.infer<typeof schema>

export function CreateAccountDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const createAccount = useCreateAccount()

  const { register, handleSubmit, formState, reset } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { name: '' },
  })

  const close = () => {
    reset()
    createAccount.reset()
    onClose()
  }

  const onSubmit = handleSubmit((values) => {
    createAccount.mutate(values, { onSuccess: close })
  })

  return (
    <Dialog open={open} onClose={close} fullWidth maxWidth="xs">
      <form onSubmit={onSubmit} noValidate>
        <DialogTitle>New account</DialogTitle>

        <DialogContent>
          <Stack spacing={2} sx={{ pt: 1 }}>
            {createAccount.isError && <ErrorState error={createAccount.error} />}

            <TextField
              label="Account name"
              autoFocus
              fullWidth
              error={Boolean(formState.errors.name)}
              helperText={formState.errors.name?.message}
              {...register('name')}
            />
          </Stack>
        </DialogContent>

        <DialogActions>
          <Button onClick={close}>Cancel</Button>
          <Button type="submit" variant="contained" disabled={createAccount.isPending}>
            {createAccount.isPending ? 'Creating…' : 'Create'}
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  )
}
