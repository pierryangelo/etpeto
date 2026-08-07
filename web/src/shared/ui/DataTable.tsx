import type { ReactNode } from "react";

/**
 * One table for every listing. A screen supplies columns and rows; how a table
 * looks — and that numeric columns line up on the digits — is decided here.
 */

export type Cell = {
  content: ReactNode;
  /** Lends weight to the cell that names the row. */
  emphasis?: boolean;
  numeric?: boolean;
  align?: "left" | "right";
};

export type Row = {
  key: string;
  cells: Cell[];
};

export function DataTable({
  columns,
  rows,
  empty,
}: {
  columns: string[];
  rows: Row[];
  empty?: ReactNode;
}) {
  if (rows.length === 0 && empty) {
    return <p className="ds-meta">{empty}</p>;
  }
  return (
    <div className="overflow-x-auto">
      <table className="w-full border-collapse overflow-hidden rounded-2 border border-line bg-paper text-12.5px">
        <thead>
          <tr className="bg-panel">
            {columns.map((column, index) => (
              <th
                key={`${column}-${index}`}
                scope="col"
                className="border-b border-line px-3 py-2 text-left text-10px font-700 tracking-wider uppercase text-subtle"
              >
                {column}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row.key} className="border-b border-line last:border-b-0">
              {row.cells.map((cell, index) => (
                <td
                  key={index}
                  className={[
                    "px-3 py-2.5",
                    cell.emphasis ? "font-550 text-ink" : "text-text",
                    cell.numeric ? "tabular-nums" : "",
                    cell.align === "right" ? "text-right" : "",
                  ].join(" ")}
                >
                  {cell.content}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
