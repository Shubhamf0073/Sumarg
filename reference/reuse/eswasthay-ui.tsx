// Reference only — NOT compiled. Generic UI primitives from Previa's eSwasthay app (src/ui.tsx @ 6cbef03),
// trimmed to the parts that fit Previa Cargo. Port into frontend/src/components/ during M0.6 / M3, then:
//  - restyle with the demo's CSS classes (reference/demo) — class names below are eSwasthay's;
//  - DataTable: keep the "Simple view" (cards) / "Table view" switch and the responsive cards, but replace the
//    client-side search/sort/7-per-page paging with server params + limit+1 "Load more" (docs/02 §4 API rule);
//  - Modal: keep native <dialog> + portal (focus trap, Esc, backdrop click); drop the health-centre eyebrow text;
//  - lucide icons: import as `import { Search } from "lucide-react"` (tree-shaken by Vite).
// Removed: Avatar, PatientCell, QR (needs the unapproved `qrcode` package), DocumentPreview (letterheads).
import { createPortal } from "react-dom";
import { useEffect, useRef, useState, type ReactNode, type FormEvent } from "react";
import { ArrowDown, ArrowUp, ArrowLeft, ArrowRight, Search, X, Inbox } from "lucide-react";

export function Button({
  children,
  onClick,
  kind = "",
  type = "button",
  disabled = false,
}: {
  children: ReactNode;
  onClick?: () => void;
  kind?: string;
  type?: "button" | "submit";
  disabled?: boolean;
}) {
  return (
    <button
      type={type}
      className={`button ${kind}`}
      onClick={onClick}
      disabled={disabled}
    >
      {children}
    </button>
  );
}

