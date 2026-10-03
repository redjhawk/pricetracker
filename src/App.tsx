import { useCallback, useEffect, useRef, useState } from "react";
import {
  Button,
  Content,
  Header,
  HeaderGlobalBar,
  HeaderName,
  InlineLoading,
  InlineNotification,
} from "@carbon/react";
import { Add, ChartLine } from "@carbon/icons-react";
import type { TrackedItem } from "./types";
import { addItem as apiAddItem, deleteItem as apiDeleteItem, getItem, listItems, refreshAllItems, refreshItem } from "./api/items";
import AddItemModal from "./components/AddItemModal";
import AppMenu from "./components/AppMenu";
import DeleteModal from "./components/DeleteModal";
import ItemDetail from "./components/ItemDetail";
import SettingsModal from "./components/SettingsModal";
import TrackedItemsPage from "./components/TrackedItemsPage";

function getItemId(pathname: string) {
  const match = pathname.match(/^\/items\/([^/]+)\/?$/);
  return match ? decodeURIComponent(match[1]) : null;
}

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : "The server could not be reached.";
}

export default function App() {
  const [items, setItems] = useState<TrackedItem[]>([]);
  const [selectedPlatform, setSelectedPlatform] = useState<TrackedItem["platform"]>("amazon");
  const [listLoading, setListLoading] = useState(true);
  const [listError, setListError] = useState<string | null>(null);
  const [refreshing, setRefreshing] = useState(false);
  const [refreshCount, setRefreshCount] = useState(0);
  const [refreshError, setRefreshError] = useState<string | null>(null);
  const [detailItem, setDetailItem] = useState<TrackedItem | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState<string | null>(null);
  const [detailRefreshing, setDetailRefreshing] = useState(false);
  const [detailRefreshError, setDetailRefreshError] = useState<string | null>(null);
  const [pathname, setPathname] = useState(window.location.pathname);
  const [addOpen, setAddOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<TrackedItem | null>(null);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const menuButtonRef = useRef<HTMLButtonElement>(null);
  const itemId = getItemId(pathname);

  const refreshItems = useCallback(async (quiet = false): Promise<TrackedItem[] | null> => {
    try {
      const currentItems = await listItems();
      setItems(currentItems);
      setListError(null);
      return currentItems;
    } catch (error) {
      if (!quiet) setListError(errorMessage(error));
      return null;
    } finally {
      if (!quiet) setListLoading(false);
    }
  }, []);

  useEffect(() => {
    const onPopState = () => setPathname(window.location.pathname);
    window.addEventListener("popstate", onPopState);
    return () => window.removeEventListener("popstate", onPopState);
  }, []);

  useEffect(() => {
    void refreshItems();
  }, [refreshItems]);

  useEffect(() => {
    if (!refreshing && !items.some((item) => item.status === "pending")) return;
    const poll = window.setInterval(async () => {
      const currentItems = await refreshItems(true);
      if (refreshing && currentItems && !currentItems.some((item) => item.status === "pending")) {
        setRefreshing(false);
      }
    }, 5_000);
    return () => window.clearInterval(poll);
  }, [items, refreshItems, refreshing]);

  useEffect(() => {
    if (!itemId) {
      setDetailItem(null);
      setDetailError(null);
      setDetailLoading(false);
      setDetailRefreshing(false);
      setDetailRefreshError(null);
      return;
    }

    let active = true;
    setDetailLoading(true);
    setDetailError(null);
    getItem(itemId)
      .then((item) => { if (active) setDetailItem(item); })
      .catch((error) => { if (active) setDetailError(errorMessage(error)); })
      .finally(() => { if (active) setDetailLoading(false); });
    return () => { active = false; };
  }, [itemId]);

  useEffect(() => {
    if (!itemId || (detailItem?.status !== "pending" && !detailRefreshing)) return;
    let active = true;
    const poll = window.setInterval(() => {
      getItem(itemId)
        .then((item) => {
          if (!active) return;
          setDetailItem(item);
          if (item.status !== "pending") setDetailRefreshing(false);
        })
        .catch((error) => {
          if (active) setDetailError(errorMessage(error));
        });
    }, 3_000);
    return () => { active = false; window.clearInterval(poll); };
  }, [itemId, detailItem?.status, detailRefreshing]);

  function navigate(path: string) {
    window.history.pushState(null, "", path);
    setPathname(path);
  }

  async function handleAdd(url: string) {
    const item = await apiAddItem(url);
    setItems((current) => [item, ...current.filter((existing) => existing.id !== item.id)]);
    setListError(null);
  }

  async function handleDelete(id: string) {
    await apiDeleteItem(id);
    setItems((current) => current.filter((item) => item.id !== id));
    if (itemId === id) navigate("/");
    setDeleteTarget(null);
  }

  async function handleRefresh() {
    setRefreshError(null);
    try {
      const result = await refreshAllItems();
      setRefreshCount(result.itemsQueued);
      setRefreshing(result.itemsQueued > 0);
      const updatedItems = await refreshItems(true);
      if (updatedItems) setRefreshing(result.itemsQueued > 0 && updatedItems.some((item) => item.status === "pending"));
    } catch (error) {
      setRefreshError(errorMessage(error));
      setRefreshing(false);
    }
  }

  async function handleRefreshItem(id: string) {
    setDetailRefreshing(true);
    setDetailRefreshError(null);
    try {
      await refreshItem(id);
      const updated = await getItem(id);
      setDetailItem(updated);
      if (updated.status !== "pending") setDetailRefreshing(false);
    } catch (error) {
      setDetailRefreshing(false);
      setDetailRefreshError(errorMessage(error));
    }
  }

  const showingDetail = itemId !== null;

  return (
    <>
      <Header aria-label="Price follower">
        <HeaderName href="/" prefix="" onClick={(event) => {
          event.preventDefault();
          navigate("/");
        }}>
          <span className="brand-mark"><ChartLine size={20} /> Price follower</span>
        </HeaderName>
        <HeaderGlobalBar>
          <Button kind="primary" size="sm" renderIcon={Add} onClick={() => setAddOpen(true)}>
            Add item
          </Button>
          <AppMenu onOpenSettings={() => setSettingsOpen(true)} triggerRef={menuButtonRef} />
        </HeaderGlobalBar>
      </Header>

      <Content className="app-content">
        <main id="main-content" className="page-shell">
          {showingDetail ? (
            detailLoading ? (
              <InlineLoading description="Loading item details…" />
            ) : detailError ? (
              <div className="page-feedback">
                <InlineNotification kind="error" title="Could not load item" subtitle={detailError} lowContrast />
                <Button kind="tertiary" onClick={() => navigate("/")}>Back to tracked items</Button>
              </div>
            ) : detailItem ? (
              <ItemDetail
                item={detailItem}
                onBack={() => navigate("/")}
                onDelete={setDeleteTarget}
                onRefresh={() => void handleRefreshItem(detailItem.id)}
                refreshing={detailRefreshing}
                refreshError={detailRefreshError}
              />
            ) : null
          ) : (
            <TrackedItemsPage
              items={items}
              selectedPlatform={selectedPlatform}
              onPlatformChange={setSelectedPlatform}
              loading={listLoading}
              error={listError}
              refreshError={refreshError}
              refreshing={refreshing}
              refreshCount={refreshCount}
              onRetry={() => void refreshItems()}
              onRefresh={() => void handleRefresh()}
              onAdd={() => setAddOpen(true)}
              onViewDetail={(item) => navigate(`/items/${encodeURIComponent(item.id)}`)}
              onDelete={setDeleteTarget}
            />
          )}
        </main>
      </Content>

      <AddItemModal
        open={addOpen}
        onClose={() => setAddOpen(false)}
        onAdded={handleAdd}
      />
      <DeleteModal
        item={deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete}
      />
      <SettingsModal
        open={settingsOpen}
        onClose={() => setSettingsOpen(false)}
        launcherButtonRef={menuButtonRef}
      />
    </>
  );
}
