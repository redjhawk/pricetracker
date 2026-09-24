import { Tag } from "@carbon/react";
import type { ItemStatus } from "../types";

const STATUS_MAP: Record<
  ItemStatus,
  { type: "green" | "gray" | "warm-gray" | "red" | "purple"; label: string }
> = {
  active: { type: "green", label: "Active" },
  stale: { type: "warm-gray", label: "Stale" },
  retrieval_error: { type: "red", label: "Retrieval error" },
  unavailable: { type: "gray", label: "Unavailable" },
  pending: { type: "purple", label: "Pending" },
};

export default function StatusTag({ status }: { status: ItemStatus }) {
  const { type, label } = STATUS_MAP[status];
  return <Tag type={type} size="sm">{label}</Tag>;
}