export function Empty({
  title = "Nothing here yet",
  text = "Records will appear here as your team works.",
  action,
}: {
  title?: string;
  text?: string;
  action?: ReactNode;
}) {
  return (
    <div className="empty">
      <Inbox size={30} />
      <h3>{title}</h3>
      <p>{text}</p>
      {action}
    </div>
  );
}
export function Field({
  label,
  children,
  wide = false,
}: {
  label: string;
  children: ReactNode;
  wide?: boolean;
}) {
  return (
    <label className={`field ${wide ? "wide" : ""}`}>
      <span>{label}</span>
      {children}
    </label>
  );
}
export function Modal({
  title,
  children,
  onClose,
  wide = false,
}: {
  title: string;
  children: ReactNode;
  onClose: () => void;
  wide?: boolean;
}) {
  const ref = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    ref.current?.showModal();
    return () => ref.current?.close();
  }, []);
  return createPortal(
    <dialog
      ref={ref}
      className={`modal ${wide ? "large" : ""}`}
      onCancel={onClose}
      onClick={(e) => {
        if (e.target === ref.current) onClose();
      }}
    >
      <div className="modal-head">
        <div>
          <span className="eyebrow">
            {title.includes("Print")
              ? "CLINICAL DOCUMENT"
              : "HEALTH CENTRE WORKSPACE"}
          </span>
          <h2>{title}</h2>
        </div>
        <button
          className="icon-button"
          onClick={onClose}
          aria-label="Close dialog"
        >
          <X size={20} />
        </button>
      </div>
      {children}
    </dialog>,
    document.body,
  );
}
export function Form({
  children,
  onSubmit,
  label = "Save changes",
  onCancel,
}: {
  children: ReactNode;
  onSubmit: (values: Record<string, string>) => boolean | void | Promise<boolean | void>;
  label?: string;
  onCancel: () => void;
}) {
  const [submitting, setSubmitting] = useState(false);
  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (submitting) return;
    const values = Object.fromEntries(new FormData(e.currentTarget)) as Record<string, string>;
    setSubmitting(true);
    try {
      await onSubmit(values);
    } finally {
      setSubmitting(false);
    }
  }
  return (
    <form onSubmit={submit} onInvalidCapture={(event) => {
      const section = (event.target as HTMLElement).closest("details") as HTMLDetailsElement | null;
      if (section) section.open = true;
    }}>
      <div className="form-grid">{children}</div>
      <div className="form-footer">
        <Button onClick={onCancel}>Cancel</Button>
        <Button type="submit" kind="primary" disabled={submitting}>
          {submitting ? "Saving…" : label}
        </Button>
      </div>
    </form>
  );
}
export type Column<T> = {
  label: string;
  render: (row: T) => ReactNode;
  sort?: (row: T) => string | number;
};
export function DataTable<T>({
  rows,
  columns,
  keyOf,
  searchText,
  placeholder = "Search records…",
  tools,
  emptyText,
}: {
  rows: T[];
  columns: Column<T>[];
  keyOf: (r: T) => string;
  searchText: (r: T) => string;
  placeholder?: string;
  tools?: ReactNode;
  emptyText?: string;
}) {
  const [search, setSearch] = useState(""),
    [page, setPage] = useState(0),
    [sort, setSort] = useState(-1),
    [asc, setAsc] = useState(true),
    [view, setView] = useState<"cards" | "table">("cards");
  let filtered = rows.filter((r) =>
    searchText(r).toLowerCase().includes(search.toLowerCase()),
  );
  if (sort >= 0 && columns[sort]?.sort) {
    const fn = columns[sort].sort!;
    filtered = [...filtered].sort((a, b) => {
      const x = fn(a),
        y = fn(b);
      return (
        (typeof x === "number" && typeof y === "number"
          ? x - y
          : String(x).localeCompare(String(y))) * (asc ? 1 : -1)
      );
    });
  }
  const current = Math.min(
      page,
      Math.max(0, Math.ceil(filtered.length / 7) - 1),
    ),
    visible = filtered.slice(current * 7, current * 7 + 7);
  return (
    <>
      <div className="table-toolbar">
        <div className="search-box">
          <Search size={16} />
          <input
            aria-label={placeholder}
            placeholder={placeholder}
            value={search}
            onChange={(e) => {
              setSearch(e.target.value);
              setPage(0);
            }}
          />
        </div>
        <div className="record-view-switch" role="group" aria-label="Record display">
          <button type="button" className={view === "cards" ? "active" : ""} aria-pressed={view === "cards"} onClick={() => setView("cards")}>Simple view</button>
          <button type="button" className={view === "table" ? "active" : ""} aria-pressed={view === "table"} onClick={() => setView("table")}>Table view</button>
        </div>
        {tools}
      </div>
      {view === "cards" ? <div className="record-cards">
        {visible.map((row) => <article className="record-card" key={keyOf(row)}>
          {columns.map((column, index) => <div className={index === 0 ? "record-card-lead" : "record-card-field"} key={column.label}>
            <span>{column.label}</span><div>{column.render(row)}</div>
          </div>)}
        </article>)}
        {!visible.length && <Empty title="No matching records" text={emptyText || "Try another search or add a record to get started."} />}
      </div> : <div className="table-scroll">
        <table>
          <thead>
            <tr>
              {columns.map((c, i) => (
                <th key={c.label}>
                  {c.sort ? (
                    <button
                      className="sort-button"
                      onClick={() => {
                        setSort(i);
                        setAsc(sort === i ? !asc : true);
                      }}
                    >
                      {c.label}
                      {sort === i ? (
                        asc ? (
                          <ArrowUp size={12} />
                        ) : (
                          <ArrowDown size={12} />
                        )
                      ) : null}
                    </button>
                  ) : (
                    c.label
                  )}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {visible.map((r) => (
              <tr key={keyOf(r)}>
                {columns.map((c) => (
                  <td key={c.label}>{c.render(r)}</td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
        {!visible.length && (
          <Empty
            title="No matching records"
            text={
              emptyText || "Try another search or add a record to get started."
            }
          />
        )}
      </div>}
      <div className="pagination">
        <span>
          {filtered.length
            ? `${current * 7 + 1}–${Math.min(current * 7 + 7, filtered.length)}`
            : "0"}{" "}
          of {filtered.length} records
        </span>
        <div>
          <button
            className="icon-button"
            disabled={current === 0}
            onClick={() => setPage(current - 1)}
            aria-label="Previous page"
          >
            <ArrowLeft size={15} />
          </button>
          <span>{current + 1}</span>
          <button
            className="icon-button"
            disabled={(current + 1) * 7 >= filtered.length}
            onClick={() => setPage(current + 1)}
            aria-label="Next page"
          >
            <ArrowRight size={15} />
          </button>
        </div>
      </div>
    </>
  );
}

export function downloadCSV(name: string, rows: (string | number)[][]) {
  const text = rows
    .map((row) =>
      row
        .map((value) => {
          let v = String(value);
          if (/^[=+@\-]/.test(v)) v = "'" + v;
          return '"' + v.replaceAll('"', '""') + '"';
        })
        .join(","),
    )
    .join("\r\n");
  const url = URL.createObjectURL(
    new Blob(["\uFEFF" + text], { type: "text/csv;charset=utf-8" }),
  );
  const a = document.createElement("a");
  a.href = url;
  a.download = name;
  a.click();
  URL.revokeObjectURL(url);
}
