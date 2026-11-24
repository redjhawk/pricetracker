import { useCallback, useEffect, useRef, useState } from "react";
import { listPurchaseGoals } from "./api/items";

export type GoalOption = { kind: "none" } | { kind: "existing"; goal: string } | { kind: "new" };
export interface GoalChoice { option: GoalOption; text: string }
export type GoalsState = { status: "loading" } | { status: "ready"; goals: string[] } | { status: "failed" };

export const noGoalChoice: GoalChoice = { option: { kind: "none" }, text: "" };

// goalLabel shows a goal by its first four words, on one line.
export function goalLabel(goal: string): string {
  const words = goal.trim().split(/\s+/);
  return words.length > 4 ? `${words.slice(0, 4).join(" ")}…` : words.join(" ");
}

// choiceFor selects a stored goal: "No goal" when empty, otherwise the goal itself.
export function choiceFor(goal: string): GoalChoice {
  return goal === "" ? noGoalChoice : { option: { kind: "existing", goal }, text: goal };
}

// isChoiceMode reports whether the drop-down is offered: goals loaded and at least one exists.
export function isChoiceMode(goals: GoalsState): goals is { status: "ready"; goals: string[] } {
  return goals.status === "ready" && goals.goals.length > 0;
}

// goalToSave is the goal text sent for the choice; the server trims it.
export function goalToSave(goals: GoalsState, choice: GoalChoice): string {
  if (goals.status === "loading") return "";
  if (isChoiceMode(goals) && choice.option.kind === "none") return "";
  return choice.text;
}

// compareGoals is the server order: letter case ignored, then the exact text.
function compareGoals(a: string, b: string): number {
  const x = a.toLowerCase();
  const y = b.toLowerCase();
  if (x !== y) return x < y ? -1 : 1;
  return a < b ? -1 : a > b ? 1 : 0;
}

// withGoal adds the item's own goal to the list when it is missing (stale list).
export function withGoal(goals: string[], goal: string): string[] {
  if (goal === "" || goals.includes(goal)) return goals;
  return [...goals, goal].sort(compareGoals);
}

// usePurchaseGoals loads the existing goals each time active becomes true.
export function usePurchaseGoals(active: boolean): { state: GoalsState; reload: () => void } {
  const [state, setState] = useState<GoalsState>({ status: "loading" });
  const latest = useRef(0);

  useEffect(() => {
    if (!active) return;
    const request = ++latest.current;
    setState({ status: "loading" });
    listPurchaseGoals().then(
      (goals) => { if (latest.current === request) setState({ status: "ready", goals }); },
      () => { if (latest.current === request) setState({ status: "failed" }); },
    );
    return () => { latest.current++; };
  }, [active]);

  const reload = useCallback(() => {
    const request = ++latest.current;
    listPurchaseGoals().then(
      (goals) => { if (latest.current === request) setState({ status: "ready", goals }); },
      () => {},
    );
  }, []);

  return { state, reload };
}
