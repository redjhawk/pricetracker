import { useMemo } from "react";
import { Dropdown, InlineLoading, TextArea } from "@carbon/react";
import { goalLabel, isChoiceMode, type GoalChoice, type GoalOption, type GoalsState } from "../purchaseGoalChoice";

interface Props {
  id: string;
  label: string;
  helperText: string;
  goals: GoalsState;
  value: GoalChoice;
  onChange: (value: GoalChoice) => void;
  disabled: boolean;
}

// Downshift selection types: string values in development builds, numbers in production builds.
// Downshift is not a direct dependency, so its constants are not imported.
const itemClick = new Set<unknown>(["__item_click__", 14]);
const keyboardSelection = new Set<unknown>(["__togglebutton_keydown_enter__", 7, "__togglebutton_keydown_space_button__", 8]);

function optionLabel(option: GoalOption | null): string {
  if (!option) return "";
  if (option.kind === "none") return "No goal";
  if (option.kind === "new") return "New goal";
  return goalLabel(option.goal);
}

function sameOption(a: GoalOption, b: GoalOption): boolean {
  if (a.kind === "existing" && b.kind === "existing") return a.goal === b.goal;
  return a.kind === b.kind;
}

export default function PurchaseGoalChoice({ id, label, helperText, goals, value, onChange, disabled }: Props) {
  const list = isChoiceMode(goals) ? goals.goals : null;
  const options = useMemo<GoalOption[]>(
    () => list ? [{ kind: "none" }, ...list.map((goal) => ({ kind: "existing" as const, goal })), { kind: "new" }] : [],
    [list],
  );

  if (goals.status === "loading") return <InlineLoading description="Loading your purchase goals…" />;

  if (!list) {
    return (
      <TextArea
        id={id}
        labelText={label}
        helperText={helperText}
        rows={3}
        value={value.text}
        onChange={(event) => onChange({ ...value, text: event.target.value })}
        disabled={disabled}
      />
    );
  }

  const selected = options.find((option) => sameOption(option, value.option)) ?? options[0];
  // Selecting an option replaces the text: an existing goal puts its full text, the others empty it.
  const select = (option: GoalOption) => onChange({ option, text: option.kind === "existing" ? option.goal : "" });

  return (
    <div className="purchase-goal-choice">
      <Dropdown<GoalOption>
        id={`${id}-choice`}
        titleText={label}
        helperText={helperText}
        label="No goal"
        autoAlign
        items={options}
        itemToString={optionLabel}
        selectedItem={selected}
        onChange={({ selectedItem }) => { if (selectedItem) select(selectedItem); }}
        downshiftProps={{
          // Downshift reports no change when the selected option is chosen again; apply it anyway.
          onStateChange: (changes) => {
            if ("selectedItem" in changes || changes.isOpen !== false) return;
            if (itemClick.has(changes.type) || (keyboardSelection.has(changes.type) && "highlightedIndex" in changes)) select(selected);
          },
        }}
        disabled={disabled}
      />
      {value.option.kind !== "none" && (
        <TextArea
          id={id}
          labelText="Goal text"
          helperText="Changes apply to this item only."
          rows={3}
          value={value.text}
          onChange={(event) => onChange({ ...value, text: event.target.value })}
          disabled={disabled}
        />
      )}
    </div>
  );
}
