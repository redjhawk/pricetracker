import { useMemo, useState } from "react";
import {
  Button,
  InlineLoading,
  InlineNotification,
  OverflowMenu,
  OverflowMenuItem,
  Tab,
  TabList,
  TabPanel,
  TabPanels,
  Tabs,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableHeader,
  TableRow,
  TextInput,
} from "@carbon/react";
import { Add, Launch, Renew } from "@carbon/icons-react";
import type { TrackedItem } from "../types";
import StatusTag from "./StatusTag";

interface Props {
  items: TrackedItem[];
  selectedPlatform: TrackedItem["platform"];
  onPlatformChange: (platform: TrackedItem["platform"]) => void;
  loading: boolean;
  error: string | null;
  refreshError: string | null;
  refreshing: boolean;
  refreshCount: number;
  onRetry: () => void;
  onRefresh: () => void;
  onAdd: () => void;
  onViewDetail: (item: TrackedItem) => void;
  onDelete: (item: TrackedItem) => void;
  onRefreshItem: (item: TrackedItem) => void;
  refreshingItemIds: ReadonlySet<string>;
  itemRefreshError: string | null;
}

const euro = new Intl.NumberFormat("de-DE", { style: "currency", currency: "EUR" });
const platforms = ["amazon", "leboncoin"] as const;
const priceLabel = (amount: number) => amount === 0 ? "Gratuit" : euro.format(amount);
const time = new Intl.DateTimeFormat("en-GB", {
  day: "2-digit",
  month: "short",
  hour: "2-digit",
  minute: "2-digit",
  timeZone: "UTC",
});

