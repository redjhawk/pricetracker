import { useCallback, useEffect, useState } from "react";
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
import { addItem as apiAddItem, deleteItem as apiDeleteItem, getItem, listItems } from "./api/items";
import AddItemModal from "./components/AddItemModal";
import DeleteModal from "./components/DeleteModal";
import ItemDetail from "./components/ItemDetail";
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
  const [listLoading, setListLoading] = useState(true);
  const [listError, setListError] = useState<string | null>(null);
  const [detailItem, setDetailItem] = useState<TrackedItem | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState<string | null>(null);
  const [pathname, setPathname] = useState(window.location.pathname);
  const [addOpen, setAddOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<TrackedItem | null>(null);
  const itemId = getItemId(pathname);

  const refreshItems = useCallback(async (quiet = false) => {
    try {
      const currentItems = await listItems();
      setItems(currentItems);
      setListError(null);
    } catch (error) {
      if (!quiet) setListError(errorMessage(error));
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
    if (!items.some((item) => item.status === "pending")) return;
    const poll = window.setInterval(() => void refreshItems(true), 5_000);
    return () => window.clearInterval(poll);
  }, [items, refreshItems]);

  useEffect(() => {
    if (!itemId) {
      setDetailItem(null);
      setDetailError(null);
      setDetailLoading(false);
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
    if (!itemId || detailItem?.status !== "pending") return;
    const poll = window.setInterval(() => {
      getItem(itemId)
        .then(setDetailItem)
        .catch((error) => setDetailError(errorMessage(error)));
    }, 5_000);
    return () => window.clearInterval(poll);
  }, [itemId, detailItem?.status]);

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
              />
            ) : null
          ) : (
            <TrackedItemsPage
              items={items}
              loading={listLoading}
              error={listError}
              onRetry={() => void refreshItems()}
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
    </>
  );
}
