import { ReactNode, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { ChevronsUpDown, ChevronUp, ChevronDown, Search, X } from "lucide-react";
import clsx from "clsx";
import { Card } from "./Card";
import { TableSkeleton } from "./Skeleton";
import { EmptyState } from "./EmptyState";
import { Input } from "./Input";
import { Button } from "./Button";

export interface DataTableColumn<T> {
  key: string;
  header: ReactNode;
  cell: (row: T) => ReactNode;
  sortValue?: (row: T) => string | number;
  headerClassName?: string;
  cellClassName?: string;
}

interface DataTableProps<T> {
  columns: DataTableColumn<T>[];
  data: T[];
  getRowId: (row: T) => string;
  isLoading?: boolean;
  emptyIcon?: ReactNode;
  emptyTitle: string;
  emptyDescription?: string;
  emptyAction?: ReactNode;
  searchPlaceholder?: string;
  searchFn?: (row: T, query: string) => boolean;
  toolbarExtra?: ReactNode;
  pageSize?: number;
  onRowClick?: (row: T) => void;
  renderExpanded?: (row: T) => ReactNode;
  isRowExpanded?: (row: T) => boolean;
  rowClassName?: (row: T) => string;
}

type SortDir = "asc" | "desc" | null;

export function DataTable<T>({
  columns,
  data,
  getRowId,
  isLoading,
  emptyIcon,
  emptyTitle,
  emptyDescription,
  emptyAction,
  searchPlaceholder,
  searchFn,
  toolbarExtra,
  pageSize = 10,
  onRowClick,
  renderExpanded,
  isRowExpanded,
  rowClassName,
}: DataTableProps<T>) {
  const { t } = useTranslation();
  const [query, setQuery] = useState("");
  const [sortKey, setSortKey] = useState<string | null>(null);
  const [sortDir, setSortDir] = useState<SortDir>(null);
  const [page, setPage] = useState(1);

  const filtered = useMemo(() => {
    if (!searchFn || !query.trim()) return data;
    return data.filter((row) => searchFn(row, query.trim()));
  }, [data, query, searchFn]);

  const sorted = useMemo(() => {
    if (!sortKey || !sortDir) return filtered;
    const col = columns.find((c) => c.key === sortKey);
    if (!col?.sortValue) return filtered;
    const copy = [...filtered];
    copy.sort((a, b) => {
      const av = col.sortValue!(a);
      const bv = col.sortValue!(b);
      if (av < bv) return sortDir === "asc" ? -1 : 1;
      if (av > bv) return sortDir === "asc" ? 1 : -1;
      return 0;
    });
    return copy;
  }, [filtered, sortKey, sortDir, columns]);

  const totalPages = Math.max(1, Math.ceil(sorted.length / pageSize));
  const currentPage = Math.min(page, totalPages);
  const paginated = sorted.slice((currentPage - 1) * pageSize, currentPage * pageSize);

  function handleSort(col: DataTableColumn<T>) {
    if (!col.sortValue) return;
    if (sortKey !== col.key) {
      setSortKey(col.key);
      setSortDir("asc");
    } else if (sortDir === "asc") {
      setSortDir("desc");
    } else if (sortDir === "desc") {
      setSortKey(null);
      setSortDir(null);
    } else {
      setSortDir("asc");
    }
    setPage(1);
  }

  const hasToolbar = !!searchFn || !!toolbarExtra;

  if (isLoading) {
    return (
      <Card>
        <TableSkeleton cols={columns.length} />
      </Card>
    );
  }

  // نکته‌ی مهم (بازخورد کاربر ۲۰۲۶-۰۷-۰۵): قبلاً وقتی data.length===0 بود، کل toolbar
  // (شاملِ toolbarExtra — مثلاً فیلتر وضعیت در AdminInstances/Instances/AdminPayments) هم
  // مخفی می‌شد، چون این‌جا زودتر از رندرِ toolbar یک return جداگانه داشتیم. یعنی اگر کاربر
  // فیلتر را روی وضعیتی می‌گذاشت که هیچ نتیجه‌ای نداشت، تنها راهش برای برگرداندن فیلتر، ترک
  // کردن کامل صفحه و برگشتن بود — چون خودِ کنترل فیلتر هم با نتیجه ناپدید می‌شد. حالا toolbar
  // همیشه (وقتی hasToolbar باشد) رندر می‌شود و فقط بدنه‌ی جدول با EmptyState جایگزین می‌شود.
  const isEmpty = data.length === 0;

  return (
    <Card className="overflow-hidden p-0">
      {hasToolbar && (
        <div className="flex flex-wrap items-center gap-2 border-b border-slate-200 p-3 dark:border-white/10">
          {searchFn && (
            <div className="relative min-w-[200px] flex-1">
              <Search className="pointer-events-none absolute start-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
              <Input
                value={query}
                onChange={(e) => {
                  setQuery(e.target.value);
                  setPage(1);
                }}
                placeholder={searchPlaceholder ?? t("common.search")}
                className="ps-9"
              />
              {query && (
                <button
                  onClick={() => setQuery("")}
                  aria-label={t("common.clearSearch")}
                  className="absolute end-2 top-1/2 -translate-y-1/2 rounded p-0.5 text-slate-400 hover:bg-slate-100 dark:hover:bg-white/10"
                >
                  <X className="h-3.5 w-3.5" />
                </button>
              )}
            </div>
          )}
          {toolbarExtra}
        </div>
      )}

      {isEmpty ? (
        <div className="p-5">
          <EmptyState icon={emptyIcon} title={emptyTitle} description={emptyDescription} action={emptyAction} />
        </div>
      ) : (
        <>
          <div className="overflow-x-auto">
            <table className="data-table w-full text-sm">
              <thead className="border-b border-slate-200 text-start text-xs text-slate-500 dark:border-white/10 dark:text-slate-400">
                <tr>
                  {columns.map((col) => (
                    <th
                      key={col.key}
                      onClick={() => handleSort(col)}
                      className={clsx(
                        "px-4 py-3 font-medium",
                        col.sortValue && "cursor-pointer select-none hover:text-slate-700 dark:hover:text-slate-200",
                        col.headerClassName
                      )}
                    >
                      <span className="inline-flex items-center gap-1">
                        {col.header}
                        {col.sortValue &&
                          (sortKey === col.key ? (
                            sortDir === "asc" ? (
                              <ChevronUp className="h-3.5 w-3.5" />
                            ) : (
                              <ChevronDown className="h-3.5 w-3.5" />
                            )
                          ) : (
                            <ChevronsUpDown className="h-3.5 w-3.5 opacity-40" />
                          ))}
                      </span>
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 dark:divide-white/5">
                {paginated.length === 0 ? (
                  <tr>
                    <td colSpan={columns.length} className="px-4 py-10 text-center">
                      <p className="text-sm text-slate-500 dark:text-slate-400">{t("common.noResults")}</p>
                      <p className="mt-1 text-xs text-slate-400">{t("common.noResultsHint")}</p>
                      {query && (
                        <Button variant="secondary" size="sm" className="mt-3" onClick={() => setQuery("")}>
                          {t("common.clearSearch")}
                        </Button>
                      )}
                    </td>
                  </tr>
                ) : (
                  paginated.map((row) => {
                    const id = getRowId(row);
                    const expanded = isRowExpanded?.(row);
                    return (
                      <>
                        <tr
                          key={id}
                          onClick={() => onRowClick?.(row)}
                          className={clsx(onRowClick && "cursor-pointer", rowClassName?.(row))}
                        >
                          {columns.map((col) => (
                            <td key={col.key} className={clsx("px-4 py-2.5", col.cellClassName)}>
                              {col.cell(row)}
                            </td>
                          ))}
                        </tr>
                        {expanded && renderExpanded && (
                          <tr key={`${id}-expanded`}>
                            <td colSpan={columns.length} className="bg-slate-50 p-0 dark:bg-white/5">
                              {renderExpanded(row)}
                            </td>
                          </tr>
                        )}
                      </>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>

          {sorted.length > 0 && (
            <div className="flex flex-wrap items-center justify-between gap-2 border-t border-slate-200 px-4 py-2.5 text-xs text-slate-500 dark:border-white/10 dark:text-slate-400">
              <span>{t("common.totalItems", { count: sorted.length })}</span>
              {totalPages > 1 && (
                <div className="flex items-center gap-2">
                  <span>{t("common.pageOf", { current: currentPage, total: totalPages })}</span>
                  <div className="flex gap-1">
                    <Button
                      size="sm"
                      variant="secondary"
                      disabled={currentPage <= 1}
                      onClick={() => setPage((p) => Math.max(1, p - 1))}
                    >
                      {t("common.previous")}
                    </Button>
                    <Button
                      size="sm"
                      variant="secondary"
                      disabled={currentPage >= totalPages}
                      onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                    >
                      {t("common.next")}
                    </Button>
                  </div>
                </div>
              )}
            </div>
          )}
        </>
      )}
    </Card>
  );
}