export default function TrackedItemsPage({ items, selectedPlatform, onPlatformChange, loading, error, refreshError, refreshing, refreshCount, onRetry, onRefresh, onAdd, onViewDetail, onDelete, onRefreshItem, refreshingItemIds, itemRefreshError }: Props) {
  const [search, setSearch] = useState("");
  const platformItems = useMemo(() => items.filter((item) => item.platform === selectedPlatform), [items, selectedPlatform]);
  const filteredItems = useMemo(() => {
    const query = search.trim().toLocaleLowerCase();
    if (!query) return platformItems;
    return platformItems.filter((item) =>
      [item.title ?? "", item.listingId, item.asin ?? "", item.marketplace, item.platform].some((value) =>
        value.toLocaleLowerCase().includes(query),
      ),
    );
  }, [platformItems, search]);

  return (
    <section className="tracked-page" aria-labelledby="tracked-heading">
      <div className="page-heading">
        <div>
          <p className="eyebrow">AMAZON · LEBONCOIN</p>
          <h1 id="tracked-heading">Tracked items</h1>
          <p className="page-description">
            {items.length} {items.length === 1 ? "item" : "items"} tracked · Prices checked twice daily
          </p>
        </div>
        <div className="page-heading-actions">
          <Button kind="tertiary" renderIcon={Renew} onClick={onRefresh} disabled={refreshing || items.length === 0}>
            Refresh all prices
          </Button>
          <Button renderIcon={Add} onClick={onAdd}>Add item</Button>
        </div>
      </div>

      {refreshing && <InlineLoading description={`Refreshing prices for ${refreshCount} ${refreshCount === 1 ? "item" : "items"}…`} />}
      {refreshError && (
        <InlineNotification
          kind="error"
          title="Could not refresh prices"
          subtitle={refreshError}
          lowContrast
          className="detail-notification"
        />
      )}
      {itemRefreshError && (
        <InlineNotification
          kind="error"
          title="Could not refresh price"
          subtitle={itemRefreshError}
          lowContrast
          className="detail-notification"
        />
      )}

      <Tabs
        selectedIndex={platforms.indexOf(selectedPlatform)}
        onChange={({ selectedIndex }) => onPlatformChange(platforms[selectedIndex])}
      >
        <TabList aria-label="Listing platforms">
          <Tab>Amazon</Tab>
          <Tab>LeBoncoin</Tab>
        </TabList>
        <TabPanels>
          {platforms.map((platform) => (
            <TabPanel key={platform}>
              {platform === selectedPlatform && <>
                {!loading && error && (
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
                    <p>Add an Amazon or LeBoncoin listing URL to start following its price.</p>
                    <Button renderIcon={Add} onClick={onAdd}>Add your first item</Button>
                  </div>
                ) : platformItems.length === 0 ? (
                  <div className="empty-state">
                    <h2>No {selectedPlatform === "amazon" ? "Amazon" : "LeBoncoin"} items tracked yet</h2>
                    <p>Add a listing URL to start following its price.</p>
                    <Button renderIcon={Add} onClick={onAdd}>Add item</Button>
                  </div>
                ) : (
                  <>
                    <div className="tracked-table-search">
                      <TextInput
                        id="item-search"
                        className="tracked-search"
                        labelText="Search tracked items"
                        placeholder="Search by title, listing ID, or platform"
                        value={search}
                        onChange={(event) => setSearch(event.target.value)}
                      />
                    </div>
                    <TableContainer className="tracked-table-container">
                      <Table size="lg" aria-label="Tracked items">
                        <TableHead>
                          <TableRow>
                            <TableHeader>Item</TableHeader>
                            <TableHeader>Prices</TableHeader>
                            {selectedPlatform === "amazon" && <TableHeader>Amazon second-hand offer</TableHeader>}
                            <TableHeader>Status</TableHeader>
                            <TableHeader aria-label="Actions" />
                          </TableRow>
                        </TableHead>
                        <TableBody>
                          {filteredItems.length === 0 ? (
                            <TableRow>
                              <TableCell colSpan={selectedPlatform === "amazon" ? 5 : 4}>
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
                                    <span className="item-asin">{item.marketplace} · {item.listingId}</span>
                                  </div>
                                </div>
                              </TableCell>
                              <TableCell>
                                {item.lastThreeDetections.length || item.latestPrice ? (
                                  <ul className="price-history-list">
                                    {(item.lastThreeDetections.length
                                      ? item.lastThreeDetections.slice(0, 3)
                                      : [item.latestPrice!]
                                    ).map((observation) => (
                                      <li key={observation.timestamp}>
                                        <span>{priceLabel(observation.amount)}</span>
                                        <time dateTime={observation.timestamp}>{time.format(new Date(observation.timestamp))}</time>
                                      </li>
                                    ))}
                                  </ul>
                                ) : <span className="muted">Awaiting first price</span>}
                              </TableCell>
                              {selectedPlatform === "amazon" && <TableCell>
                                {item.secondHandOffer?.status === "available" && item.secondHandOffer.latestDetection ? (
                                  <div className="latest-price">
                                    <strong>{priceLabel(item.secondHandOffer.latestDetection.amount)}</strong>
                                    <span>{item.secondHandOffer.latestDetection.conditionLabel}</span>
                                  </div>
                                ) : item.secondHandOffer?.status === "pending" ? (
                                  <span className="muted">Checking Amazon offers…</span>
                                ) : item.secondHandOffer?.status === "not_found" ? (
                                  <span className="muted">No Amazon offer</span>
                                ) : item.secondHandOffer?.latestDetection ? (
                                  <div className="latest-price">
                                    <strong>{priceLabel(item.secondHandOffer.latestDetection.amount)}</strong>
                                    <span>{item.secondHandOffer.latestDetection.conditionLabel} · Last detected</span>
                                    <span className="muted">Latest check failed</span>
                                  </div>
                                ) : item.secondHandOffer ? <span className="muted">Amazon offer check unavailable</span> : <span className="muted">—</span>}
                              </TableCell>}
                              <TableCell><StatusTag status={item.status} /></TableCell>
                              <TableCell>
                                <div className="row-actions">
                                  <Button
                                    kind="ghost"
                                    size="sm"
                                    hasIconOnly
                                    renderIcon={Launch}
                                    iconDescription={`Open ${item.platform} listing in a new tab`}
                                    onClick={() => window.open(item.url, "_blank", "noopener,noreferrer")}
                                  />
                                  <OverflowMenu flipped size="sm" aria-label={`Actions for ${item.title ?? item.listingId}`}>
                                    <OverflowMenuItem itemText="View details" onClick={() => onViewDetail(item)} />
                                    <OverflowMenuItem
                                      itemText={`Open on ${item.platform === "amazon" ? "Amazon" : "LeBoncoin"}`}
                                      onClick={() => window.open(item.url, "_blank", "noopener,noreferrer")}
                                    />
                                    <OverflowMenuItem
                                      itemText="Refresh price"
                                      disabled={item.status === "pending" || refreshingItemIds.has(item.id)}
                                      onClick={() => onRefreshItem(item)}
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
                  </>
                )}
              </>}
            </TabPanel>
          ))}
        </TabPanels>
      </Tabs>
    </section>
  );
}
