import type { ReactNode } from 'react'
import Box from '@mui/material/Box'
import LinearProgress from '@mui/material/LinearProgress'
import Skeleton from '@mui/material/Skeleton'
import Table from '@mui/material/Table'
import TableBody from '@mui/material/TableBody'
import TableCell from '@mui/material/TableCell'
import TableContainer from '@mui/material/TableContainer'
import TableHead from '@mui/material/TableHead'
import TablePagination from '@mui/material/TablePagination'
import TableRow from '@mui/material/TableRow'
import type { SxProps, Theme } from '@mui/material/styles'
import { EmptyState } from '@/components/feedback/States'

/**
 * The server-paginated read-only table used across the console.
 */

export interface Pagination {
  page: number
  pageSize: number
}

export interface Column<T> {
  key: string
  header: ReactNode
  /**
   * Fixed width in px. Leave it off exactly one column. That one absorbs the
   * slack, which is what DataGrid's `flex: 1` did.
   */
  width?: number
  align?: 'left' | 'right' | 'center'
  /** Defaults to `row[key]`. */
  value?: (row: T) => unknown
  /**
   * Formats the accessed value. Typed `never` so a call site can narrow it
   * freely: `(v: string) => ...` is assignable, since parameters are checked
   * contravariantly and `never` is assignable to everything. The one unsafe
   * cast that buys is confined to `renderCell` below.
   */
  format?: (value: never, row: T) => ReactNode
  /** Full control over the cell. Wins over `value`/`format`. */
  render?: (row: T) => ReactNode
  cellSx?: SxProps<Theme>
}

export interface DataTableProps<T> {
  rows: readonly T[]
  columns: readonly Column<T>[]
  /** Defaults to `row.id`. */
  getRowId?: (row: T) => string | number
  loading?: boolean
  rowCount: number
  pagination: Pagination
  onPaginationChange: (next: Pagination) => void
  pageSizeOptions?: number[]
  onRowClick?: (row: T) => void
  emptyTitle?: string
  emptyDescription?: string
  /** Floor before the container scrolls sideways instead of squashing. */
  minWidth?: number
  /** When set, the container scrolls and the header sticks. */
  maxHeight?: number | string
  /** Accessible name for the table. */
  label: string
}

function renderCell<T>(column: Column<T>, row: T): ReactNode {
  if (column.render) return column.render(row)

  const raw = column.value ? column.value(row) : (row as Record<string, unknown>)[column.key]
  if (column.format) return column.format(raw as never, row)

  return raw as ReactNode
}

export function DataTable<T>({
  rows,
  columns,
  getRowId = (row) => (row as { id: string | number }).id,
  loading = false,
  rowCount,
  pagination,
  onPaginationChange,
  pageSizeOptions = [10, 25, 50, 100],
  onRowClick,
  emptyTitle = 'Nothing to show',
  emptyDescription,
  minWidth,
  maxHeight,
  label,
}: DataTableProps<T>) {
  // A filter change can leave the page index past the end of a shrunken result
  // set. TablePagination warns and renders a negative range when that happens.
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
          {/*
            Under `table-layout: fixed`, a <col> with no width shares out
            whatever is left over. That is the whole of DataGrid's flex sizing.
          */}
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
            {/*
              Refetching keeps the current rows and shows a hairline above them.
              Blanking the table on every page change reads as an error.
            */}
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
                key={getRowId(row)}
                hover={Boolean(onRowClick)}
                onClick={onRowClick ? () => onRowClick(row) : undefined}
                // A bare onClick on a row is mouse-only.
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
                    sx={[
                      { overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' },
                      ...(Array.isArray(column.cellSx) ? column.cellSx : [column.cellSx]),
                    ]}
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
        // Changing page size while deep in the list would otherwise land on a
        // page that no longer exists.
        onRowsPerPageChange={(event) =>
          onPaginationChange({ page: 0, pageSize: Number(event.target.value) })
        }
      />
    </Box>
  )
}
