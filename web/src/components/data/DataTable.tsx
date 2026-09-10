import type { ReactNode } from 'react'
import {
  Box,
  LinearProgress,
  Skeleton,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TablePagination,
  TableRow,
} from '@mui/material'
import { EmptyState } from '@/components/feedback/States'

export interface Pagination {
  page: number
  pageSize: number
}

export interface Column<T> {
  key: string
  header: ReactNode
  width?: number
  align?: 'left' | 'right' | 'center'
  format?(value: unknown): ReactNode
  render?: (row: T) => ReactNode
}

interface DataTableProps<T> {
  rows: readonly T[]
  columns: readonly Column<T>[]
  loading?: boolean
  rowCount: number
  pagination: Pagination
  onPaginationChange: (next: Pagination) => void
  pageSizeOptions?: number[]
  onRowClick?: (row: T) => void
  emptyTitle: string
  emptyDescription?: string
  minWidth?: number
  maxHeight?: number | string
  label: string
}

function renderCell<T>(column: Column<T>, row: T): ReactNode {
  if (column.render) return column.render(row)

  const raw = (row as Record<string, unknown>)[column.key]
  if (column.format) return column.format(raw)

  return raw as ReactNode
}

export function DataTable<T>({
  rows,
  columns,
  loading = false,
  rowCount,
  pagination,
  onPaginationChange,
  pageSizeOptions = [10, 25, 50, 100],
  onRowClick,
  emptyTitle,
  emptyDescription,
  minWidth,
  maxHeight,
  label,
}: DataTableProps<T>) {
  const lastPage = Math.max(0, Math.ceil(rowCount / pagination.pageSize) - 1)
  const page = Math.min(pagination.page, lastPage)

  const firstLoad = loading && rows.length === 0
  const isEmpty = !loading && rows.length === 0

  return (
    <Box>
      <TableContainer sx={{ maxHeight, overflowX: 'auto' }}>
        <Table
          size="small"
          stickyHeader={maxHeight !== undefined}
          aria-label={label}
          sx={{ tableLayout: 'fixed', minWidth }}
        >
          <colgroup>
            {columns.map((column) => (
              <col key={column.key} style={column.width ? { width: column.width } : undefined} />
            ))}
          </colgroup>

          <TableHead>
            <TableRow>
              {columns.map((column) => (
                <TableCell key={column.key} align={column.align}>
                  {column.header}
                </TableCell>
              ))}
            </TableRow>
          </TableHead>

          <TableBody>
            {loading && rows.length > 0 && (
              <TableRow sx={{ '&:hover': { bgcolor: 'transparent' } }}>
                <TableCell colSpan={columns.length} sx={{ p: 0, border: 0 }}>
                  <LinearProgress sx={{ height: 2 }} />
                </TableCell>
              </TableRow>
            )}

            {firstLoad &&
              Array.from({ length: Math.min(pagination.pageSize, 8) }, (_, i) => (
                <TableRow key={`skeleton-${i}`}>
                  {columns.map((column) => (
                    <TableCell key={column.key}>
                      <Skeleton variant="rounded" height={18} />
                    </TableCell>
                  ))}
                </TableRow>
              ))}

            {isEmpty && (
              <TableRow sx={{ '&:hover': { bgcolor: 'transparent' } }}>
                <TableCell colSpan={columns.length} sx={{ border: 0 }}>
                  <EmptyState title={emptyTitle} description={emptyDescription} />
                </TableCell>
              </TableRow>
            )}

            {rows.map((row) => (
              <TableRow
                key={(row as { id: string | number }).id}
                hover={Boolean(onRowClick)}
                onClick={onRowClick ? () => onRowClick(row) : undefined}
                tabIndex={onRowClick ? 0 : undefined}
                onKeyDown={
                  onRowClick
                    ? (event) => {
                        if (event.key === 'Enter' || event.key === ' ') {
                          event.preventDefault()
                          onRowClick(row)
                        }
                      }
                    : undefined
                }
                sx={onRowClick ? { cursor: 'pointer' } : undefined}
              >
                {columns.map((column) => (
                  <TableCell
                    key={column.key}
                    align={column.align}
                    sx={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
                  >
                    {renderCell(column, row)}
                  </TableCell>
                ))}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>

      <TablePagination
        component="div"
        count={rowCount}
        page={page}
        rowsPerPage={pagination.pageSize}
        rowsPerPageOptions={pageSizeOptions}
        onPageChange={(_, next) => onPaginationChange({ ...pagination, page: next })}
        onRowsPerPageChange={(event) =>
          onPaginationChange({ page: 0, pageSize: Number(event.target.value) })
        }
      />
    </Box>
  )
}
