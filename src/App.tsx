import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import {
  Button,
  Content,
  Header,
  HeaderGlobalBar,
  HeaderName,
  InlineLoading,
  InlineNotification,
  Loading,
} from "@carbon/react";
import { Add, ChartLine } from "@carbon/icons-react";
import type { TrackedItem } from "./types";
import {
  addItem as apiAddItem,
  deleteItem as apiDeleteItem,
  getItem,
  listItems,
  refreshAllItems,
  refreshItem,
  requestAiReview,
} from "./api/items";
import { getSession, logout, type SessionState } from "./api/auth";
import { authRequiredEvent } from "./api/client";
import AddItemModal from "./components/AddItemModal";
import AdminPage from "./components/AdminPage";
import LoginPage from "./components/LoginPage";
import AppMenu from "./components/AppMenu";
import DeleteModal from "./components/DeleteModal";
import ItemDetail from "./components/ItemDetail";
import SettingsModal from "./components/SettingsModal";
import TrackedItemsPage, { type Tab } from "./components/TrackedItemsPage";
import AmazonSearchItems from "./components/AmazonSearchItems";

function getItemId(pathname: string) {
  const match = pathname.match(/^\/items\/([^/]+)\/?$/);
  return match ? decodeURIComponent(match[1]) : null;
}

function getSearchId(pathname: string) {
  const match = pathname.match(/^\/searches\/([^/]+)\/?$/);
  return match ? decodeURIComponent(match[1]) : null;
}

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : "The server could not be reached.";
}

// App loads the session state first, then shows the screen for the mode and the logged-in user.
export default function App() {
  const [session, setSession] = useState<SessionState | null>(null);
  const [sessionError, setSessionError] = useState<string | null>(null);

  const loadSession = useCallback(() => {
    getSession()
      .then((current) => { setSession(current); setSessionError(null); })
      .catch((error) => setSessionError(errorMessage(error)));
  }, []);

  useEffect(() => {
    loadSession();
    window.addEventListener(authRequiredEvent, loadSession);
    return () => window.removeEventListener(authRequiredEvent, loadSession);
  }, [loadSession]);

  async function handleLogout() {
    try {
      await logout();
    } finally {
      window.location.assign("/");
    }
  }

  if (sessionError) {
    return (
      <main id="main-content" className="page-shell page-feedback">
        <InlineNotification kind="error" title="Could not reach Price follower" subtitle={sessionError} lowContrast hideCloseButton />
        <Button kind="tertiary" onClick={loadSession}>Retry</Button>
      </main>
    );
  }
  if (!session) return <Loading description="Loading Price follower" />;
  if (session.user?.role === "admin") {
    return (
      <>
        <AppHeader>
          <AppMenu onLogout={() => void handleLogout()} />
        </AppHeader>
        <Content className="app-content">
          <main id="main-content" className="page-shell"><AdminPage /></main>
        </Content>
      </>
    );
  }
  if (!session.user && (session.mode === "protected" || window.location.pathname === "/login")) {
    return <LoginPage openMode={session.mode === "open"} />;
  }
  return <TrackerApp openMode={session.mode === "open"} onLogout={session.user ? () => void handleLogout() : undefined} />;
}

function AppHeader({ onHome, children }: { onHome?: () => void; children: ReactNode }) {
  return (
    <Header aria-label="Price follower">
      <HeaderName href="/" prefix="" onClick={(event) => {
        if (!onHome) return;
        event.preventDefault();
        onHome();
      }}>
        <span className="brand-mark"><ChartLine size={20} /> Price follower</span>
      </HeaderName>
      <HeaderGlobalBar>{children}</HeaderGlobalBar>
    </Header>
  );
}

function TrackerApp({ openMode, onLogout }: { openMode: boolean; onLogout?: () => void }) {
  const [items, setItems] = useState<TrackedItem[]>([]);
  const [selectedPlatform, setSelectedPlatform] = useState<Tab>("amazon");
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
  const [aiReviewRequesting, setAiReviewRequesting] = useState(false);
  const [aiReviewError, setAiReviewError] = useState<string | null>(null);
  const [refreshingItemIds, setRefreshingItemIds] = useState<ReadonlySet<string>>(new Set());
  const [itemRefreshError, setItemRefreshError] = useState<string | null>(null);
  const [pathname, setPathname] = useState(window.location.pathname);
  const [addOpen, setAddOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<TrackedItem | null>(null);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const menuButtonRef = useRef<HTMLButtonElement>(null);
  const itemId = getItemId(pathname);
  const searchId = getSearchId(pathname);

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
    setAiReviewRequesting(false);
    setAiReviewError(null);
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
    const aiReviewRunning = Boolean(detailItem?.aiReview?.running);
    if (!itemId || (detailItem?.status !== "pending" && !detailRefreshing && !aiReviewRunning)) return;
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
  }, [itemId, detailItem?.status, detailItem?.aiReview?.running, detailRefreshing]);

  function navigate(path: string) {
    window.history.pushState(null, "", path);
    setPathname(path);
  }

  async function handleAdd(url: string, purchaseGoal: string) {
    const item = await apiAddItem(url, purchaseGoal);
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

  async function handleAiReviewRefresh(id: string) {
    if (aiReviewRequesting) return;
    setAiReviewRequesting(true);
    setAiReviewError(null);
    try {
      await requestAiReview(id);
      setDetailItem(await getItem(id));
    } catch (error) {
      setAiReviewError(errorMessage(error));
    } finally {
      setAiReviewRequesting(false);
    }
  }

  async function handleRefreshListItem(item: TrackedItem) {
    setItemRefreshError(null);
    setRefreshingItemIds((current) => new Set(current).add(item.id));
    try {
      await refreshItem(item.id);
      await refreshItems(true);
    } catch (error) {
      setItemRefreshError(`${item.title ?? item.listingId}: ${errorMessage(error)}`);
    } finally {
      setRefreshingItemIds((current) => {
        const next = new Set(current);
        next.delete(item.id);
        return next;
      });
    }
  }

  const showingDetail = itemId !== null;

  return (
    <>
      <AppHeader onHome={() => navigate("/")}>
        {openMode && <Button kind="ghost" size="sm" href="/login">Log in</Button>}
        <Button kind="primary" size="sm" renderIcon={Add} onClick={() => setAddOpen(true)}>
          Add item
        </Button>
        <AppMenu onOpenSettings={() => setSettingsOpen(true)} onLogout={onLogout} triggerRef={menuButtonRef} />
      </AppHeader>

      <Content className="app-content">
        <main id="main-content" className="page-shell">
          {searchId ? (
            <AmazonSearchItems
              searchId={searchId}
              onBack={() => navigate("/")}
              onViewDetail={(item) => navigate(`/items/${encodeURIComponent(item.id)}`)}
              onTracked={() => void refreshItems(true)}
            />
          ) : showingDetail ? (
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
                onAiReviewRefresh={() => void handleAiReviewRefresh(detailItem.id)}
                aiReviewRequesting={aiReviewRequesting}
                aiReviewError={aiReviewError}
                onPurchaseGoalSaved={() => void getItem(detailItem.id).then(setDetailItem).catch((error) => setDetailError(errorMessage(error)))}
              />
            ) : null
          ) : (
            <TrackedItemsPage
              items={items}
              selectedPlatform={selectedPlatform}
              onPlatformChange={setSelectedPlatform}
              onOpenSearch={(search) => navigate(`/searches/${encodeURIComponent(search.id)}`)}
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
              onRefreshItem={(item) => void handleRefreshListItem(item)}
              refreshingItemIds={refreshingItemIds}
              itemRefreshError={itemRefreshError}
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
