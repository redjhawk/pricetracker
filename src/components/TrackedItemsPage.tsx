import { useMemo, useState } from "react";
import {
  Button,
  InlineLoading,
  InlineNotification,
  OverflowMenu,
  OverflowMenuItem,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableHeader,
  TableRow,
  TableToolbar,
  TableToolbarContent,
  TextInput,
  Tag,
} from "@carbon/react";
import { Add, Launch } from "@carbon/icons-react";
import type { TrackedItem } from "../types";
import StatusTag from "./StatusTag";

interface Props {
  items: TrackedItem[];
  loading: boolean;
  error: string | null;
  onRetry: () => void;
  onAdd: () => void;
  onViewDetail: (item: TrackedItem) => void;
  onDelete: (item: TrackedItem) => void;
}

const euro = new Intl.NumberFormat("de-DE", { style: "currency", currency: "EUR" });
const time = new Intl.DateTimeFormat("en-GB", {
  day: "2-digit",
  month: "short",
  hour: "2-digit",
  minute: "2-digit",
  timeZone: "UTC",
});

export default function TrackedItemsPage({ items, loading, error, onRetry, onAdd, onViewDetail, onDelete }: Props) {
  const [search, setSearch] = useState("");
  const filteredItems = useMemo(() => {
    const query = search.trim().toLocaleLowerCase();
    if (!query) return items;
    return items.filter((item) =>
      [item.title ?? "", item.asin, item.marketplace].some((value) =>
        value.toLocaleLowerCase().includes(query),
      ),
    );
  }, [items, search]);

  return (
    <section className="tracked-page" aria-labelledby="tracked-heading">
      <div className="page-heading">
        <div>
          <p className="eyebrow">AMAZON EUROPE</p>
          <h1 id="tracked-heading">Tracked items</h1>
          <p className="page-description">
            {items.length} {items.length === 1 ? "item" : "items"} tracked · Prices checked twice daily
          </p>
        </div>
        <Button renderIcon={Add} onClick={onAdd}>Add item</Button>
      </div>

      {error && (
        <div className="page-feedback">
          <InlineNotification
            kind="error"
            title="Could not load tracked items"
            subtitle={error}
            lowContrast
            className="detail-notification"
          />
          <Button kind="tertiary" size="sm" onClick={onRetry}>Retry</Button>
        </div>
      )}

      {loading ? (
        <div className="empty-state"><InlineLoading description="Loading tracked items…" /></div>
      ) : error ? null : items.length === 0 ? (
        <div className="empty-state">
          <h2>No items tracked yet</h2>
          <p>Add an Amazon listing URL to start following its price.</p>
          <Button renderIcon={Add} onClick={onAdd}>Add your first item</Button>
        </div>
      ) : (
        <TableContainer className="tracked-table-container">
          <TableToolbar>
            <TableToolbarContent>
              <TextInput
                id="item-search"
                labelText="Search tracked items"
                placeholder="Search by title, ASIN, or marketplace"
                value={search}
                onChange={(event) => setSearch(event.target.value)}
              />
            </TableToolbarContent>
          </TableToolbar>
          <Table size="lg" aria-label="Tracked items">
            <TableHead>
              <TableRow>
                <TableHeader>Item</TableHeader>
                <TableHeader>Marketplace</TableHeader>
                <TableHeader>Latest price</TableHeader>
                <TableHeader>Last 3 detections</TableHeader>
                <TableHeader>Status</TableHeader>
                <TableHeader aria-label="Actions" />
              </TableRow>
            </TableHead>
            <TableBody>
              {filteredItems.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6}>
                    <p className="table-empty">No items match your search.</p>
                  </TableCell>
                </TableRow>
              ) : filteredItems.map((item) => (
                <TableRow key={item.id}>
                  <TableCell>
                    <div className="item-cell">
                      {item.thumbnailUrl ? (
                        <img className="item-thumbnail" src={item.thumbnailUrl} alt="" loading="lazy" />
                      ) : (
                        <div className="item-thumbnail item-thumbnail-placeholder" aria-hidden="true">—</div>
                      )}
                      <div className="item-identity">
                        <Button kind="ghost" className="item-title" onClick={() => onViewDetail(item)}>
                          {item.title ?? "Title unavailable"}
                        </Button>
                        <span className="item-asin">{item.asin}</span>
                      </div>
                    </div>
                  </TableCell>
                  <TableCell><Tag type="cool-gray" size="sm">{item.marketplace}</Tag></TableCell>
                  <TableCell>
                    {item.latestPrice ? (
                      <div className="latest-price">
                        <strong>{euro.format(item.latestPrice.amount)}</strong>
                        <time dateTime={item.latestPrice.timestamp}>{time.format(new Date(item.latestPrice.timestamp))} UTC</time>
                      </div>
                    ) : <span className="muted">Awaiting first price</span>}
                  </TableCell>
                  <TableCell>
                    {item.lastThreeDetections.length ? (
                      <ul className="price-history-list">
                        {item.lastThreeDetections.slice(0, 3).map((observation) => (
                          <li key={observation.timestamp}>
                            <span>{euro.format(observation.amount)}</span>
                            <time dateTime={observation.timestamp}>{time.format(new Date(observation.timestamp))}</time>
                          </li>
                        ))}
                      </ul>
                    ) : <span className="muted">No detections yet</span>}
                  </TableCell>
                  <TableCell><StatusTag status={item.status} /></TableCell>
                  <TableCell>
                    <div className="row-actions">
                      <Button
                        kind="ghost"
                        size="sm"
                        hasIconOnly
                        renderIcon={Launch}
                        iconDescription={`Open ${item.marketplace} listing in a new tab`}
                        onClick={() => window.open(item.url, "_blank", "noopener,noreferrer")}
                      />
                      <OverflowMenu flipped size="sm" aria-label={`Actions for ${item.title ?? item.asin}`}>
                        <OverflowMenuItem itemText="View details" onClick={() => onViewDetail(item)} />
                        <OverflowMenuItem
                          itemText="Open on Amazon"
                          onClick={() => window.open(item.url, "_blank", "noopener,noreferrer")}
                        />
                        <OverflowMenuItem itemText="Delete" isDelete hasDivider onClick={() => onDelete(item)} />
                      </OverflowMenu>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}
    </section>
  );
}
