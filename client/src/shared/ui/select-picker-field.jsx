import { Children, isValidElement, useMemo, useState } from "react";
import { AsyncSearchSelect } from "./async-search-select";

function getOptionText(children) {
  return Children.toArray(children)
    .map((child) => {
      if (typeof child === "string" || typeof child === "number") {
        return String(child);
      }

      if (isValidElement(child)) {
        return getOptionText(child.props.children);
      }

      return "";
    })
    .join("")
    .trim();
}

export function SelectField({
  children,
  disabled = false,
  name,
  onChange,
  value,
}) {
  const [searchValue, setSearchValue] = useState("");
  const optionItems = useMemo(
    () =>
      Children.toArray(children)
        .filter((child) => isValidElement(child) && child.type === "option")
        .map((child) => ({
          disabled: Boolean(child.props.disabled),
          id: String(child.props.value ?? ""),
          label: getOptionText(child.props.children),
        })),
    [children],
  );
  const emptyOption = optionItems.find((option) => option.id === "");
  const activeValue = String(value ?? "");
  const selectedOption = optionItems.find((option) => option.id === activeValue);
  const normalizedSearchValue = searchValue.trim().toLowerCase();
  const options = optionItems.filter((option) => {
    if (option.disabled || option.id === "") {
      return false;
    }

    if (!normalizedSearchValue) {
      return true;
    }

    return option.label.toLowerCase().includes(normalizedSearchValue);
  });

  function emitChange(nextValue) {
    onChange?.({
      target: {
        name,
        value: nextValue,
      },
    });
  }

  return (
    <AsyncSearchSelect
      allowClear={Boolean(emptyOption)}
      clearLabel={emptyOption?.label || "Очистить выбор"}
      disabled={disabled}
      emptyMessage="Ничего не найдено."
      getOptionLabel={(option) => option.label}
      onSearchChange={setSearchValue}
      onSelect={(option) => emitChange(option?.id || "")}
      options={options}
      placeholder={emptyOption?.label || "Выберите значение"}
      searchPlaceholder="Поиск по вариантам"
      selectedLabel={selectedOption?.label || ""}
      value={activeValue}
    />
  );
}
